package blockchain_test

import (
	"testing"

	kms "github.com/oxygenpay/oxygen/internal/kms/wallet"
	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/service/blockchain"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(_ *testing.T) {
	// todo returns sorted list
	// todo returns sorted by blockchain
	// todo add token to two different blockchain
}

func TestDefaultSetupIncludesBitcoin(t *testing.T) {
	currencies := blockchain.NewCurrencies()
	require.NoError(t, blockchain.DefaultSetup(currencies))

	btc, err := currencies.GetCurrencyByTicker("BTC")
	require.NoError(t, err)

	assert.Equal(t, money.Blockchain(kms.BTC), btc.Blockchain)
	assert.Equal(t, "Bitcoin", btc.BlockchainName)
	assert.Equal(t, "mainnet", btc.NetworkID)
	assert.Equal(t, "testnet", btc.TestNetworkID)
	assert.Equal(t, money.Coin, btc.Type)
	assert.Equal(t, "BTC", btc.Ticker)
	assert.Equal(t, "BTC", btc.Name)
	assert.Equal(t, int64(8), btc.Decimals)
	assert.Contains(t, currencies.ListSupportedBlockchains(), money.Blockchain(kms.BTC))

	native, err := currencies.GetNativeCoin(money.Blockchain(kms.BTC))
	require.NoError(t, err)
	assert.Equal(t, btc, native)

	minWithdrawal, err := currencies.GetMinimalWithdrawalByTicker("BTC")
	require.NoError(t, err)
	assert.Equal(t, "4000", minWithdrawal.StringRaw())
}

func TestDefaultSetupIncludesLitecoin(t *testing.T) {
	currencies := blockchain.NewCurrencies()
	require.NoError(t, blockchain.DefaultSetup(currencies))

	ltc, err := currencies.GetCurrencyByTicker("LTC")
	require.NoError(t, err)

	assert.Equal(t, money.Blockchain(kms.LTC), ltc.Blockchain)
	assert.Equal(t, "Litecoin", ltc.BlockchainName)
	assert.Equal(t, "mainnet", ltc.NetworkID)
	assert.Equal(t, "testnet", ltc.TestNetworkID)
	assert.Equal(t, money.Coin, ltc.Type)
	assert.Equal(t, "LTC", ltc.Ticker)
	assert.Equal(t, "LTC", ltc.Name)
	assert.Equal(t, int64(8), ltc.Decimals)
	assert.Contains(t, currencies.ListSupportedBlockchains(), money.Blockchain(kms.LTC))

	native, err := currencies.GetNativeCoin(money.Blockchain(kms.LTC))
	require.NoError(t, err)
	assert.Equal(t, ltc, native)

	minWithdrawal, err := currencies.GetMinimalWithdrawalByTicker("LTC")
	require.NoError(t, err)
	assert.Equal(t, "4000", minWithdrawal.StringRaw())
}

func TestCreatePaymentLink(t *testing.T) {
	currencies := blockchain.NewCurrencies()
	require.NoError(t, blockchain.DefaultSetup(currencies))

	const (
		evmAddr     = "0xc2132d05d31c914a87c6611c10748aeb04b58e8f"
		tronAddr    = "TVEaDaTKJZ2RsQUWREWykouuHak9scyZaf"
		btcAddr     = "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq"
		btcTestAddr = "tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx"
		ltcAddr     = "ltc1q43ugfc3tawhzvxgjrycq0papwmndlc7qc77pzz"
	)

	for _, tt := range []struct {
		address  string
		ticker   string
		currency money.CryptoCurrency
		amount   string
		isTest   bool
		expected string
	}{
		{
			address:  evmAddr,
			ticker:   "ETH",
			amount:   "123",
			isTest:   false,
			expected: "ethereum:0xc2132d05d31c914a87c6611c10748aeb04b58e8f@1?value=123",
		},
		{
			address:  evmAddr,
			ticker:   "ETH",
			amount:   "123",
			isTest:   true,
			expected: "ethereum:0xc2132d05d31c914a87c6611c10748aeb04b58e8f@5?value=123",
		},
		{
			address:  evmAddr,
			ticker:   "ETH_USDT",
			amount:   "333",
			isTest:   false,
			expected: "ethereum:0xdac17f958d2ee523a2206206994597c13d831ec7@1/transfer?address=0xc2132d05d31c914a87c6611c10748aeb04b58e8f&uint256=333",
		},
		{
			address:  evmAddr,
			ticker:   "ETH_USDT",
			amount:   "333",
			isTest:   true,
			expected: "ethereum:0xdac17f958d2ee523a2206206994597c13d831ec7@5/transfer?address=0xc2132d05d31c914a87c6611c10748aeb04b58e8f&uint256=333",
		},
		{
			address:  evmAddr,
			ticker:   "MATIC",
			amount:   "123",
			isTest:   true,
			expected: "ethereum:0xc2132d05d31c914a87c6611c10748aeb04b58e8f@80001?value=123",
		},
		{
			address:  evmAddr,
			ticker:   "MATIC_USDT",
			amount:   "333",
			isTest:   false,
			expected: "ethereum:0xc2132d05d31c914a87c6611c10748aeb04b58e8f@137/transfer?address=0xc2132d05d31c914a87c6611c10748aeb04b58e8f&uint256=333",
		},
		{
			address:  tronAddr,
			ticker:   "TRON",
			amount:   "444",
			isTest:   false,
			expected: "tron:TVEaDaTKJZ2RsQUWREWykouuHak9scyZaf?amount=0.000444",
		},
		{
			address:  tronAddr,
			ticker:   "TRON_USDT",
			amount:   "444",
			isTest:   true,
			expected: "tron:TVEaDaTKJZ2RsQUWREWykouuHak9scyZaf?amount=0.000444",
		},
		{
			address: btcAddr,
			currency: money.CryptoCurrency{
				Blockchain:     money.Blockchain(kms.BTC),
				BlockchainName: "Bitcoin",
				NetworkID:      "mainnet",
				TestNetworkID:  "testnet",
				Type:           money.Coin,
				Ticker:         "BTC",
				Name:           "BTC",
				Decimals:       8,
			},
			amount:   "12345678",
			isTest:   false,
			expected: "bitcoin:bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq?amount=0.12345678",
		},
		{
			address: btcTestAddr,
			currency: money.CryptoCurrency{
				Blockchain:     money.Blockchain(kms.BTC),
				BlockchainName: "Bitcoin",
				NetworkID:      "mainnet",
				TestNetworkID:  "testnet",
				Type:           money.Coin,
				Ticker:         "BTC",
				Name:           "BTC",
				Decimals:       8,
			},
			amount:   "123",
			isTest:   true,
			expected: "bitcoin:tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx?amount=0.00000123",
		},
		{
			address:  ltcAddr,
			ticker:   "LTC",
			amount:   "12345678",
			isTest:   false,
			expected: "litecoin:ltc1q43ugfc3tawhzvxgjrycq0papwmndlc7qc77pzz?amount=0.12345678",
		},
	} {
		t.Run(tt.expected, func(t *testing.T) {
			// ARRANGE
			currency := tt.currency
			if tt.ticker != "" {
				var err error
				currency, err = currencies.GetCurrencyByTicker(tt.ticker)
				require.NoError(t, err)
			}

			amount := lo.Must(currency.MakeAmount(tt.amount))

			// ACT
			actual, err := blockchain.CreatePaymentLink(tt.address, currency, amount, tt.isTest)

			// ASSERT
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestExplorerTXLink(t *testing.T) {
	btc := money.Blockchain("BTC")
	ltc := money.Blockchain("LTC")
	eth := money.Blockchain("ETH")
	matic := money.Blockchain("MATIC")
	tron := money.Blockchain("TRON")

	for _, tt := range []struct {
		blockchain  money.Blockchain
		networkID   string
		expectError bool
		expected    string
	}{
		{blockchain: btc, networkID: "mainnet", expected: "https://blockstream.info/tx/0x123"},
		{blockchain: btc, networkID: "testnet", expected: "https://blockstream.info/testnet/tx/0x123"},
		{blockchain: ltc, networkID: "mainnet", expected: "https://litecoinspace.org/tx/0x123"},
		{blockchain: ltc, networkID: "testnet", expected: "https://litecoinspace.org/testnet/tx/0x123"},
		{blockchain: eth, networkID: "1", expected: "https://etherscan.io/tx/0x123"},
		{blockchain: eth, networkID: "5", expected: "https://goerli.etherscan.io/tx/0x123"},
		{blockchain: matic, networkID: "137", expected: "https://polygonscan.com/tx/0x123"},
		{blockchain: matic, networkID: "80001", expected: "https://mumbai.polygonscan.com/tx/0x123"},
		{blockchain: tron, networkID: "mainnet", expected: "https://tronscan.org/#/transaction/0x123"},
		{blockchain: tron, networkID: "testnet", expected: "https://shasta.tronscan.org/#/transaction/0x123"},
		{blockchain: "abc", networkID: "1", expectError: true},
		{blockchain: matic, networkID: "1", expectError: true},
		{blockchain: tron, networkID: "1", expectError: true},
	} {
		t.Run(tt.expected, func(t *testing.T) {
			actual, err := blockchain.CreateExplorerTXLink(tt.blockchain, tt.networkID, "0x123")

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
