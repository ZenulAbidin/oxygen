package paymentevents_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/oxygenpay/oxygen/internal/bus"
	"github.com/oxygenpay/oxygen/internal/db/repository"
	"github.com/oxygenpay/oxygen/internal/event/paymentevents"
	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/service/merchant"
	"github.com/oxygenpay/oxygen/internal/service/payment"
	"github.com/oxygenpay/oxygen/internal/service/transaction"
	"github.com/oxygenpay/oxygen/internal/service/wallet"
	"github.com/oxygenpay/oxygen/internal/test"
	"github.com/oxygenpay/oxygen/internal/util"
	outboundwebhook "github.com/oxygenpay/oxygen/internal/webhook"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) (*test.IntegrationTest, *paymentevents.Handler, *[]string) {
	tc := test.NewIntegrationTest(t)

	var responses []string

	okResponder := func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		responses = append(responses, string(raw))

		writer.WriteHeader(http.StatusOK)
	}

	handler := paymentevents.New(
		tc.Services.Merchants,
		tc.Services.Processing,
		tc.Services.Payment,
		httptest.NewServer(http.HandlerFunc(okResponder)).URL,
		tc.Logger,
	)

	return tc, handler, &responses
}

func TestHandler_ProcessPaymentStatusUpdate(t *testing.T) {
	tc, handler, responses := setup(t)

	const merchantID = 1

	// ARRANGE
	// Given a mocked merchant server
	var actualWebhook paymentevents.PaymentWebhook
	var actualEventIDHeader string
	var actualEventTypeHeader string
	srv := assertServer(t, func(t *testing.T, writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)

		raw, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &actualWebhook))
		assert.True(t, outboundwebhook.ValidateHMAC(raw, "abc", request.Header.Get(outboundwebhook.HeaderSignature)))
		actualEventIDHeader = request.Header.Get(outboundwebhook.HeaderEventID)
		actualEventTypeHeader = request.Header.Get(outboundwebhook.HeaderEventType)
	})

	// And a merchant
	mt, err := tc.Services.Merchants.Create(tc.Context, merchantID, "my-site", "my-site.com", merchant.Settings{
		merchant.PropertySignatureSecret: "abc",
		merchant.PropertyWebhookURL:      srv.URL,
	})

	require.NoError(t, err)
	require.Equal(t, "abc", mt.Settings().WebhookSignatureSecret())
	require.Equal(t, srv.URL, mt.Settings().WebhookURL())

	// ... and a payment make from a link
	// This payment represents the most extended webhook case with payment link
	link, err := tc.Services.Payment.CreatePaymentLink(tc.Context, mt.ID, payment.CreateLinkProps{
		Name:          "test link",
		Price:         lo.Must(money.USD.MakeAmount("5000")),
		SuccessAction: payment.SuccessActionRedirect,
		RedirectURL:   util.Ptr("https://site.com"),
	})
	require.NoError(t, err)

	p, err := tc.Services.Payment.CreatePaymentFromLink(tc.Context, link)
	require.NoError(t, err)

	// And a transaction
	tx := tc.Must.CreateTransaction(t, merchantID, func(p *transaction.CreateTransaction) {
		p.RecipientWallet = tc.Must.CreateWallet(t, "ETH", "0x123", "pub-key", wallet.TypeInbound)
	})

	// And a customer
	person, err := tc.Services.Payment.AssignCustomerByEmail(tc.Context, p, "test@me.com")
	require.NoError(t, err)

	// And the payment mocked as "successful"
	_, err = tc.Repository.UpdatePayment(tc.Context, repository.UpdatePaymentParams{
		ID:         p.ID,
		MerchantID: p.MerchantID,
		Status:     payment.StatusSuccess.String(),
		UpdatedAt:  time.Now(),
	})
	require.NoError(t, err)

	// ACT
	msg := marshal(bus.PaymentStatusUpdateEvent{
		MerchantID: p.MerchantID,
		PaymentID:  p.ID,
	})

	assert.NoError(t, handler.ProcessPaymentStatusUpdate(tc.Context, msg))
	assert.NoError(t, handler.SendSuccessfulPaymentNotification(tc.Context, msg))

	// ASSERT
	assert.NotEmpty(t, actualWebhook.EventID)
	assert.Equal(t, actualWebhook.EventID, actualEventIDHeader)
	assert.Equal(t, "payment.status_changed", actualWebhook.EventType)
	assert.Equal(t, actualWebhook.EventType, actualEventTypeHeader)
	assert.Equal(t, "1", actualWebhook.Version)
	assert.NotZero(t, actualWebhook.OccurredAt)

	assert.Equal(t, p.MerchantOrderUUID.String(), actualWebhook.ID)
	assert.Equal(t, payment.TypePayment.String(), actualWebhook.Type)
	assert.Equal(t, payment.StatusSuccess.String(), actualWebhook.Status)
	assert.Equal(t, person.Email, actualWebhook.CustomerEmail)
	assert.Equal(t, tx.Currency.Blockchain.String(), actualWebhook.SelectedBlockchain)
	assert.Equal(t, tx.Currency.Ticker, actualWebhook.SelectedCurrency)
	assert.Equal(t, util.Ptr(link.PublicID.String()), actualWebhook.LinkID)
	assert.Equal(t, p.IsTest, actualWebhook.IsTest)

	assert.Equal(t, p.MerchantOrderUUID.String(), actualWebhook.Payment.ID)
	assert.Equal(t, p.PublicID.String(), actualWebhook.Payment.PublicID)
	assert.Equal(t, payment.TypePayment.String(), actualWebhook.Payment.Type)
	assert.Equal(t, payment.StatusSuccess.String(), actualWebhook.Payment.Status)
	assert.Equal(t, "50", actualWebhook.Payment.Amount.Value)
	assert.Equal(t, "5000", actualWebhook.Payment.Amount.Raw)
	assert.Equal(t, "USD", actualWebhook.Payment.Amount.Currency)
	assert.Equal(t, int64(2), actualWebhook.Payment.Amount.Decimals)

	require.NotNil(t, actualWebhook.Customer)
	assert.Equal(t, person.Email, actualWebhook.Customer.Email)

	require.NotNil(t, actualWebhook.PaymentMethod)
	assert.Equal(t, tx.Currency.Blockchain.String(), actualWebhook.PaymentMethod.Blockchain)
	assert.Equal(t, tx.Currency.Ticker, actualWebhook.PaymentMethod.Currency)
	assert.Equal(t, tx.NetworkID(), actualWebhook.PaymentMethod.NetworkID)

	require.NotNil(t, actualWebhook.PaymentLink)
	assert.Equal(t, link.PublicID.String(), actualWebhook.PaymentLink.ID)

	require.NotNil(t, actualWebhook.Transaction)
	assert.Equal(t, string(transaction.TypeIncoming), actualWebhook.Transaction.Type)
	assert.Equal(t, string(transaction.StatusPending), actualWebhook.Transaction.Status)
	assert.Equal(t, tx.RecipientAddress, actualWebhook.Transaction.RecipientAddress)
	assert.Equal(t, tx.Amount.String(), actualWebhook.Transaction.Amount.Value)
	assert.Equal(t, tx.Amount.StringRaw(), actualWebhook.Transaction.Amount.Raw)
	assert.Equal(t, tx.Currency.Ticker, actualWebhook.Transaction.Amount.Currency)

	// Check that webhook timestamp was updated
	freshPayment, err := tc.Services.Payment.GetByID(tc.Context, merchantID, p.ID)
	assert.NoError(t, err)
	assert.NotNil(t, freshPayment.WebhookSentAt)

	// Check Slack notification
	assert.Len(t, *responses, 1)
	assert.Contains(t, (*responses)[0], "processed payment #1: 50 USD for merchant")
	assert.Contains(t, (*responses)[0], mt.Name)
	assert.Contains(t, (*responses)[0], "isTest=false")
	assert.Contains(t, (*responses)[0], "pay.o2pay.co")
}

func marshal(v any) []byte {
	return lo.Must(json.Marshal(v))
}

func assertBind(t *testing.T, request *http.Request, v any) {
	bytes, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(bytes, v))
}

func assertServer(t *testing.T, handler func(*testing.T, http.ResponseWriter, *http.Request)) *httptest.Server {
	fn := func(writer http.ResponseWriter, request *http.Request) {
		handler(t, writer, request)
	}

	return httptest.NewServer(http.HandlerFunc(fn))
}
