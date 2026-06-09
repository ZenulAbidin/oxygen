package blockchain

import (
	"testing"
	"time"

	"github.com/oxygenpay/oxygen/internal/money"
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

	t.Run("supports sub-sat-per-vbyte fee rates", func(t *testing.T) {
		selected, feeSats, changeSats, estimatedVBytes, err := selectBitcoinUTXOs([]BitcoinUTXO{
			{Hash: "single", AmountSats: 557},
		}, 546, 0.1)

		require.NoError(t, err)
		require.Len(t, selected, 1)
		assert.Equal(t, int64(11), feeSats)
		assert.Zero(t, changeSats)
		assert.Equal(t, int64(109), estimatedVBytes)
	})
}

func TestSelectBitcoinSweepUTXOs(t *testing.T) {
	t.Run("sweeps economical UTXOs minus fee", func(t *testing.T) {
		selected, amountSats, feeSats, estimatedVBytes, err := selectBitcoinSweepUTXOs([]BitcoinUTXO{
			{Hash: "large", AmountSats: 3_076},
			{Hash: "dust-after-fee", AmountSats: 1_538},
		}, 10)

		require.NoError(t, err)
		require.Len(t, selected, 2)
		assert.Equal(t, int64(2_844), amountSats)
		assert.Equal(t, int64(1_770), feeSats)
		assert.Equal(t, int64(177), estimatedVBytes)
	})

	t.Run("skips UTXOs that cost more to spend than they add", func(t *testing.T) {
		selected, amountSats, feeSats, estimatedVBytes, err := selectBitcoinSweepUTXOs([]BitcoinUTXO{
			{Hash: "uneconomical", AmountSats: 500},
			{Hash: "large", AmountSats: 3_076},
		}, 10)

		require.NoError(t, err)
		require.Len(t, selected, 1)
		assert.Equal(t, "large", selected[0].Hash)
		assert.Equal(t, int64(1_986), amountSats)
		assert.Equal(t, int64(1_090), feeSats)
		assert.Equal(t, int64(109), estimatedVBytes)
	})

	t.Run("requires sweep output above dust", func(t *testing.T) {
		selected, amountSats, feeSats, estimatedVBytes, err := selectBitcoinSweepUTXOs([]BitcoinUTXO{
			{Hash: "small", AmountSats: 1_538},
		}, 10)

		assert.ErrorIs(t, err, ErrInsufficientFunds)
		assert.Empty(t, selected)
		assert.Zero(t, amountSats)
		assert.Zero(t, feeSats)
		assert.Zero(t, estimatedVBytes)
	})
}

func TestEconomicalUTXOFeeRate(t *testing.T) {
	feeRate, ok := economicalUTXOFeeRate(map[string]float64{
		"1":   1.5,
		"144": 0.1,
	})

	require.True(t, ok)
	assert.Equal(t, 0.1, feeRate)
}

func TestBitcoinSweepAmountFromUTXOsSupportsLitecoin(t *testing.T) {
	ltc := testUTXOCurrency("LTC", "LTC")
	fee := testUTXOFee(ltc, "0.746")

	amount, err := BitcoinSweepAmountFromUTXOs(ltc, fee, []BitcoinUTXO{
		{Hash: "small", AmountSats: 1_538},
	})

	require.NoError(t, err)
	assert.Equal(t, "1456", amount.StringRaw())
	assert.Equal(t, "0.00001456", amount.String())
}

func TestMaxBitcoinTransactionAmountFromUTXOsIncludesFinalFee(t *testing.T) {
	ltc := testUTXOCurrency("LTC", "LTC")
	fee := testUTXOFee(ltc, "0.746")
	maxTotalCost := loMust(ltc.MakeAmount("6152"))

	amount, effectiveFee, err := MaxBitcoinTransactionAmountAndFeeFromUTXOs(ltc, fee, maxTotalCost, []BitcoinUTXO{
		{Hash: "consolidated-1", AmountSats: 1_456},
		{Hash: "consolidated-2", AmountSats: 1_456},
		{Hash: "consolidated-3", AmountSats: 1_456},
	})

	require.NoError(t, err)
	assert.Equal(t, "4185", amount.StringRaw())
	assert.Equal(t, "0.00004185", amount.String())
	assert.Equal(t, "183", effectiveFee.StringRaw())
	assert.Equal(t, "0.00000183", effectiveFee.String())

	amountOnly, err := MaxBitcoinTransactionAmountFromUTXOs(ltc, fee, maxTotalCost, []BitcoinUTXO{
		{Hash: "consolidated-1", AmountSats: 1_456},
		{Hash: "consolidated-2", AmountSats: 1_456},
		{Hash: "consolidated-3", AmountSats: 1_456},
	})
	require.NoError(t, err)
	assert.Equal(t, amount.StringRaw(), amountOnly.StringRaw())
}

func testUTXOCurrency(ticker string, blockchain money.Blockchain) money.CryptoCurrency {
	return money.CryptoCurrency{
		Blockchain:    blockchain,
		NetworkID:     "mainnet",
		TestNetworkID: "testnet",
		Type:          money.Coin,
		Ticker:        ticker,
		Decimals:      8,
	}
}

func testUTXOFee(currency money.CryptoCurrency, feeRate string) Fee {
	return NewFee(currency, time.Now().UTC(), false, BitcoinFee{FeeSatPerVByte: feeRate})
}

func loMust[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}

	return value
}
