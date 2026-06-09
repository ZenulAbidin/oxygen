package paymentevents

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/oxygenpay/oxygen/internal/bus"
	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/service/merchant"
	"github.com/oxygenpay/oxygen/internal/service/payment"
	"github.com/oxygenpay/oxygen/internal/service/processing"
	"github.com/oxygenpay/oxygen/internal/service/transaction"
	"github.com/oxygenpay/oxygen/internal/slack"
	"github.com/oxygenpay/oxygen/internal/util"
	"github.com/oxygenpay/oxygen/internal/webhook"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type Handler struct {
	merchants       *merchant.Service
	processing      *processing.Service
	payments        *payment.Service
	slackWebhookURL string
	logger          *zerolog.Logger
}

func New(
	merchants *merchant.Service,
	processingService *processing.Service,
	payments *payment.Service,
	slackWebhookURL string,
	logger *zerolog.Logger,
) *Handler {
	log := logger.With().Str("channel", "payment_events_consumer").Logger()

	return &Handler{
		merchants:       merchants,
		processing:      processingService,
		payments:        payments,
		slackWebhookURL: slackWebhookURL,
		logger:          &log,
	}
}

func (h *Handler) Consumers() map[bus.Topic][]bus.Consumer {
	return map[bus.Topic][]bus.Consumer{
		bus.TopicPaymentStatusUpdate: {
			h.ProcessPaymentStatusUpdate,
			h.SendSuccessfulPaymentNotification,
		},
		bus.TopicWithdrawals: {h.ProcessWithdrawals},
	}
}

type PaymentWebhook struct {
	EventID   string `json:"eventId"`
	EventType string `json:"eventType"`
	Version   string `json:"version"`

	ID      string  `json:"id"`
	OrderID *string `json:"orderId,omitempty"`
	Type    string  `json:"type"`
	Status  string  `json:"status"`

	CustomerEmail string `json:"customerEmail"`

	SelectedBlockchain string `json:"selectedBlockchain"`
	SelectedCurrency   string `json:"selectedCurrency"`

	IsTest bool `json:"isTest"`

	LinkID *string `json:"paymentLinkId"`

	OccurredAt time.Time      `json:"occurredAt"`
	Payment    WebhookPayment `json:"payment"`

	Customer      *WebhookCustomer      `json:"customer,omitempty"`
	PaymentMethod *WebhookPaymentMethod `json:"paymentMethod,omitempty"`
	PaymentLink   *WebhookPaymentLink   `json:"paymentLink,omitempty"`
	Transaction   *WebhookTransaction   `json:"transaction,omitempty"`
}

type WebhookPayment struct {
	ID          string       `json:"id"`
	PublicID    string       `json:"publicId"`
	OrderID     *string      `json:"orderId,omitempty"`
	Type        string       `json:"type"`
	Status      string       `json:"status"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	ExpiresAt   *time.Time   `json:"expiresAt,omitempty"`
	Amount      WebhookMoney `json:"amount"`
	RedirectURL string       `json:"redirectUrl,omitempty"`
	PaymentURL  string       `json:"paymentUrl,omitempty"`
	Description *string      `json:"description,omitempty"`
	IsTest      bool         `json:"isTest"`
}

type WebhookCustomer struct {
	Email string `json:"email"`
}

type WebhookPaymentMethod struct {
	Blockchain     string `json:"blockchain"`
	BlockchainName string `json:"blockchainName"`
	Currency       string `json:"currency"`
	Name           string `json:"name"`
	DisplayName    string `json:"displayName"`
	NetworkID      string `json:"networkId"`
	IsTest         bool   `json:"isTest"`
}

type WebhookPaymentLink struct {
	ID string `json:"id"`
}

type WebhookTransaction struct {
	Type             string        `json:"type"`
	Status           string        `json:"status"`
	Hash             *string       `json:"hash,omitempty"`
	ExplorerLink     *string       `json:"explorerLink,omitempty"`
	NetworkID        string        `json:"networkId"`
	SenderAddress    *string       `json:"senderAddress,omitempty"`
	RecipientAddress string        `json:"recipientAddress"`
	Amount           WebhookMoney  `json:"amount"`
	FactAmount       *WebhookMoney `json:"factAmount,omitempty"`
	ServiceFee       WebhookMoney  `json:"serviceFee"`
	NetworkFee       *WebhookMoney `json:"networkFee,omitempty"`
	CreatedAt        time.Time     `json:"createdAt"`
	UpdatedAt        time.Time     `json:"updatedAt"`
	IsTest           bool          `json:"isTest"`
}

type WebhookMoney struct {
	Value    string `json:"value"`
	Raw      string `json:"raw"`
	Currency string `json:"currency"`
	Decimals int64  `json:"decimals"`
}

func (w PaymentWebhook) WebhookEventID() string {
	return w.EventID
}

func (w PaymentWebhook) WebhookEventType() string {
	return w.EventType
}

func (h *Handler) ProcessPaymentStatusUpdate(ctx context.Context, message bus.Message) error {
	req, err := bus.Bind[bus.PaymentStatusUpdateEvent](message)
	if err != nil {
		return err
	}

	mt, err := h.merchants.GetByID(ctx, req.MerchantID, false)
	if err != nil {
		return errors.Wrap(err, "unable to get merchant")
	}

	webhookURL := mt.Settings().WebhookURL()
	signatureSecret := mt.Settings().WebhookSignatureSecret()

	if webhookURL == "" {
		h.logger.Warn().
			Int64("merchant_id", req.MerchantID).Int64("payment_id", req.PaymentID).
			Msg("webhook not set; skipping sending")

		return nil
	}

	p, err := h.processing.GetDetailedPayment(ctx, req.MerchantID, req.PaymentID)
	if err != nil {
		return errors.Wrap(err, "unable to get detailed payment")
	}

	// omit "locked" event
	if p.Payment.Status == payment.StatusLocked {
		return nil
	}

	wh, err := h.buildPaymentWebhook(ctx, mt, p)
	if err != nil {
		return err
	}

	if err := webhook.Send(ctx, webhookURL, signatureSecret, wh); err != nil {
		h.logger.Warn().Err(err).
			Int64("merchant_id", req.MerchantID).
			Int64("payment_id", req.PaymentID).
			Interface("webhook", wh).
			Str("webhook_url", webhookURL).
			Msg("unable to send webhook")

		// todo some visual alert for merchant

		return nil
	}

	if err := h.payments.SetWebhookTimestamp(ctx, req.MerchantID, req.PaymentID, time.Now()); err != nil {
		return errors.Wrap(err, "unable to set webhook timestamp")
	}

	h.logger.Info().
		Int64("merchant_id", req.MerchantID).
		Int64("payment_id", req.PaymentID).
		Str("webhook_url", webhookURL).
		Msg("sent webhook to merchant")

	return nil
}

func (h *Handler) buildPaymentWebhook(
	ctx context.Context,
	mt *merchant.Merchant,
	details *processing.DetailedPayment,
) (PaymentWebhook, error) {
	pt := details.Payment
	wh := PaymentWebhook{
		EventID:   paymentWebhookEventID(pt),
		EventType: fmt.Sprintf("%s.status_changed", pt.Type),
		Version:   "1",

		ID:      pt.MerchantOrderUUID.String(),
		OrderID: pt.MerchantOrderID,
		Type:    pt.Type.String(),
		Status:  pt.Status.String(),
		IsTest:  pt.IsTest,

		OccurredAt: pt.UpdatedAt,
		Payment: WebhookPayment{
			ID:          pt.MerchantOrderUUID.String(),
			PublicID:    pt.PublicID.String(),
			OrderID:     pt.MerchantOrderID,
			Type:        pt.Type.String(),
			Status:      pt.Status.String(),
			CreatedAt:   pt.CreatedAt,
			UpdatedAt:   pt.UpdatedAt,
			ExpiresAt:   pt.ExpiresAt,
			Amount:      moneyToWebhook(pt.Price),
			RedirectURL: pt.RedirectURL,
			PaymentURL:  pt.PaymentURL,
			Description: pt.Description,
			IsTest:      pt.IsTest,
		},
	}

	if details.Customer != nil {
		wh.CustomerEmail = details.Customer.Email
		wh.Customer = &WebhookCustomer{Email: details.Customer.Email}
	}

	if details.PaymentMethod != nil {
		currency := details.PaymentMethod.Currency
		wh.SelectedBlockchain = currency.Blockchain.String()
		wh.SelectedCurrency = currency.Ticker
		wh.PaymentMethod = &WebhookPaymentMethod{
			Blockchain:     currency.Blockchain.String(),
			BlockchainName: currency.BlockchainName,
			Currency:       currency.Ticker,
			Name:           currency.Name,
			DisplayName:    currency.DisplayName(),
			NetworkID:      details.PaymentMethod.NetworkID,
			IsTest:         details.PaymentMethod.IsTest,
		}

		if tx := details.PaymentMethod.TX(); tx != nil {
			wh.Transaction = transactionToWebhook(tx)
		}
	}

	if pt.LinkID() != 0 {
		link, err := h.payments.GetPaymentLinkByID(ctx, mt.ID, pt.LinkID())
		if err != nil {
			return PaymentWebhook{}, errors.Wrap(err, "unable to get payment link")
		}

		wh.LinkID = util.Ptr(link.PublicID.String())
		wh.PaymentLink = &WebhookPaymentLink{ID: link.PublicID.String()}
	}

	return wh, nil
}

func paymentWebhookEventID(pt *payment.Payment) string {
	return fmt.Sprintf("%s:%s:%s", pt.Type.String(), pt.MerchantOrderUUID.String(), pt.Status.String())
}

func transactionToWebhook(tx *transaction.Transaction) *WebhookTransaction {
	wh := &WebhookTransaction{
		Type:             string(tx.Type),
		Status:           string(tx.Status),
		Hash:             tx.HashID,
		NetworkID:        tx.NetworkID(),
		SenderAddress:    tx.SenderAddress,
		RecipientAddress: tx.RecipientAddress,
		Amount:           moneyToWebhook(tx.Amount),
		ServiceFee:       moneyToWebhook(tx.ServiceFee),
		CreatedAt:        tx.CreatedAt,
		UpdatedAt:        tx.UpdatedAt,
		IsTest:           tx.IsTest,
	}

	if tx.FactAmount != nil {
		amount := moneyToWebhook(*tx.FactAmount)
		wh.FactAmount = &amount
	}
	if tx.NetworkFee != nil {
		amount := moneyToWebhook(*tx.NetworkFee)
		wh.NetworkFee = &amount
	}
	if link, err := tx.ExplorerLink(); err == nil && link != "" {
		wh.ExplorerLink = &link
	}

	return wh
}

func moneyToWebhook(m money.Money) WebhookMoney {
	return WebhookMoney{
		Value:    m.String(),
		Raw:      m.StringRaw(),
		Currency: m.Ticker(),
		Decimals: m.Decimals(),
	}
}

func (h *Handler) ProcessWithdrawals(ctx context.Context, message bus.Message) error {
	req, err := bus.Bind[bus.WithdrawalCreatedEvent](message)
	if err != nil {
		return err
	}

	h.logger.Info().
		Int64("merchant_id", req.MerchantID).
		Int64("payment_id", req.PaymentID).
		Msg("incoming withdrawal request")

	_, err = h.processing.BatchCreateWithdrawals(ctx, []int64{req.PaymentID})
	if err != nil {
		return errors.Wrap(err, "unable to process withdrawal creation")
	}

	return nil
}

func (h *Handler) SendSuccessfulPaymentNotification(ctx context.Context, message bus.Message) error {
	req, err := bus.Bind[bus.PaymentStatusUpdateEvent](message)
	if err != nil {
		return err
	}

	p, err := h.processing.GetDetailedPayment(ctx, req.MerchantID, req.PaymentID)
	if err != nil {
		return errors.Wrap(err, "unable to get detailed payment")
	}

	if p.Payment.Status != payment.StatusSuccess {
		// skip
		return nil
	}

	content := fmt.Sprintf(
		"[%s] 💰processed payment #%d: %s %s for merchant %q id#%d (isTest=%t) ",
		extractHost(p.Payment.PaymentURL),
		p.Payment.ID,
		p.Payment.Price.String(),
		p.Payment.Price.Ticker(),
		p.Merchant.Name,
		p.Merchant.ID,
		p.Payment.IsTest,
	)

	return h.sendSlackMessage(content)
}

func (h *Handler) sendSlackMessage(messages ...string) error {
	if h.slackWebhookURL == "" {
		h.logger.Warn().Msg("Skipping slack notification send due to empty webhook url")
		return nil
	}

	return slack.SendWebhook(h.slackWebhookURL, messages...)
}

func extractHost(u string) string {
	link, _ := url.Parse(u)

	return link.Host
}
