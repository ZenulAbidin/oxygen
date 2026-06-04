package processing

import (
	"testing"

	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/stretchr/testify/assert"
)

func TestInputValidateAllowsBitcoinWebhookWithoutSender(t *testing.T) {
	btc := money.CryptoCurrency{
		Blockchain: money.Blockchain("BTC"),
		Ticker:     "BTC",
		Decimals:   8,
	}
	amount := money.MustCryptoFromRaw("BTC", "100000", 8)

	err := Input{
		Currency:      btc,
		Amount:        amount,
		TransactionID: "btc-tx",
		NetworkID:     "mainnet",
	}.validate()

	assert.NoError(t, err)
}

func TestInputValidateRequiresSenderForAccountBasedWebhook(t *testing.T) {
	eth := money.CryptoCurrency{
		Blockchain: money.Blockchain("ETH"),
		Ticker:     "ETH",
		Decimals:   18,
	}
	amount := money.MustCryptoFromRaw("ETH", "1000000000000000000", 18)

	err := Input{
		Currency:      eth,
		Amount:        amount,
		TransactionID: "eth-tx",
		NetworkID:     "1",
	}.validate()

	assert.ErrorContains(t, err, "missing SenderAddress")
}
