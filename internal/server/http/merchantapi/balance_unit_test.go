package merchantapi

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/service/blockchain"
	"github.com/oxygenpay/oxygen/internal/service/wallet"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type balanceBlockchainStub struct {
	currency       money.CryptoCurrency
	currencyErr    error
	minWithdrawal  money.Money
	minimumErr     error
	conversion     blockchain.Conversion
	conversionErr  error
	lastTicker     string
	lastConversion money.Money
}

func (s *balanceBlockchainStub) ListSupportedCurrencies(bool) []money.CryptoCurrency {
	panic("unexpected call")
}

func (s *balanceBlockchainStub) ListBlockchainCurrencies(money.Blockchain) []money.CryptoCurrency {
	panic("unexpected call")
}

func (s *balanceBlockchainStub) GetCurrencyByTicker(ticker string) (money.CryptoCurrency, error) {
	s.lastTicker = ticker
	return s.currency, s.currencyErr
}

func (s *balanceBlockchainStub) GetNativeCoin(money.Blockchain) (money.CryptoCurrency, error) {
	panic("unexpected call")
}

func (s *balanceBlockchainStub) GetCurrencyByBlockchainAndContract(money.Blockchain, string, string) (money.CryptoCurrency, error) {
	panic("unexpected call")
}

func (s *balanceBlockchainStub) GetMinimalWithdrawalByTicker(string) (money.Money, error) {
	return s.minWithdrawal, s.minimumErr
}

func (s *balanceBlockchainStub) GetUSDMinimalInternalTransferByTicker(string) (money.Money, error) {
	panic("unexpected call")
}

func (s *balanceBlockchainStub) GetExchangeRate(context.Context, string, string) (blockchain.ExchangeRate, error) {
	panic("unexpected call")
}

func (s *balanceBlockchainStub) Convert(context.Context, string, string, string) (blockchain.Conversion, error) {
	panic("unexpected call")
}

func (s *balanceBlockchainStub) FiatToFiat(context.Context, money.Money, money.FiatCurrency) (blockchain.Conversion, error) {
	panic("unexpected call")
}

func (s *balanceBlockchainStub) FiatToCrypto(context.Context, money.Money, money.CryptoCurrency) (blockchain.Conversion, error) {
	panic("unexpected call")
}

func (s *balanceBlockchainStub) CryptoToFiat(_ context.Context, from money.Money, _ money.FiatCurrency) (blockchain.Conversion, error) {
	s.lastConversion = from
	return s.conversion, s.conversionErr
}

func TestBalanceToResponse_FallsBackForUnsupportedCurrency(t *testing.T) {
	handler := &Handler{
		blockchain: &balanceBlockchainStub{
			currencyErr: errors.Wrap(blockchain.ErrCurrencyNotFound, "BTC"),
		},
	}

	balance := &wallet.Balance{
		UUID:      uuid.New(),
		Network:   "BTC",
		NetworkID: "mainnet",
		Currency:  "BTC",
		Amount:    money.MustCryptoFromRaw("BTC", "50000000", 8),
	}

	response := handler.balanceToResponse(context.Background(), balance)

	assert.Equal(t, balance.UUID.String(), response.ID)
	assert.Equal(t, "BTC", response.Blockchain)
	assert.Equal(t, "BTC", response.BlockchainName)
	assert.Equal(t, "BTC", response.Name)
	assert.Equal(t, "BTC", response.Currency)
	assert.Equal(t, "BTC", response.Ticker)
	assert.Equal(t, "0.5", response.Amount)
	assert.Equal(t, "", response.UsdAmount)
	assert.Equal(t, "", response.MinimalWithdrawalAmountUSD)
	assert.False(t, response.IsTest)
}

func TestBalanceToResponse_UsesResolvedCurrencyAndBestEffortUSD(t *testing.T) {
	eth := money.CryptoCurrency{
		Blockchain:     money.Blockchain("ETH"),
		BlockchainName: "Ethereum",
		NetworkID:      "1",
		TestNetworkID:  "5",
		Type:           money.Coin,
		Ticker:         "ETH",
		Name:           "ETH",
		Decimals:       18,
	}
	usdAmount := lo.Must(money.FiatFromFloat64(money.USD, 900))
	balance := &wallet.Balance{
		UUID:      uuid.New(),
		Network:   "ETH",
		NetworkID: "1",
		Currency:  "ETH",
		Amount:    money.MustCryptoFromRaw("ETH", "500000000000000000", 18),
	}
	handler := &Handler{
		blockchain: &balanceBlockchainStub{
			currency: eth,
			conversion: blockchain.Conversion{
				Type: blockchain.ConversionTypeCryptoToFiat,
				From: balance.Amount,
				To:   usdAmount,
				Rate: 1800,
			},
		},
	}

	response := handler.balanceToResponse(context.Background(), balance)

	assert.Equal(t, balance.UUID.String(), response.ID)
	assert.Equal(t, "ETH", response.Blockchain)
	assert.Equal(t, "Ethereum", response.BlockchainName)
	assert.Equal(t, "ETH (Ethereum)", response.Name)
	assert.Equal(t, "ETH", response.Currency)
	assert.Equal(t, "ETH", response.Ticker)
	assert.Equal(t, "0.5", response.Amount)
	assert.Equal(t, "900", response.UsdAmount)
	assert.Equal(t, "", response.MinimalWithdrawalAmountUSD)
	assert.False(t, response.IsTest)

	stub := handler.blockchain.(*balanceBlockchainStub)
	require.True(t, stub.lastConversion.Equals(balance.Amount))
	assert.Equal(t, "ETH", stub.lastTicker)
}
