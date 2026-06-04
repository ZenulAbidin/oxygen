package processing

import (
	"context"
	"testing"

	kmswallet "github.com/oxygenpay/oxygen/internal/kms/wallet"
	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/service/blockchain"
	"github.com/oxygenpay/oxygen/internal/service/merchant"
	"github.com/oxygenpay/oxygen/internal/service/payment"
	"github.com/oxygenpay/oxygen/internal/service/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type runtimeGuardBlockchainStub struct {
	baseCurrency money.CryptoCurrency
	currency     money.CryptoCurrency

	getCurrencyCalls               int
	getNativeCoinCalls             int
	calculateFeeCalls              int
	calculateWithdrawalFeeUSDCalls int

	lastTicker          string
	lastNativeCoinChain money.Blockchain
}

func (s *runtimeGuardBlockchainStub) ListSupportedCurrencies(bool) []money.CryptoCurrency {
	panic("unexpected call")
}

func (s *runtimeGuardBlockchainStub) ListBlockchainCurrencies(money.Blockchain) []money.CryptoCurrency {
	panic("unexpected call")
}

func (s *runtimeGuardBlockchainStub) GetCurrencyByTicker(ticker string) (money.CryptoCurrency, error) {
	s.getCurrencyCalls++
	s.lastTicker = ticker

	return s.currency, nil
}

func (s *runtimeGuardBlockchainStub) GetNativeCoin(blockchain money.Blockchain) (money.CryptoCurrency, error) {
	s.getNativeCoinCalls++
	s.lastNativeCoinChain = blockchain

	return s.baseCurrency, nil
}

func (s *runtimeGuardBlockchainStub) GetCurrencyByBlockchainAndContract(money.Blockchain, string, string) (money.CryptoCurrency, error) {
	panic("unexpected call")
}

func (s *runtimeGuardBlockchainStub) GetMinimalWithdrawalByTicker(string) (money.Money, error) {
	panic("unexpected call")
}

func (s *runtimeGuardBlockchainStub) GetUSDMinimalInternalTransferByTicker(string) (money.Money, error) {
	panic("unexpected call")
}

func (s *runtimeGuardBlockchainStub) BroadcastTransaction(context.Context, money.Blockchain, string, bool) (string, error) {
	panic("unexpected call")
}

func (s *runtimeGuardBlockchainStub) GetTransactionReceipt(context.Context, money.Blockchain, string, bool) (*blockchain.TransactionReceipt, error) {
	panic("unexpected call")
}

func (s *runtimeGuardBlockchainStub) ListIncomingTransactions(
	context.Context,
	string,
	money.CryptoCurrency,
	bool,
) ([]blockchain.IncomingTransaction, error) {
	panic("unexpected call")
}

func (s *runtimeGuardBlockchainStub) CalculateFee(context.Context, money.CryptoCurrency, money.CryptoCurrency, bool) (blockchain.Fee, error) {
	s.calculateFeeCalls++
	panic("unexpected call")
}

func (s *runtimeGuardBlockchainStub) CalculateWithdrawalFeeUSD(context.Context, money.CryptoCurrency, money.CryptoCurrency, bool) (money.Money, error) {
	s.calculateWithdrawalFeeUSDCalls++
	panic("unexpected call")
}

func (s *runtimeGuardBlockchainStub) GetExchangeRate(context.Context, string, string) (blockchain.ExchangeRate, error) {
	panic("unexpected call")
}

func (s *runtimeGuardBlockchainStub) Convert(context.Context, string, string, string) (blockchain.Conversion, error) {
	panic("unexpected call")
}

func (s *runtimeGuardBlockchainStub) FiatToFiat(context.Context, money.Money, money.FiatCurrency) (blockchain.Conversion, error) {
	panic("unexpected call")
}

func (s *runtimeGuardBlockchainStub) FiatToCrypto(context.Context, money.Money, money.CryptoCurrency) (blockchain.Conversion, error) {
	panic("unexpected call")
}

func (s *runtimeGuardBlockchainStub) CryptoToFiat(context.Context, money.Money, money.FiatCurrency) (blockchain.Conversion, error) {
	panic("unexpected call")
}

func TestCreateInternalTransfer_RejectsUnsupportedRuntimeBeforeFeeCalculation(t *testing.T) {
	doge := money.CryptoCurrency{
		Blockchain:     money.Blockchain("DOGE"),
		BlockchainName: "Dogecoin",
		NetworkID:      "mainnet",
		TestNetworkID:  "testnet",
		Type:           money.Coin,
		Ticker:         "DOGE",
		Name:           "DOGE",
		Decimals:       8,
	}

	blockchainStub := &runtimeGuardBlockchainStub{
		baseCurrency: doge,
		currency:     doge,
	}
	service := &Service{blockchain: blockchainStub}

	amount := money.MustCryptoFromRaw("DOGE", "100000000", 8)
	sender := &wallet.Wallet{Blockchain: kmswallet.Blockchain("DOGE")}

	out, err := service.createInternalTransfer(context.Background(), sender, internalTransferInput{
		SenderWallet:    sender,
		SenderBalance:   &wallet.Balance{NetworkID: doge.NetworkID},
		RecipientWallet: &wallet.Wallet{Address: "doge-recipient"},
		Amount:          amount,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, blockchain.ErrUnsupportedRuntime)
	assert.ErrorContains(t, err, "internal transfer")
	assert.Equal(t, 1, blockchainStub.getNativeCoinCalls)
	assert.Equal(t, 1, blockchainStub.getCurrencyCalls)
	assert.Equal(t, money.Blockchain("DOGE"), blockchainStub.lastNativeCoinChain)
	assert.Equal(t, "DOGE", blockchainStub.lastTicker)
	assert.Zero(t, blockchainStub.calculateFeeCalls)
	assert.Zero(t, blockchainStub.calculateWithdrawalFeeUSDCalls)
	assert.Equal(t, internalTransferOutput{}, out)
}

func TestCreateWithdrawal_RejectsUnsupportedRuntimeAndMarksPaymentFailed(t *testing.T) {
	doge := money.CryptoCurrency{
		Blockchain:     money.Blockchain("DOGE"),
		BlockchainName: "Dogecoin",
		NetworkID:      "mainnet",
		TestNetworkID:  "testnet",
		Type:           money.Coin,
		Ticker:         "DOGE",
		Name:           "DOGE",
		Decimals:       8,
	}

	blockchainStub := &runtimeGuardBlockchainStub{
		baseCurrency: doge,
		currency:     doge,
	}
	service := &Service{blockchain: blockchainStub}

	amount := money.MustCryptoFromRaw("DOGE", "100000000", 8)

	out, err := service.createWithdrawal(context.Background(), withdrawalInput{
		Withdrawal: &payment.Payment{Price: amount},
		MerchantBalance: &wallet.Balance{
			Network:   doge.Blockchain.String(),
			NetworkID: doge.NetworkID,
			Currency:  doge.Ticker,
		},
		MerchantAddress: &merchant.Address{Address: "doge-recipient"},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, blockchain.ErrUnsupportedRuntime)
	assert.ErrorContains(t, err, "withdrawal")
	assert.Equal(t, 1, blockchainStub.getNativeCoinCalls)
	assert.Equal(t, 1, blockchainStub.getCurrencyCalls)
	assert.Equal(t, money.Blockchain("DOGE"), blockchainStub.lastNativeCoinChain)
	assert.Equal(t, "DOGE", blockchainStub.lastTicker)
	assert.Zero(t, blockchainStub.calculateFeeCalls)
	assert.Zero(t, blockchainStub.calculateWithdrawalFeeUSDCalls)
	assert.True(t, out.MarkPaymentAsFailed)
	assert.Empty(t, out.TransactionRaw)
	assert.Nil(t, out.Transaction)
	assert.False(t, out.BalanceDecremented)
}
