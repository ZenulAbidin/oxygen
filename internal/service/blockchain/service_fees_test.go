package blockchain

import (
	"context"
	"testing"

	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/provider/tatum"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateFee_BitcoinUsesUTXOFeeEstimate(t *testing.T) {
	logger := zerolog.Nop()
	s := &Service{
		providers: Providers{
			Tatum: tatum.New(tatum.Config{}, nil, &logger),
		},
	}

	btc := money.CryptoCurrency{
		Blockchain: money.Blockchain("BTC"),
		Ticker:     "BTC",
		Name:       "BTC",
		Type:       money.Coin,
		Decimals:   8,
	}

	fee, err := s.CalculateFee(context.Background(), btc, btc, true)

	require.NoError(t, err)
	btcFee, err := fee.ToBitcoinFee()
	require.NoError(t, err)
	assert.Equal(t, "2", btcFee.FeeSatPerVByte)
	assert.NotEmpty(t, btcFee.TotalCostSats)
	assert.NotEmpty(t, btcFee.TotalCostUSD)
}

func TestCalculateWithdrawalFeeUSD_BitcoinDoesNotApplyMerchantWithdrawalMarkup(t *testing.T) {
	logger := zerolog.Nop()
	s := &Service{
		providers: Providers{
			Tatum: tatum.New(tatum.Config{}, nil, &logger),
		},
	}

	btc := money.CryptoCurrency{
		Blockchain: money.Blockchain("BTC"),
		Ticker:     "BTC",
		Name:       "BTC",
		Type:       money.Coin,
		Decimals:   8,
	}

	fee, err := s.CalculateWithdrawalFeeUSD(context.Background(), btc, btc, false)

	require.NoError(t, err)
	assert.Equal(t, "0.91", fee.String())
}

func TestCalculateFee_LitecoinUsesUTXOFeeEstimate(t *testing.T) {
	logger := zerolog.Nop()
	s := &Service{
		providers: Providers{
			Tatum: tatum.New(tatum.Config{}, nil, &logger),
		},
	}

	ltc := money.CryptoCurrency{
		Blockchain: money.Blockchain("LTC"),
		Ticker:     "LTC",
		Name:       "LTC",
		Type:       money.Coin,
		Decimals:   8,
	}

	fee, err := s.CalculateFee(context.Background(), ltc, ltc, true)

	require.NoError(t, err)
	ltcFee, err := fee.ToBitcoinFee()
	require.NoError(t, err)
	assert.Equal(t, "1", ltcFee.FeeSatPerVByte)
	assert.NotEmpty(t, ltcFee.TotalCostSats)
	assert.NotEmpty(t, ltcFee.TotalCostUSD)
}

func TestCalculateFee_RejectsInvalidArguments(t *testing.T) {
	s := &Service{}

	baseCurrency := money.CryptoCurrency{
		Blockchain: money.Blockchain("ETH"),
		Ticker:     "ETH",
		Name:       "ETH",
		Type:       money.Token,
	}
	currency := money.CryptoCurrency{
		Blockchain: money.Blockchain("ETH"),
		Ticker:     "ETH_USDT",
		Name:       "USDT",
		Type:       money.Token,
	}

	fee, err := s.CalculateFee(context.Background(), baseCurrency, currency, false)

	assert.Equal(t, Fee{}, fee)
	assert.ErrorIs(t, err, ErrValidation)
	assert.ErrorContains(t, err, "native coin")
}
