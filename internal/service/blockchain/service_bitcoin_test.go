package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectBitcoinUTXOs(t *testing.T) {
	t.Run("returns change output when change is above dust", func(t *testing.T) {
		selected, feeSats, changeSats, estimatedVBytes, err := selectBitcoinUTXOs([]BitcoinUTXO{
			{Hash: "large", AmountSats: 6_000},
			{Hash: "small", AmountSats: 3_000},
		}, 4_000, 2)

		require.NoError(t, err)
		require.Len(t, selected, 1)
		assert.Equal(t, "large", selected[0].Hash)
		assert.Equal(t, int64(280), feeSats)
		assert.Equal(t, int64(1_720), changeSats)
		assert.Equal(t, int64(140), estimatedVBytes)
	})

	t.Run("drops dust change into the fee", func(t *testing.T) {
		selected, feeSats, changeSats, estimatedVBytes, err := selectBitcoinUTXOs([]BitcoinUTXO{
			{Hash: "single", AmountSats: 4_400},
		}, 4_000, 2)

		require.NoError(t, err)
		require.Len(t, selected, 1)
		assert.Equal(t, int64(400), feeSats)
		assert.Zero(t, changeSats)
		assert.Equal(t, int64(109), estimatedVBytes)
	})

	t.Run("requires enough value for amount and fee", func(t *testing.T) {
		selected, feeSats, changeSats, estimatedVBytes, err := selectBitcoinUTXOs([]BitcoinUTXO{
			{Hash: "insufficient", AmountSats: 4_100},
		}, 4_000, 2)

		assert.ErrorIs(t, err, ErrInsufficientFunds)
		assert.Empty(t, selected)
		assert.Zero(t, feeSats)
		assert.Zero(t, changeSats)
		assert.Zero(t, estimatedVBytes)
	})
}
