package payment

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/oxygenpay/oxygen/internal/db/repository"
	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/util"
	"github.com/pkg/errors"
)

type Link struct {
	ID       int64
	PublicID uuid.UUID
	Slug     string
	URL      string

	CreatedAt time.Time
	UpdatedAt time.Time

	MerchantID int64
	Type       LinkType
	Name       string

	Currency    money.FiatCurrency
	Price       money.Money
	Description *string

	SuccessAction  SuccessAction
	RedirectURL    *string
	SuccessMessage *string

	IsTest bool
}

type LinkType string

const (
	LinkTypePayment  LinkType = "payment"
	LinkTypeDonation LinkType = "donation"
)

type SuccessAction string

const (
	SuccessActionRedirect    SuccessAction = "redirect"
	SuccessActionShowMessage SuccessAction = "showMessage"
)

type CreateLinkProps struct {
	Type LinkType
	Name string

	Currency    money.FiatCurrency
	Price       money.Money
	Description *string

	SuccessAction  SuccessAction
	RedirectURL    *string
	SuccessMessage *string

	IsTest bool
}

func (s *Service) ListPaymentLinks(ctx context.Context, merchantID int64) ([]*Link, error) {
	entries, err := s.repo.ListPaymentLinks(ctx, repository.ListPaymentLinksParams{
		MerchantID: merchantID,
		Limit:      100,
	})
	if err != nil {
		return nil, err
	}

	links := make([]*Link, len(entries))

	for i := range entries {
		link, err := s.entryToLink(entries[i])
		if err != nil {
			return nil, err
		}

		links[i] = link
	}

	return links, nil
}

func (s *Service) GetPaymentLinkBySlug(ctx context.Context, slug string) (*Link, error) {
	link, err := s.repo.GetPaymentLinkBySlug(ctx, slug)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, ErrNotFound
	case err != nil:
		return nil, err
	}

	return s.entryToLink(link)
}

func (s *Service) GetPaymentLinkByPublicID(ctx context.Context, merchantID int64, id uuid.UUID) (*Link, error) {
	link, err := s.repo.GetPaymentLinkByPublicID(ctx, repository.GetPaymentLinkByPublicIDParams{
		MerchantID: merchantID,
		Uuid:       id,
	})

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, ErrNotFound
	case err != nil:
		return nil, err
	}

	return s.entryToLink(link)
}

func (s *Service) GetPaymentLinkByID(ctx context.Context, merchantID, id int64) (*Link, error) {
	link, err := s.repo.GetPaymentLinkByID(ctx, repository.GetPaymentLinkByIDParams{
		MerchantID: merchantID,
		ID:         id,
	})

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, ErrNotFound
	case err != nil:
		return nil, err
	}

	return s.entryToLink(link)
}

func (s *Service) CreatePaymentLink(ctx context.Context, merchantID int64, props CreateLinkProps) (*Link, error) {
	props.fillDefaults()
	if err := props.validate(); err != nil {
		return nil, err
	}

	_, err := s.merchants.GetByID(ctx, merchantID, false)
	if err != nil {
		return nil, err
	}

	var description string
	if props.Description != nil {
		description = *props.Description
	}

	var price pgtype.Numeric
	if props.Type == LinkTypePayment {
		price = repository.MoneyToNumeric(props.Price)
	} else {
		price = pgtype.Numeric{Status: pgtype.Null}
	}

	link, err := s.repo.CreatePaymentLink(ctx, repository.CreatePaymentLinkParams{
		Uuid:           uuid.New(),
		Slug:           util.Strings.Random(8),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		MerchantID:     merchantID,
		Type:           props.Type.String(),
		Name:           props.Name,
		Description:    description,
		Price:          price,
		Decimals:       int32(money.FiatDecimals),
		Currency:       props.Currency.String(),
		SuccessAction:  string(props.SuccessAction),
		RedirectUrl:    repository.PointerStringToNullable(props.RedirectURL),
		SuccessMessage: repository.PointerStringToNullable(props.SuccessMessage),
		IsTest:         props.IsTest,
	})

	if err != nil {
		return nil, err
	}

	return s.entryToLink(link)
}

func (s *Service) DeletePaymentLinkByPublicID(ctx context.Context, merchantID int64, id uuid.UUID) error {
	if _, err := s.GetPaymentLinkByPublicID(ctx, merchantID, id); err != nil {
		return err
	}

	return s.repo.DeletePaymentLinkByPublicID(ctx, repository.DeletePaymentLinkByPublicIDParams{
		MerchantID: merchantID,
		Uuid:       id,
	})
}

func (s *Service) CreatePaymentFromLink(ctx context.Context, link *Link, amount ...money.Money) (*Payment, error) {
	price, err := linkPaymentAmount(link, amount...)
	if err != nil {
		return nil, err
	}

	props := CreatePaymentProps{
		MerchantOrderUUID: uuid.New(),
		Money:             price,
		RedirectURL:       link.RedirectURL,
		Description:       link.Description,
		IsTest:            link.IsTest,
	}

	return s.CreatePayment(ctx, link.MerchantID, props, FromLink(link))
}

func (p *CreateLinkProps) fillDefaults() {
	if p.Type == "" {
		p.Type = LinkTypePayment
	}
	if p.Currency == "" && p.Price.Ticker() != "" {
		p.Currency = money.FiatCurrency(p.Price.Ticker())
	}
}

func (p CreateLinkProps) validate() error {
	if p.Name == "" {
		return errors.Wrap(ErrLinkValidation, "name required")
	}

	switch p.Type {
	case LinkTypePayment:
		if p.Price.Type() != money.Fiat {
			return errors.Wrap(ErrLinkValidation, "invalid currency")
		}
		if p.Currency != "" && p.Price.Ticker() != p.Currency.String() {
			return errors.Wrap(ErrLinkValidation, "price currency should match link currency")
		}

		float, err := p.Price.FiatToFloat64()
		if err != nil {
			return errors.Wrap(ErrLinkValidation, "invalid price")
		}

		if float <= 0.0 {
			return errors.Wrap(ErrLinkValidation, "price can't be zero or negative")
		}
	case LinkTypeDonation:
		if _, err := money.MakeFiatCurrency(p.Currency.String()); err != nil {
			return errors.Wrap(ErrLinkValidation, "invalid currency")
		}
	default:
		return errors.Wrap(ErrLinkValidation, "invalid link type")
	}

	if p.Currency == "" {
		return errors.Wrap(ErrLinkValidation, "invalid currency")
	}

	if _, err := money.MakeFiatCurrency(p.Currency.String()); err != nil {
		return errors.Wrap(ErrLinkValidation, "invalid currency")
	}

	switch p.SuccessAction {
	case SuccessActionRedirect:
		if p.RedirectURL == nil {
			return errors.Wrap(ErrLinkValidation, "redirectUrl required")
		}
		if err := validateURL(*p.RedirectURL); err != nil {
			return errors.Wrapf(ErrLinkValidation, "invalid redirect url: %s", err.Error())
		}
		if p.SuccessMessage != nil {
			return errors.Wrap(ErrLinkValidation, "successMessage should not be present")
		}
	case SuccessActionShowMessage:
		if p.SuccessMessage == nil || *p.SuccessMessage == "" {
			return errors.Wrapf(ErrLinkValidation, "successMessage required")
		}
		if p.RedirectURL != nil {
			return errors.Wrap(ErrLinkValidation, "redirectUrl should not be present")
		}
	default:
		return errors.Wrap(ErrLinkValidation, "invalid successAction")
	}

	return nil
}

func linkPaymentAmount(link *Link, amount ...money.Money) (money.Money, error) {
	switch link.Type {
	case LinkTypePayment:
		if len(amount) > 0 {
			return money.Money{}, errors.Wrap(ErrLinkValidation, "amount should not be supplied for fixed payment links")
		}

		return link.Price, nil
	case LinkTypeDonation:
		if len(amount) == 0 {
			return money.Money{}, errors.Wrap(ErrLinkValidation, "donation amount required")
		}
		if len(amount) > 1 {
			return money.Money{}, errors.Wrap(ErrLinkValidation, "too many donation amounts")
		}
		if amount[0].Type() != money.Fiat || amount[0].Ticker() != link.Currency.String() {
			return money.Money{}, errors.Wrap(ErrLinkValidation, "invalid donation currency")
		}

		float, err := amount[0].FiatToFloat64()
		if err != nil {
			return money.Money{}, errors.Wrap(ErrLinkValidation, "invalid donation amount")
		}
		if float <= 0.0 {
			return money.Money{}, errors.Wrap(ErrLinkValidation, "donation amount can't be zero or negative")
		}

		return amount[0], nil
	default:
		return money.Money{}, errors.Wrap(ErrLinkValidation, "invalid link type")
	}
}

func (t LinkType) String() string {
	if t == "" {
		return string(LinkTypePayment)
	}

	return string(t)
}

func linkType(raw string) LinkType {
	switch LinkType(raw) {
	case LinkTypeDonation:
		return LinkTypeDonation
	default:
		return LinkTypePayment
	}
}

func nullableLinkPrice(link repository.PaymentLink, currency money.FiatCurrency) (money.Money, error) {
	if link.Price.Status == pgtype.Null {
		return money.NewFromBigInt(money.Fiat, currency.String(), big.NewInt(0), money.FiatDecimals)
	}

	bigInt, err := repository.NumericToBigInt(link.Price)
	if err != nil {
		return money.Money{}, err
	}

	return money.NewFromBigInt(money.Fiat, currency.String(), bigInt, int64(link.Decimals))
}

func (s *Service) linkURL(slug string) string {
	return fmt.Sprintf("%s/link/%s", s.basePath, slug)
}

func (s *Service) entryToLink(link repository.PaymentLink) (*Link, error) {
	currency, err := money.MakeFiatCurrency(link.Currency)
	if err != nil {
		return nil, err
	}

	price, err := nullableLinkPrice(link, currency)
	if err != nil {
		return nil, err
	}

	var desc *string
	if link.Description != "" {
		desc = &link.Description
	}

	return &Link{
		ID:       link.ID,
		PublicID: link.Uuid,
		Slug:     link.Slug,
		URL:      s.linkURL(link.Slug),

		CreatedAt: link.CreatedAt,
		UpdatedAt: link.UpdatedAt,

		MerchantID: link.MerchantID,
		Type:       linkType(link.Type),
		Name:       link.Name,

		Currency:    currency,
		Price:       price,
		Description: desc,

		SuccessAction:  SuccessAction(link.SuccessAction),
		RedirectURL:    repository.NullableStringToPointer(link.RedirectUrl),
		SuccessMessage: repository.NullableStringToPointer(link.SuccessMessage),

		IsTest: link.IsTest,
	}, nil
}
