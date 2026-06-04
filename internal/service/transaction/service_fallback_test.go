package transaction

import (
	"database/sql"
	"testing"
	"time"

	"github.com/jackc/pgtype"
	"github.com/oxygenpay/oxygen/internal/db/repository"
	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/service/blockchain"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type resolverStub struct {
	currency       money.CryptoCurrency
	currencyErr    error
	nativeCoin     money.CryptoCurrency
	nativeCoinErr  error
	lastTicker     string
	lastBlockchain money.Blockchain
}

func (s *resolverStub) ListSupportedCurrencies(bool) []money.CryptoCurrency {
	panic("unexpected call")
}

func (s *resolverStub) ListBlockchainCurrencies(money.Blockchain) []money.CryptoCurrency {
	panic("unexpected call")
}

func (s *resolverStub) GetCurrencyByTicker(ticker string) (money.CryptoCurrency, error) {
	s.lastTicker = ticker
	return s.currency, s.currencyErr
}

func (s *resolverStub) GetNativeCoin(blockchain money.Blockchain) (money.CryptoCurrency, error) {
	s.lastBlockchain = blockchain
	return s.nativeCoin, s.nativeCoinErr
}

func (s *resolverStub) GetCurrencyByBlockchainAndContract(money.Blockchain, string, string) (money.CryptoCurrency, error) {
	panic("unexpected call")
}

func (s *resolverStub) GetMinimalWithdrawalByTicker(string) (money.Money, error) {
	panic("unexpected call")
}

func (s *resolverStub) GetUSDMinimalInternalTransferByTicker(string) (money.Money, error) {
	panic("unexpected call")
}

func TestEntryToTransaction_FallsBackForUnsupportedCurrency(t *testing.T) {
	svc := &Service{
		blockchain: &resolverStub{
			currencyErr:   errors.Wrap(blockchain.ErrCurrencyNotFound, "BTC"),
			nativeCoinErr: blockchain.ErrCurrencyNotFound,
		},
	}

	entry := repository.Transaction{
		ID:               1,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		MerchantID:       11,
		Status:           string(StatusPending),
		Type:             string(TypeWithdrawal),
		EntityID:         sql.NullInt64{Int64: 22, Valid: true},
		RecipientAddress: "bc1qrecipient",
		Blockchain:       "BTC",
		CurrencyType:     string(money.Coin),
		Currency:         "BTC",
		Decimals:         8,
		Amount:           repository.MoneyToNumeric(money.MustCryptoFromRaw("BTC", "50000000", 8)),
		ServiceFee:       repository.MoneyToNumeric(money.MustCryptoFromRaw("BTC", "1000", 8)),
		UsdAmount:        repository.MoneyToNumeric(mustUSD(t, 100)),
		Metadata:         pgtype.JSONB{Status: pgtype.Null},
		NetworkID:        repository.StringToNullable("mainnet"),
		IsTest:           false,
		NetworkDecimals:  8,
	}

	tx, err := svc.entryToTransaction(entry)
	require.NoError(t, err)

	assert.Equal(t, "BTC", tx.Currency.Ticker)
	assert.Equal(t, money.Blockchain("BTC"), tx.Currency.Blockchain)
	assert.Equal(t, "BTC", tx.Currency.Name)
	assert.Equal(t, "BTC", tx.Currency.BlockchainName)
	assert.Equal(t, "mainnet", tx.Currency.NetworkID)
	assert.Equal(t, int64(8), tx.Currency.Decimals)
	assert.Equal(t, "0.5", tx.Amount.String())

	stub := svc.blockchain.(*resolverStub)
	assert.Equal(t, "BTC", stub.lastTicker)
	assert.Equal(t, money.Blockchain("BTC"), stub.lastBlockchain)
}

func TestEntryToTransaction_FallsBackForUnsupportedNetworkFeeCurrency(t *testing.T) {
	legacyToken := money.CryptoCurrency{
		Blockchain:     money.Blockchain("BTC"),
		BlockchainName: "Bitcoin",
		NetworkID:      "mainnet",
		Type:           money.Token,
		Ticker:         "BTC_OLD",
		Name:           "BTC_OLD",
		Decimals:       8,
	}

	svc := &Service{
		blockchain: &resolverStub{
			currency:      legacyToken,
			nativeCoinErr: blockchain.ErrCurrencyNotFound,
		},
	}

	entry := repository.Transaction{
		ID:               2,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		MerchantID:       11,
		Status:           string(StatusFailed),
		Type:             string(TypeWithdrawal),
		EntityID:         sql.NullInt64{Int64: 33, Valid: true},
		RecipientAddress: "bc1qrecipient",
		TransactionHash:  repository.StringToNullable("tx-hash"),
		Blockchain:       "BTC",
		CurrencyType:     string(money.Token),
		Currency:         "BTC_OLD",
		Decimals:         8,
		Amount:           repository.MoneyToNumeric(money.MustCryptoFromRaw("BTC_OLD", "50000000", 8)),
		NetworkFee:       repository.MoneyToNumeric(money.MustCryptoFromRaw("BTC", "1234", 8)),
		ServiceFee:       repository.MoneyToNumeric(money.MustCryptoFromRaw("BTC_OLD", "1000", 8)),
		UsdAmount:        repository.MoneyToNumeric(mustUSD(t, 100)),
		Metadata:         pgtype.JSONB{Status: pgtype.Null},
		NetworkID:        repository.StringToNullable("mainnet"),
		IsTest:           false,
		NetworkDecimals:  8,
	}

	tx, err := svc.entryToTransaction(entry)
	require.NoError(t, err)
	require.NotNil(t, tx.NetworkFee)

	assert.Equal(t, "BTC", tx.NetworkFee.Ticker())
	assert.Equal(t, "0.00001234", tx.NetworkFee.String())
}

func TestNetworkCurrencyForBalanceUpdate_FallsBackForUnsupportedNativeCoin(t *testing.T) {
	legacyToken := money.CryptoCurrency{
		Blockchain:     money.Blockchain("BTC"),
		BlockchainName: "Bitcoin",
		NetworkID:      "mainnet",
		Type:           money.Token,
		Ticker:         "BTC_OLD",
		Name:           "BTC_OLD",
		Decimals:       8,
	}
	networkFee := money.MustCryptoFromRaw("BTC", "1234", 8)

	svc := &Service{
		blockchain: &resolverStub{
			nativeCoinErr: blockchain.ErrCurrencyNotFound,
		},
	}

	currency, err := svc.networkCurrencyForBalanceUpdate(&Transaction{
		Currency:   legacyToken,
		NetworkFee: &networkFee,
	})
	require.NoError(t, err)

	assert.Equal(t, money.Blockchain("BTC"), currency.Blockchain)
	assert.Equal(t, "Bitcoin", currency.BlockchainName)
	assert.Equal(t, money.Coin, currency.Type)
	assert.Equal(t, "BTC", currency.Ticker)
	assert.Equal(t, "BTC", currency.Name)
	assert.Equal(t, int64(8), currency.Decimals)
	assert.Equal(t, "mainnet", currency.NetworkID)

	stub := svc.blockchain.(*resolverStub)
	assert.Equal(t, money.Blockchain("BTC"), stub.lastBlockchain)
}

func mustUSD(t *testing.T, amount float64) money.Money {
	t.Helper()

	m, err := money.FiatFromFloat64(money.USD, amount)
	require.NoError(t, err)

	return m
}
