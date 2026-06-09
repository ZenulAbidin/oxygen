package paymentapi

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/server/http/common"
	"github.com/oxygenpay/oxygen/internal/service/payment"
	"github.com/oxygenpay/oxygen/internal/util"
	"github.com/oxygenpay/oxygen/pkg/api-payment/v1/model"
	"github.com/pkg/errors"
)

const paramPaymentLinkSlug = "paymentLinkSlug"

func (h *Handler) GetPaymentLink(c echo.Context) error {
	ctx := c.Request().Context()
	slug := c.Param(paramPaymentLinkSlug)

	link, err := h.payments.GetPaymentLinkBySlug(ctx, slug)

	switch {
	case errors.Is(err, payment.ErrNotFound):
		return common.NotFoundResponse(c, "payment link not found")
	case err != nil:
		return err
	}

	mt, err := h.merchants.GetByID(ctx, link.MerchantID, false)
	if err != nil {
		return err
	}

	var price *float64
	if link.Type == payment.LinkTypePayment {
		value, errPrice := link.Price.FiatToFloat64()
		if errPrice != nil {
			return errPrice
		}
		price = util.Ptr(value)
	}

	return c.JSON(http.StatusOK, &model.PaymentLink{
		MerchantName: mt.Name,
		Type:         link.Type.String(),
		Currency:     link.Currency.String(),
		Price:        price,
		Description:  link.Description,
	})
}

func (h *Handler) CreatePaymentFromLink(c echo.Context) error {
	ctx := c.Request().Context()
	slug := c.Param(paramPaymentLinkSlug)

	link, err := h.payments.GetPaymentLinkBySlug(ctx, slug)

	switch {
	case errors.Is(err, payment.ErrNotFound):
		return common.NotFoundResponse(c, "payment link not found")
	case err != nil:
		return err
	}

	var amount []money.Money
	if link.Type == payment.LinkTypeDonation {
		var req model.CreatePaymentFromLinkRequest
		if errBind := c.Bind(&req); errBind != nil {
			return common.ValidationErrorResponse(c, "Invalid JSON")
		}

		price, errPrice := money.FiatFromFloat64(link.Currency, req.Price)
		if errPrice != nil {
			return common.ValidationErrorItemResponse(c, "price", "price should be between %.2f and %.0f", money.FiatMin, money.FiatMax)
		}
		amount = append(amount, price)
	}

	pt, err := h.payments.CreatePaymentFromLink(ctx, link, amount...)
	if err != nil {
		switch {
		case errors.Is(err, payment.ErrLinkValidation):
			return common.ValidationErrorResponse(c, err.Error())
		default:
			return errors.Wrap(err, "unable to create payment from link")
		}
	}

	return c.JSON(http.StatusCreated, &model.PaymentRedirectInfo{ID: pt.PublicID.String()})
}
