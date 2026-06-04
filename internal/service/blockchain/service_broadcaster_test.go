package blockchain

import (
	"context"
	"testing"

	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/stretchr/testify/assert"
)

func TestBroadcastTransaction_RejectsUnsupportedRuntimeBeforeProviderAccess(t *testing.T) {
	s := &Service{}

	txHash, err := s.BroadcastTransaction(context.Background(), money.Blockchain("DOGE"), "raw-tx", true)

	assert.Empty(t, txHash)
	assert.ErrorIs(t, err, ErrUnsupportedRuntime)
	assert.ErrorContains(t, err, "unsupported blockchain")
}

func TestGetTransactionReceipt_RejectsUnsupportedRuntimeBeforeProviderAccess(t *testing.T) {
	s := &Service{}

	receipt, err := s.GetTransactionReceipt(context.Background(), money.Blockchain("DOGE"), "tx-id", true)

	assert.Nil(t, receipt)
	assert.ErrorIs(t, err, ErrUnsupportedRuntime)
	assert.ErrorContains(t, err, "unsupported blockchain")
}

func TestValidateTransactionRuntimeBlockchain(t *testing.T) {
	for _, tc := range []struct {
		name          string
		blockchain    money.Blockchain
		expectErr     bool
		errorContains string
	}{
		{name: "eth", blockchain: money.Blockchain("ETH")},
		{name: "matic", blockchain: money.Blockchain("MATIC")},
		{name: "bsc", blockchain: money.Blockchain("BSC")},
		{name: "tron", blockchain: money.Blockchain("TRON")},
		{name: "btc", blockchain: money.Blockchain("BTC")},
		{name: "ltc", blockchain: money.Blockchain("LTC")},
		{name: "unknown", blockchain: money.Blockchain("DOGE"), expectErr: true, errorContains: "unsupported blockchain"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTransactionRuntimeBlockchain(tc.blockchain)
			if tc.expectErr {
				assert.ErrorIs(t, err, ErrUnsupportedRuntime)
				assert.ErrorContains(t, err, tc.errorContains)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestListRuntimeSupportedBlockchains(t *testing.T) {
	currencies := NewCurrencies()

	for _, currency := range []money.CryptoCurrency{
		{Blockchain: money.Blockchain("ETH"), Ticker: "ETH", Name: "ETH"},
		{Blockchain: money.Blockchain("TRON"), Ticker: "TRON", Name: "TRON"},
		{Blockchain: money.Blockchain("MATIC"), Ticker: "MATIC", Name: "MATIC"},
		{Blockchain: money.Blockchain("BSC"), Ticker: "BNB", Name: "BNB"},
		{Blockchain: money.Blockchain("BTC"), Ticker: "BTC", Name: "BTC"},
		{Blockchain: money.Blockchain("LTC"), Ticker: "LTC", Name: "LTC"},
		{Blockchain: money.Blockchain("DOGE"), Ticker: "DOGE", Name: "DOGE"},
	} {
		currencies.addCurrency(currency)
	}

	assert.Equal(t, []money.Blockchain{
		money.Blockchain("BSC"),
		money.Blockchain("BTC"),
		money.Blockchain("ETH"),
		money.Blockchain("LTC"),
		money.Blockchain("MATIC"),
		money.Blockchain("TRON"),
	}, currencies.ListRuntimeSupportedBlockchains())
}
