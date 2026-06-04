package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	kms "github.com/oxygenpay/oxygen/internal/kms/wallet"
	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

const (
	defaultBitcoinMainnetExplorerURL  = "https://blockstream.info/api"
	defaultBitcoinTestnetExplorerURL  = "https://blockstream.info/testnet/api"
	defaultBitcoinMainnetFallbackURL  = "https://mempool.space/api"
	defaultBitcoinTestnetFallbackURL  = "https://mempool.space/testnet/api"
	defaultLitecoinMainnetExplorerURL = "https://litecoinspace.org/api"
	defaultLitecoinTestnetExplorerURL = "https://litecoinspace.org/testnet/api"
	defaultEVMScanBlocks              = int64(1200)
	getCacheTTL                       = 5 * time.Second
	transientFailureCooldown          = 10 * time.Second
	rateLimitCooldown                 = time.Minute
)

type Config struct {
	BitcoinMainnetExplorerURL   string   `yaml:"bitcoin_mainnet_explorer_url" env:"CHAIN_BITCOIN_MAINNET_EXPLORER_URL" env-description:"Blockstream-compatible Bitcoin mainnet API base URL"`
	BitcoinMainnetFallbackURLs  []string `yaml:"bitcoin_mainnet_explorer_fallback_urls" env:"CHAIN_BITCOIN_MAINNET_EXPLORER_FALLBACK_URLS" env-description:"Comma separated list of fallback Blockstream-compatible Bitcoin mainnet API base URLs"`
	BitcoinTestnetExplorerURL   string   `yaml:"bitcoin_testnet_explorer_url" env:"CHAIN_BITCOIN_TESTNET_EXPLORER_URL" env-description:"Blockstream-compatible Bitcoin testnet API base URL"`
	BitcoinTestnetFallbackURLs  []string `yaml:"bitcoin_testnet_explorer_fallback_urls" env:"CHAIN_BITCOIN_TESTNET_EXPLORER_FALLBACK_URLS" env-description:"Comma separated list of fallback Blockstream-compatible Bitcoin testnet API base URLs"`
	LitecoinMainnetExplorerURL  string   `yaml:"litecoin_mainnet_explorer_url" env:"CHAIN_LITECOIN_MAINNET_EXPLORER_URL" env-description:"Blockstream-compatible Litecoin mainnet API base URL"`
	LitecoinMainnetFallbackURLs []string `yaml:"litecoin_mainnet_explorer_fallback_urls" env:"CHAIN_LITECOIN_MAINNET_EXPLORER_FALLBACK_URLS" env-description:"Comma separated list of fallback Blockstream-compatible Litecoin mainnet API base URLs"`
	LitecoinTestnetExplorerURL  string   `yaml:"litecoin_testnet_explorer_url" env:"CHAIN_LITECOIN_TESTNET_EXPLORER_URL" env-description:"Blockstream-compatible Litecoin testnet API base URL"`
	LitecoinTestnetFallbackURLs []string `yaml:"litecoin_testnet_explorer_fallback_urls" env:"CHAIN_LITECOIN_TESTNET_EXPLORER_FALLBACK_URLS" env-description:"Comma separated list of fallback Blockstream-compatible Litecoin testnet API base URLs"`

	EthereumMainnetRPCURL string `yaml:"ethereum_mainnet_rpc_url" env:"CHAIN_ETHEREUM_MAINNET_RPC_URL" env-description:"Ethereum mainnet JSON-RPC URL for payment tracking"`
	EthereumTestnetRPCURL string `yaml:"ethereum_testnet_rpc_url" env:"CHAIN_ETHEREUM_TESTNET_RPC_URL" env-description:"Ethereum testnet JSON-RPC URL for payment tracking"`
	MaticMainnetRPCURL    string `yaml:"matic_mainnet_rpc_url" env:"CHAIN_MATIC_MAINNET_RPC_URL" env-description:"Polygon mainnet JSON-RPC URL for payment tracking"`
	MaticTestnetRPCURL    string `yaml:"matic_testnet_rpc_url" env:"CHAIN_MATIC_TESTNET_RPC_URL" env-description:"Polygon testnet JSON-RPC URL for payment tracking"`
	BSCMainnetRPCURL      string `yaml:"bsc_mainnet_rpc_url" env:"CHAIN_BSC_MAINNET_RPC_URL" env-description:"BNB Chain mainnet JSON-RPC URL for payment tracking"`
	BSCTestnetRPCURL      string `yaml:"bsc_testnet_rpc_url" env:"CHAIN_BSC_TESTNET_RPC_URL" env-description:"BNB Chain testnet JSON-RPC URL for payment tracking"`

	EVMScanBlocks int64 `yaml:"evm_scan_blocks" env:"CHAIN_EVM_SCAN_BLOCKS" env-description:"Recent EVM blocks to scan for native coin incoming payments"`
}

type Provider struct {
	config Config
	client http.Client
	logger *zerolog.Logger

	mu          sync.Mutex
	getCache    map[string]cachedResponse
	getCooldown map[string]time.Time
}

type cachedResponse struct {
	body      []byte
	expiresAt time.Time
}

func New(cfg Config, logger *zerolog.Logger) *Provider {
	useDefaultBitcoinMainnetFallback := usesDefaultURL(cfg.BitcoinMainnetExplorerURL, defaultBitcoinMainnetExplorerURL)
	useDefaultBitcoinTestnetFallback := usesDefaultURL(cfg.BitcoinTestnetExplorerURL, defaultBitcoinTestnetExplorerURL)

	cfg.BitcoinMainnetExplorerURL = normalizeBaseURL(cfg.BitcoinMainnetExplorerURL, defaultBitcoinMainnetExplorerURL)
	cfg.BitcoinTestnetExplorerURL = normalizeBaseURL(cfg.BitcoinTestnetExplorerURL, defaultBitcoinTestnetExplorerURL)
	cfg.LitecoinMainnetExplorerURL = normalizeBaseURL(cfg.LitecoinMainnetExplorerURL, defaultLitecoinMainnetExplorerURL)
	cfg.LitecoinTestnetExplorerURL = normalizeBaseURL(cfg.LitecoinTestnetExplorerURL, defaultLitecoinTestnetExplorerURL)
	cfg.BitcoinMainnetFallbackURLs = normalizeFallbackURLs(
		cfg.BitcoinMainnetFallbackURLs,
		defaultFallbackURLs(useDefaultBitcoinMainnetFallback, defaultBitcoinMainnetFallbackURL),
		cfg.BitcoinMainnetExplorerURL,
	)
	cfg.BitcoinTestnetFallbackURLs = normalizeFallbackURLs(
		cfg.BitcoinTestnetFallbackURLs,
		defaultFallbackURLs(useDefaultBitcoinTestnetFallback, defaultBitcoinTestnetFallbackURL),
		cfg.BitcoinTestnetExplorerURL,
	)
	cfg.LitecoinMainnetFallbackURLs = normalizeFallbackURLs(
		cfg.LitecoinMainnetFallbackURLs,
		nil,
		cfg.LitecoinMainnetExplorerURL,
	)
	cfg.LitecoinTestnetFallbackURLs = normalizeFallbackURLs(
		cfg.LitecoinTestnetFallbackURLs,
		nil,
		cfg.LitecoinTestnetExplorerURL,
	)
	if cfg.EVMScanBlocks <= 0 {
		cfg.EVMScanBlocks = defaultEVMScanBlocks
	}

	log := logger.With().Str("channel", "chain_provider").Logger()

	return &Provider{
		config:      cfg,
		client:      http.Client{Timeout: 10 * time.Second},
		logger:      &log,
		getCache:    make(map[string]cachedResponse),
		getCooldown: make(map[string]time.Time),
	}
}

func (p *Provider) EVMScanBlocks() int64 {
	return p.config.EVMScanBlocks
}

func (p *Provider) EVMRPCURL(blockchain money.Blockchain, isTest bool) string {
	switch kms.Blockchain(blockchain) {
	case kms.ETH:
		if isTest {
			return p.config.EthereumTestnetRPCURL
		}
		return p.config.EthereumMainnetRPCURL
	case kms.MATIC:
		if isTest {
			return p.config.MaticTestnetRPCURL
		}
		return p.config.MaticMainnetRPCURL
	case kms.BSC:
		if isTest {
			return p.config.BSCTestnetRPCURL
		}
		return p.config.BSCMainnetRPCURL
	default:
		return ""
	}
}

func (p *Provider) EVMRPC(ctx context.Context, blockchain money.Blockchain, isTest bool) (*ethclient.Client, error) {
	url := p.EVMRPCURL(blockchain, isTest)
	if url == "" {
		return nil, errors.Errorf("RPC URL is not configured for %s test=%t", blockchain.String(), isTest)
	}

	return ethclient.DialContext(ctx, url)
}

func (p *Provider) BitcoinAddressTransactions(
	ctx context.Context,
	address string,
	isTest bool,
	mempool bool,
) ([]BitcoinTransaction, error) {
	return p.UTXOAddressTransactions(ctx, kms.BTC, address, isTest, mempool)
}

func (p *Provider) LitecoinAddressTransactions(
	ctx context.Context,
	address string,
	isTest bool,
	mempool bool,
) ([]BitcoinTransaction, error) {
	return p.UTXOAddressTransactions(ctx, kms.LTC, address, isTest, mempool)
}

func (p *Provider) UTXOAddressTransactions(
	ctx context.Context,
	blockchain kms.Blockchain,
	address string,
	isTest bool,
	mempool bool,
) ([]BitcoinTransaction, error) {
	path := fmt.Sprintf("/address/%s/txs", address)
	if mempool {
		path += "/mempool"
	}

	var out []BitcoinTransaction
	if err := p.getJSONFromURLs(ctx, p.utxoURLs(blockchain, path, isTest), &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (p *Provider) UTXOAddressUTXOs(
	ctx context.Context,
	blockchain kms.Blockchain,
	address string,
	isTest bool,
) ([]BitcoinUTXO, error) {
	var out []BitcoinUTXO
	if err := p.getJSONFromURLs(ctx, p.utxoURLs(blockchain, "/address/"+address+"/utxo", isTest), &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (p *Provider) BitcoinTransaction(ctx context.Context, txID string, isTest bool) (BitcoinTransaction, error) {
	return p.UTXOTransaction(ctx, kms.BTC, txID, isTest)
}

func (p *Provider) LitecoinTransaction(ctx context.Context, txID string, isTest bool) (BitcoinTransaction, error) {
	return p.UTXOTransaction(ctx, kms.LTC, txID, isTest)
}

func (p *Provider) UTXOTransaction(ctx context.Context, blockchain kms.Blockchain, txID string, isTest bool) (BitcoinTransaction, error) {
	var out BitcoinTransaction
	if err := p.getJSONFromURLs(ctx, p.utxoURLs(blockchain, "/tx/"+txID, isTest), &out); err != nil {
		return BitcoinTransaction{}, err
	}

	return out, nil
}

func (p *Provider) BitcoinTipHeight(ctx context.Context, isTest bool) (int64, error) {
	return p.UTXOTipHeight(ctx, kms.BTC, isTest)
}

func (p *Provider) LitecoinTipHeight(ctx context.Context, isTest bool) (int64, error) {
	return p.UTXOTipHeight(ctx, kms.LTC, isTest)
}

func (p *Provider) UTXOTipHeight(ctx context.Context, blockchain kms.Blockchain, isTest bool) (int64, error) {
	raw, err := p.getRawFromURLs(ctx, p.utxoURLs(blockchain, "/blocks/tip/height", isTest))
	if err != nil {
		return 0, err
	}

	var height int64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(raw)), "%d", &height); err != nil {
		return 0, errors.Wrapf(err, "unable to parse %s tip height", blockchain)
	}

	return height, nil
}

func (p *Provider) BroadcastUTXOTransaction(
	ctx context.Context,
	blockchain kms.Blockchain,
	rawTX string,
	isTest bool,
) (string, error) {
	res, err := p.postText(ctx, p.utxoURL(blockchain, "/tx", isTest), rawTX)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(raw)), nil
}

func (p *Provider) bitcoinURL(path string, isTest bool) string {
	return p.utxoURL(kms.BTC, path, isTest)
}

func (p *Provider) utxoURL(blockchain kms.Blockchain, path string, isTest bool) string {
	urls := p.utxoURLs(blockchain, path, isTest)
	if len(urls) == 0 {
		return ""
	}

	return urls[0]
}

func (p *Provider) utxoURLs(blockchain kms.Blockchain, path string, isTest bool) []string {
	bases := []string{p.config.BitcoinMainnetExplorerURL}

	switch blockchain {
	case kms.LTC:
		bases = []string{p.config.LitecoinMainnetExplorerURL}
		if isTest {
			bases = []string{p.config.LitecoinTestnetExplorerURL}
			bases = append(bases, p.config.LitecoinTestnetFallbackURLs...)
		} else {
			bases = append(bases, p.config.LitecoinMainnetFallbackURLs...)
		}
	default:
		if isTest {
			bases = []string{p.config.BitcoinTestnetExplorerURL}
			bases = append(bases, p.config.BitcoinTestnetFallbackURLs...)
		} else {
			bases = append(bases, p.config.BitcoinMainnetFallbackURLs...)
		}
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	urls := make([]string, 0, len(bases))
	for _, base := range bases {
		if base == "" {
			continue
		}
		urls = append(urls, base+path)
	}

	return urls
}

func (p *Provider) getJSONFromURLs(ctx context.Context, urls []string, out any) error {
	raw, err := p.getRawFromURLs(ctx, urls)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return errors.Wrap(err, "unable to decode response")
	}

	return nil
}

func (p *Provider) getRawFromURLs(ctx context.Context, urls []string) ([]byte, error) {
	var lastErr error
	for idx, url := range urls {
		raw, err := p.getRaw(ctx, url)
		if err == nil {
			if idx > 0 {
				p.logger.Info().
					Str("url", url).
					Msg("chain provider fallback request succeeded")
			}
			return raw, nil
		}

		lastErr = err
		if idx < len(urls)-1 {
			p.logger.Warn().
				Err(err).
				Str("url", url).
				Str("fallback_url", urls[idx+1]).
				Msg("chain provider request failed, trying fallback")
		}
	}

	if lastErr == nil {
		lastErr = errors.New("chain provider endpoint list is empty")
	}

	return nil, lastErr
}

func (p *Provider) getJSON(ctx context.Context, url string, out any) error {
	raw, err := p.getRaw(ctx, url)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return errors.Wrap(err, "unable to decode response")
	}

	return nil
}

func (p *Provider) getRaw(ctx context.Context, url string) ([]byte, error) {
	if body, ok := p.cachedGet(url); ok {
		return body, nil
	}

	if until, ok := p.cooldownUntil(url); ok {
		return nil, errors.Errorf("chain provider request skipped during cooldown until %s", until.Format(time.RFC3339))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	res, err := p.client.Do(req)
	if err != nil {
		p.setCooldown(url, transientFailureCooldown)
		return nil, err
	}
	defer res.Body.Close()

	raw, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		p.setCooldown(url, transientFailureCooldown)
		return nil, readErr
	}

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		p.cacheGet(url, raw)
		return raw, nil
	}

	if res.StatusCode == http.StatusTooManyRequests {
		p.setCooldown(url, rateLimitCooldown)
	} else if res.StatusCode >= 500 {
		p.setCooldown(url, transientFailureCooldown)
	}

	p.logger.Warn().
		Str("url", url).
		Int("status_code", res.StatusCode).
		Str("response", string(raw)).
		Msg("chain provider request failed")

	return nil, errors.Errorf("chain provider request failed with %d", res.StatusCode)
}

func (p *Provider) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	res, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return res, nil
	}

	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)
	p.logger.Warn().
		Str("url", url).
		Int("status_code", res.StatusCode).
		Str("response", string(raw)).
		Msg("chain provider request failed")

	return nil, errors.Errorf("chain provider request failed with %d", res.StatusCode)
}

func (p *Provider) postText(ctx context.Context, url string, body string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/plain")

	res, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return res, nil
	}

	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)
	p.logger.Warn().
		Str("url", url).
		Int("status_code", res.StatusCode).
		Str("response", string(raw)).
		Msg("chain provider request failed")

	return nil, errors.Errorf("chain provider request failed with %d: %s", res.StatusCode, strings.TrimSpace(string(raw)))
}

func (p *Provider) cachedGet(url string) ([]byte, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cached, ok := p.getCache[url]
	if !ok {
		return nil, false
	}
	if time.Now().After(cached.expiresAt) {
		delete(p.getCache, url)
		return nil, false
	}

	body := append([]byte(nil), cached.body...)
	return body, true
}

func (p *Provider) cacheGet(url string, body []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.getCache[url] = cachedResponse{
		body:      append([]byte(nil), body...),
		expiresAt: time.Now().Add(getCacheTTL),
	}
	delete(p.getCooldown, url)
}

func (p *Provider) cooldownUntil(url string) (time.Time, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	until, ok := p.getCooldown[url]
	if !ok {
		return time.Time{}, false
	}
	if time.Now().After(until) {
		delete(p.getCooldown, url)
		return time.Time{}, false
	}

	return until, true
}

func (p *Provider) setCooldown(url string, ttl time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.getCooldown[url] = time.Now().Add(ttl)
}

func normalizeBaseURL(value, fallback string) string {
	if value == "" {
		value = fallback
	}

	return strings.TrimRight(value, "/")
}

func usesDefaultURL(value, fallback string) bool {
	return normalizeBaseURL(value, fallback) == strings.TrimRight(fallback, "/")
}

func defaultFallbackURLs(enabled bool, urls ...string) []string {
	if !enabled {
		return nil
	}

	return urls
}

func normalizeFallbackURLs(values, defaults []string, excluded ...string) []string {
	all := make([]string, 0, len(values)+len(defaults))
	all = append(all, values...)
	all = append(all, defaults...)

	excludedSet := make(map[string]struct{}, len(excluded))
	for _, value := range excluded {
		value = normalizeBaseURL(value, "")
		if value != "" {
			excludedSet[value] = struct{}{}
		}
	}

	out := make([]string, 0, len(all))
	seen := make(map[string]struct{}, len(all))
	for _, value := range all {
		value = normalizeBaseURL(value, "")
		if value == "" {
			continue
		}
		if _, ok := excludedSet[value]; ok {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		out = append(out, value)
	}

	return out
}

type BitcoinTransaction struct {
	TxID    string          `json:"txid"`
	Version int64           `json:"version"`
	Lock    int64           `json:"locktime"`
	Vin     []BitcoinInput  `json:"vin"`
	Vout    []BitcoinOutput `json:"vout"`
	Size    int64           `json:"size"`
	Weight  int64           `json:"weight"`
	Fee     int64           `json:"fee"`
	Status  BitcoinStatus   `json:"status"`
}

type BitcoinStatus struct {
	Confirmed   bool   `json:"confirmed"`
	BlockHeight int64  `json:"block_height"`
	BlockHash   string `json:"block_hash"`
	BlockTime   int64  `json:"block_time"`
}

type BitcoinUTXO struct {
	TxID   string        `json:"txid"`
	Vout   uint32        `json:"vout"`
	Status BitcoinStatus `json:"status"`
	Value  int64         `json:"value"`
}

type BitcoinInput struct {
	TxID    string         `json:"txid"`
	Vout    int64          `json:"vout"`
	PrevOut *BitcoinOutput `json:"prevout"`
}

type BitcoinOutput struct {
	ScriptPubKeyAddress string `json:"scriptpubkey_address"`
	Value               int64  `json:"value"`
}
