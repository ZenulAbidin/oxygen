package transaction

import (
	"testing"

	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/stretchr/testify/assert"
)

func TestReceiveTransactionValidateAllowsEmptySenderAddress(t *testing.T) {
	err := ReceiveTransaction{
		Status:          StatusInProgress,
		TransactionHash: "tx-id",
		FactAmount:      money.MustCryptoFromRaw("BTC", "100000", 8),
	}.validate()

	assert.NoError(t, err)
}

func TestConfirmTransactionValidateAllowsEmptySenderAddress(t *testing.T) {
	params := ConfirmTransaction{
		Status:              StatusCompleted,
		TransactionHash:     "tx-id",
		FactAmount:          money.MustCryptoFromRaw("BTC", "100000", 8),
		NetworkFee:          money.MustCryptoFromRaw("BTC", "0", 8),
		allowZeroNetworkFee: true,
	}
	err := params.validate()

	assert.NoError(t, err)
}
