package internalapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/service/blockchain"
	admin "github.com/oxygenpay/oxygen/pkg/api-admin/v1/model"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockchainServiceStub struct {
	currency           money.CryptoCurrency
	currencyErr        error
	nativeCoin         money.CryptoCurrency
	nativeCoinErr      error
	fee                blockchain.Fee
	feeErr             error
	broadcastTxHash    string
	broadcastErr       error
	receipt            *blockchain.TransactionReceipt
	err                error
	getCurrencyCalls   int
	getNativeCoinCalls int
	calculateFeeCalls  int
	broadcastCalls     int

	getTransactionReceiptCalls int
	lastBlockchain             money.Blockchain
	lastTxID                   string
	lastIsTest                 bool

	lastTicker             string
	lastNativeCoinChain    money.Blockchain
	lastFeeBaseCurrency    money.CryptoCurrency
	lastFeeCurrency        money.CryptoCurrency
	lastCalculateFeeIsTest bool
	lastBroadcastHex       string
}

func (s *blockchainServiceStub) ListSupportedCurrencies(bool) []money.CryptoCurrency {
	panic("unexpected call")
}

func (s *blockchainServiceStub) ListBlockchainCurrencies(money.Blockchain) []money.CryptoCurrency {
	panic("unexpected call")
}

func (s *blockchainServiceStub) GetCurrencyByTicker(ticker string) (money.CryptoCurrency, error) {
	s.getCurrencyCalls++
	s.lastTicker = ticker

	return s.currency, s.currencyErr
}

func (s *blockchainServiceStub) GetNativeCoin(blockchain money.Blockchain) (money.CryptoCurrency, error) {
	s.getNativeCoinCalls++
	s.lastNativeCoinChain = blockchain

	return s.nativeCoin, s.nativeCoinErr
}

func (s *blockchainServiceStub) GetCurrencyByBlockchainAndContract(money.Blockchain, string, string) (money.CryptoCurrency, error) {
	panic("unexpected call")
}

func (s *blockchainServiceStub) GetMinimalWithdrawalByTicker(string) (money.Money, error) {
	panic("unexpected call")
}

func (s *blockchainServiceStub) GetUSDMinimalInternalTransferByTicker(string) (money.Money, error) {
	panic("unexpected call")
}

func (s *blockchainServiceStub) BroadcastTransaction(
	_ context.Context,
	blockchain money.Blockchain,
	hex string,
	isTest bool,
) (string, error) {
	s.broadcastCalls++
	s.lastBlockchain = blockchain
	s.lastBroadcastHex = hex
	s.lastIsTest = isTest

	return s.broadcastTxHash, s.broadcastErr
}

func (s *blockchainServiceStub) GetTransactionReceipt(
	_ context.Context,
	blockchain money.Blockchain,
	transactionID string,
	isTest bool,
) (*blockchain.TransactionReceipt, error) {
	s.getTransactionReceiptCalls++
	s.lastBlockchain = blockchain
	s.lastTxID = transactionID
	s.lastIsTest = isTest

	return s.receipt, s.err
}

func (s *blockchainServiceStub) CalculateFee(
	_ context.Context,
	baseCurrency money.CryptoCurrency,
	currency money.CryptoCurrency,
	isTest bool,
) (blockchain.Fee, error) {
	s.calculateFeeCalls++
	s.lastFeeBaseCurrency = baseCurrency
	s.lastFeeCurrency = currency
	s.lastCalculateFeeIsTest = isTest

	return s.fee, s.feeErr
}

func (s *blockchainServiceStub) CalculateWithdrawalFeeUSD(context.Context, money.CryptoCurrency, money.CryptoCurrency, bool) (money.Money, error) {
	panic("unexpected call")
}

func TestBroadcastTransaction_UnsupportedRuntimeReturnsValidationError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(
		http.MethodPost,
		"/wallets/broadcast",
		strings.NewReader(`{"blockchain":"BTC","hex":"deadbeef","isTest":true}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	blockchainStub := &blockchainServiceStub{
		broadcastErr: errors.Wrap(blockchain.ErrUnsupportedRuntime, "BTC requires dedicated UTXO runtime support"),
	}
	handler := &Handler{blockchain: blockchainStub}

	err := handler.BroadcastTransaction(ctx)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, 1, blockchainStub.broadcastCalls)
	assert.Equal(t, money.Blockchain("BTC"), blockchainStub.lastBlockchain)
	assert.Equal(t, "deadbeef", blockchainStub.lastBroadcastHex)
	assert.True(t, blockchainStub.lastIsTest)

	var body admin.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Errors, 1)
	assert.Equal(t, "validation_error", body.Status)
	assert.Contains(t, body.Errors[0].Message, "UTXO runtime support")
}

func TestBroadcastTransaction_BroadcastErrorPreserved(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(
		http.MethodPost,
		"/wallets/broadcast",
		strings.NewReader(`{"blockchain":"ETH","hex":"deadbeef"}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	blockchainStub := &blockchainServiceStub{
		broadcastErr: errors.Wrap(blockchain.ErrInvalidTransaction, "Unable to broadcast transaction."),
	}
	handler := &Handler{blockchain: blockchainStub}

	err := handler.BroadcastTransaction(ctx)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, 1, blockchainStub.broadcastCalls)
	assert.Equal(t, money.Blockchain("ETH"), blockchainStub.lastBlockchain)
	assert.Equal(t, "deadbeef", blockchainStub.lastBroadcastHex)
	assert.False(t, blockchainStub.lastIsTest)

	var body admin.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "broadcast_error", body.Status)
	assert.Contains(t, body.Message, "Unable to broadcast transaction.")
}

func TestCalculateTransactionFee_UnsupportedCurrencyReturnsValidationError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/wallets/fees", strings.NewReader(`{"currency":"BTC"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	blockchainStub := &blockchainServiceStub{currencyErr: blockchain.ErrCurrencyNotFound}
	handler := &Handler{blockchain: blockchainStub}

	err := handler.CalculateTransactionFee(ctx)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, 1, blockchainStub.getCurrencyCalls)
	assert.Equal(t, "BTC", blockchainStub.lastTicker)
	assert.Zero(t, blockchainStub.getNativeCoinCalls)
	assert.Zero(t, blockchainStub.calculateFeeCalls)

	var body admin.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Errors, 1)
	assert.Equal(t, "validation_error", body.Status)
	assert.Equal(t, "unsupported currency", body.Errors[0].Message)
}

func TestCalculateTransactionFee_UnsupportedRuntimeReturnsValidationError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(
		http.MethodPost,
		"/wallets/fees",
		strings.NewReader(`{"currency":"BTC","isTest":true}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	btc := money.CryptoCurrency{
		Blockchain: money.Blockchain("BTC"),
		Ticker:     "BTC",
		Name:       "BTC",
		Type:       money.Coin,
	}
	blockchainStub := &blockchainServiceStub{
		currency:   btc,
		nativeCoin: btc,
		feeErr:     errors.Wrap(blockchain.ErrUnsupportedRuntime, "BTC requires dedicated UTXO runtime support"),
	}
	handler := &Handler{blockchain: blockchainStub}

	err := handler.CalculateTransactionFee(ctx)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, 1, blockchainStub.getCurrencyCalls)
	assert.Equal(t, 1, blockchainStub.getNativeCoinCalls)
	assert.Equal(t, 1, blockchainStub.calculateFeeCalls)
	assert.Equal(t, money.Blockchain("BTC"), blockchainStub.lastNativeCoinChain)
	assert.Equal(t, btc, blockchainStub.lastFeeBaseCurrency)
	assert.Equal(t, btc, blockchainStub.lastFeeCurrency)
	assert.True(t, blockchainStub.lastCalculateFeeIsTest)

	var body admin.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Errors, 1)
	assert.Equal(t, "validation_error", body.Status)
	assert.Contains(t, body.Errors[0].Message, "UTXO runtime support")
}

func TestCalculateTransactionFee_LitecoinReturnsUTXOFee(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(
		http.MethodPost,
		"/wallets/fees",
		strings.NewReader(`{"currency":"LTC","isTest":true}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	ltc := money.CryptoCurrency{
		Blockchain: money.Blockchain("LTC"),
		Ticker:     "LTC",
		Name:       "LTC",
		Type:       money.Coin,
	}
	blockchainStub := &blockchainServiceStub{
		currency:   ltc,
		nativeCoin: ltc,
		fee: blockchain.NewFee(ltc, time.Now(), true, blockchain.BitcoinFee{
			FeeSatPerVByte: "1",
			EstimatedBytes: "171",
			TotalCostSats:  "171",
			TotalCostBTC:   "0.00000171",
			TotalCostUSD:   "$0.01",
		}),
	}
	handler := &Handler{blockchain: blockchainStub}

	err := handler.CalculateTransactionFee(ctx)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, blockchainStub.getCurrencyCalls)
	assert.Equal(t, 1, blockchainStub.getNativeCoinCalls)
	assert.Equal(t, 1, blockchainStub.calculateFeeCalls)

	var body blockchain.BitcoinFee
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "1", body.FeeSatPerVByte)
	assert.Equal(t, "171", body.TotalCostSats)
}

func TestGetTransactionReceipt_RequiresTxID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/receipt?blockchain=ETH", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	blockchainStub := &blockchainServiceStub{}
	handler := &Handler{blockchain: blockchainStub}

	err := handler.GetTransactionReceipt(ctx)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Zero(t, blockchainStub.getTransactionReceiptCalls)

	var body admin.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Errors, 1)
	assert.Equal(t, "txId", body.Errors[0].Field)
	assert.Equal(t, "required", body.Errors[0].Message)
}

func TestGetTransactionReceipt_ReturnsReceipt(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/receipt?blockchain=ETH&txId=0xabc&isTest=true", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	networkFee := money.MustCryptoFromRaw("ETH", "21000000000000", 18)
	blockchainStub := &blockchainServiceStub{
		receipt: &blockchain.TransactionReceipt{
			Blockchain:    money.Blockchain("ETH"),
			IsTest:        true,
			Sender:        "0xsender",
			Recipient:     "0xrecipient",
			Hash:          "0xabc",
			Nonce:         7,
			NetworkFee:    networkFee,
			Success:       true,
			Confirmations: 12,
			IsConfirmed:   true,
		},
	}

	handler := &Handler{blockchain: blockchainStub}

	err := handler.GetTransactionReceipt(ctx)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, blockchainStub.getTransactionReceiptCalls)
	assert.Equal(t, money.Blockchain("ETH"), blockchainStub.lastBlockchain)
	assert.Equal(t, "0xabc", blockchainStub.lastTxID)
	assert.True(t, blockchainStub.lastIsTest)

	var body admin.TransactionReceiptResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "ETH", body.Blockchain)
	assert.Equal(t, "0xabc", body.TransactionHash)
	assert.Equal(t, int64(7), body.Nonce)
	assert.Equal(t, "0xsender", body.Sender)
	assert.Equal(t, "0xrecipient", body.Recipient)
	assert.Equal(t, networkFee.StringRaw(), body.NetworkFee)
	assert.Equal(t, networkFee.String(), body.NetworkFeeFormatted)
	assert.Equal(t, int64(12), body.Confirmations)
	assert.True(t, body.Success)
	assert.True(t, body.IsConfirmed)
	assert.True(t, body.IsTest)
}

func TestGetTransactionReceipt_UnsupportedRuntimeReturnsValidationError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/receipt?blockchain=BTC&txId=tx-id", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	blockchainStub := &blockchainServiceStub{
		err: errors.Wrap(blockchain.ErrUnsupportedRuntime, "BTC requires dedicated UTXO runtime support"),
	}
	handler := &Handler{blockchain: blockchainStub}

	err := handler.GetTransactionReceipt(ctx)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, 1, blockchainStub.getTransactionReceiptCalls)
	assert.Equal(t, money.Blockchain("BTC"), blockchainStub.lastBlockchain)
	assert.Equal(t, "tx-id", blockchainStub.lastTxID)
	assert.False(t, blockchainStub.lastIsTest)

	var body admin.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Errors, 1)
	assert.Equal(t, "validation_error", body.Status)
	assert.Contains(t, body.Errors[0].Message, "UTXO runtime support")
}
