package payment

import (
	"strconv"
	"testing"

	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/service/blockchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestMinimumWithdrawalAmount(t *testing.T) {
	for _, tc := range []struct {
		name           string
		currency       money.CryptoCurrency
		expectedAmount string
		expectedRaw    string
	}{
		{
			name: "account chain",
			currency: money.CryptoCurrency{
				Blockchain: money.Blockchain("ETH"),
				Ticker:     "ETH",
				Decimals:   18,
			},
			expectedAmount: "0",
			expectedRaw:    "0",
		},
		{
			name: "utxo chain",
			currency: money.CryptoCurrency{
				Blockchain: money.Blockchain("BTC"),
				Ticker:     "BTC",
				Decimals:   8,
			},
			expectedAmount: "0.00000546",
			expectedRaw:    strconv.FormatInt(blockchain.UTXODustSats, 10),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			amount, err := minimumWithdrawalAmount(tc.currency)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedAmount, amount.String())
			assert.Equal(t, tc.expectedRaw, amount.StringRaw())
		})
	}
}
