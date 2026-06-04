package payment

import (
	"testing"

	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/stretchr/testify/assert"
)

func TestValidateWithdrawalRuntimeCurrency(t *testing.T) {
	for _, tc := range []struct {
		name          string
		currency      money.CryptoCurrency
		expectErr     bool
		errorContains string
	}{
		{
			name: "runtime supported",
			currency: money.CryptoCurrency{
				Blockchain: money.Blockchain("ETH"),
				Ticker:     "ETH",
				Type:       money.Coin,
			},
		},
		{
			name: "btc runtime supported",
			currency: money.CryptoCurrency{
				Blockchain: money.Blockchain("BTC"),
				Ticker:     "BTC",
				Type:       money.Coin,
			},
		},
		{
			name: "unsupported runtime",
			currency: money.CryptoCurrency{
				Blockchain: money.Blockchain("DOGE"),
				Ticker:     "DOGE",
				Type:       money.Coin,
			},
			expectErr:     true,
			errorContains: "dedicated runtime support",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateWithdrawalRuntimeCurrency(tc.currency)
			if tc.expectErr {
				assert.ErrorIs(t, err, ErrValidation)
				assert.ErrorContains(t, err, tc.errorContains)
				return
			}

			assert.NoError(t, err)
		})
	}
}
