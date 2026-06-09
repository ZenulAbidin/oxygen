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
	displayAmount := h.balanceDisplayAmount(ctx, b, currency, isTest)

	usdAmount := "0"
	if !isTest {
		if conv, err := h.blockchain.CryptoToFiat(ctx, displayAmount, money.USD); err == nil {
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
		Amount:                     displayAmount.String(),
		UsdAmount:                  usdAmount,
		MinimalWithdrawalAmountUSD: "0",
	}
}

func (h *Handler) balanceDisplayAmount(
	ctx context.Context,
	b *wallet.Balance,
	currency money.CryptoCurrency,
	isTest bool,
) money.Money {
	if h.payments == nil || !isNativeUTXOCurrency(currency) {
		return b.Amount
	}

	fee, err := h.payments.GetWithdrawalFee(ctx, b.EntityID, b.UUID)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn().Err(err).
				Int64("balance_id", b.ID).
				Str("currency", b.Currency).
				Msg("using ledger balance: unable to calculate UTXO spendable balance")
		}

		return b.Amount
	}

	if fee.MaximumAmount.IsZero() {
		zero, err := currency.MakeAmount("0")
		if err != nil {
			return b.Amount
		}

		return zero
	}

	amount, err := fee.MaximumAmount.Add(fee.CryptoFee)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn().Err(err).
				Int64("balance_id", b.ID).
				Str("currency", b.Currency).
				Msg("using ledger balance: unable to calculate UTXO display amount")
		}

		return b.Amount
	}

	return amount
}

func isNativeUTXOCurrency(currency money.CryptoCurrency) bool {
	if currency.Type != money.Coin {
		return false
	}

	switch currency.Blockchain {
	case money.Blockchain("BTC"), money.Blockchain("LTC"):
		return true
	default:
		return false
	}
}

func fallbackBalanceName(b *wallet.Balance) string {
	if b.Network == "" || b.Network == b.Currency {
		return b.Currency
	}

	return b.Currency + " (" + b.Network + ")"
}
