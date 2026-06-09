package merchantapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/server/http/middleware"
	"github.com/oxygenpay/oxygen/internal/service/wallet"
	"github.com/oxygenpay/oxygen/internal/util"
	"github.com/oxygenpay/oxygen/pkg/api-dashboard/v1/model"
)

func (h *Handler) ListBalances(c echo.Context) error {
	ctx := c.Request().Context()
	mt := middleware.ResolveMerchant(c)

	balances, err := h.wallets.ListBalances(ctx, wallet.EntityTypeMerchant, mt.ID, false)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &model.MerchantBalanceList{
		Results: util.MapSlice(balances, func(b *wallet.Balance) *model.MerchantBalance {
			return h.balanceToResponse(ctx, b)
		}),
	})
}

func (h *Handler) balanceToResponse(ctx context.Context, b *wallet.Balance) *model.MerchantBalance {
	currency, err := h.blockchain.GetCurrencyByTicker(b.Currency)
	if err != nil {
		return &model.MerchantBalance{
			ID:                         b.UUID.String(),
			Blockchain:                 b.Network,
			BlockchainName:             b.Network,
			IsTest:                     strings.Contains(strings.ToLower(b.NetworkID), "test"),
			Name:                       fallbackBalanceName(b),
			Currency:                   b.Currency,
			Ticker:                     b.Currency,
			Amount:                     b.Amount.String(),
			UsdAmount:                  "",
			MinimalWithdrawalAmountUSD: "0",
		}
	}

	isTest := b.NetworkID != currency.NetworkID

	usdAmount := "0"
	if !isTest {
		if conv, err := h.blockchain.CryptoToFiat(ctx, b.Amount, money.USD); err == nil {
			usdAmount = conv.To.String()
		} else {
			usdAmount = ""
		}
	}

	return &model.MerchantBalance{
		ID:                         b.UUID.String(),
		Blockchain:                 currency.Blockchain.String(),
		BlockchainName:             currency.BlockchainName,
		IsTest:                     isTest,
		Name:                       currency.DisplayName(),
		Currency:                   currency.Name,
		Ticker:                     currency.Ticker,
		Amount:                     b.Amount.String(),
		UsdAmount:                  usdAmount,
		MinimalWithdrawalAmountUSD: "0",
	}
}

func fallbackBalanceName(b *wallet.Balance) string {
	if b.Network == "" || b.Network == b.Currency {
		return b.Currency
	}

	return b.Currency + " (" + b.Network + ")"
}
