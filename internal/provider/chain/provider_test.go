package chain

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	kms "github.com/oxygenpay/oxygen/internal/kms/wallet"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAddsDefaultBitcoinFallbacksOnlyForDefaultPrimary(t *testing.T) {
	logger := zerolog.New(io.Discard)

	defaultProvider := New(Config{}, &logger)
	assert.Equal(t, defaultBitcoinMainnetExplorerURL, defaultProvider.config.BitcoinMainnetExplorerURL)
	assert.Equal(t, []string{defaultBitcoinMainnetFallbackURL}, defaultProvider.config.BitcoinMainnetFallbackURLs)
	assert.Equal(t, []string{defaultBitcoinTestnetFallbackURL}, defaultProvider.config.BitcoinTestnetFallbackURLs)

	privateProvider := New(Config{BitcoinMainnetExplorerURL: "https://btc.local/api"}, &logger)
	assert.Equal(t, "https://btc.local/api", privateProvider.config.BitcoinMainnetExplorerURL)
	assert.Empty(t, privateProvider.config.BitcoinMainnetFallbackURLs)

	explicitFallbackProvider := New(Config{
		BitcoinMainnetExplorerURL:  "https://btc.local/api",
		BitcoinMainnetFallbackURLs: []string{"https://btc-backup.local/api/"},
		BitcoinTestnetExplorerURL:  "https://btc-test.local/api",
		BitcoinTestnetFallbackURLs: []string{"https://btc-test-backup.local/api/"},
	}, &logger)
	assert.Equal(t, []string{"https://btc-backup.local/api"}, explicitFallbackProvider.config.BitcoinMainnetFallbackURLs)
	assert.Equal(t, []string{"https://btc-test-backup.local/api"}, explicitFallbackProvider.config.BitcoinTestnetFallbackURLs)
}

func TestUTXOAddressTransactionsFallsBackWhenPrimaryIsRateLimited(t *testing.T) {
	logger := zerolog.New(io.Discard)

	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer primary.Close()

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/address/bc1qrecipient/txs/mempool", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"txid":"fallback-tx","vout":[{"scriptpubkey_address":"bc1qrecipient","value":1538}]}]`))
	}))
	defer fallback.Close()

	provider := New(Config{
		BitcoinMainnetExplorerURL:  primary.URL,
		BitcoinMainnetFallbackURLs: []string{fallback.URL},
		BitcoinTestnetExplorerURL:  primary.URL,
		BitcoinTestnetFallbackURLs: nil,
		LitecoinMainnetExplorerURL: primary.URL,
		LitecoinTestnetExplorerURL: primary.URL,
	}, &logger)

	txs, err := provider.UTXOAddressTransactions(context.Background(), kms.BTC, "bc1qrecipient", false, true)

	require.NoError(t, err)
	require.Len(t, txs, 1)
	assert.Equal(t, "fallback-tx", txs[0].TxID)
}

func TestUTXOAddressTransactionsCachesSuccessfulGETs(t *testing.T) {
	logger := zerolog.New(io.Discard)
	hits := 0

	explorer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"txid":"cached-tx"}]`))
	}))
	defer explorer.Close()

	provider := New(Config{
		BitcoinMainnetExplorerURL:  explorer.URL,
		BitcoinTestnetExplorerURL:  explorer.URL,
		LitecoinMainnetExplorerURL: explorer.URL,
		LitecoinTestnetExplorerURL: explorer.URL,
	}, &logger)

	for i := 0; i < 2; i++ {
		txs, err := provider.UTXOAddressTransactions(context.Background(), kms.BTC, "bc1qrecipient", false, false)
		require.NoError(t, err)
		require.Len(t, txs, 1)
		assert.Equal(t, "cached-tx", txs[0].TxID)
	}

	assert.Equal(t, 1, hits)
}

func TestUTXOFeeEstimates(t *testing.T) {
	logger := zerolog.New(io.Discard)

	explorer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/fee-estimates", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"1":1.5,"144":0.1}`))
	}))
	defer explorer.Close()

	provider := New(Config{
		BitcoinMainnetExplorerURL:  explorer.URL,
		BitcoinTestnetExplorerURL:  explorer.URL,
		LitecoinMainnetExplorerURL: explorer.URL,
		LitecoinTestnetExplorerURL: explorer.URL,
	}, &logger)

	estimates, err := provider.UTXOFeeEstimates(context.Background(), kms.BTC, false)

	require.NoError(t, err)
	assert.Equal(t, 1.5, estimates["1"])
	assert.Equal(t, 0.1, estimates["144"])
}
