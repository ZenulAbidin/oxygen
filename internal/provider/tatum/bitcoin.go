package tatum

import (
	"context"
	"net/http"

	"github.com/antihax/optional"
	client "github.com/oxygenpay/tatum-sdk/tatum"
)

func (p *Provider) bitcoinAPI(isTest bool) *client.APIClient {
	if isTest {
		return p.Test()
	}

	return p.Main()
}

func (p *Provider) BitcoinTransactionsByAddress(
	ctx context.Context,
	address string,
	pageSize float64,
	offset *float64,
	isTest bool,
) ([]client.BtcTx, *http.Response, error) {
	var opts *client.BitcoinApiBtcGetTxByAddressOpts
	if offset != nil {
		opts = &client.BitcoinApiBtcGetTxByAddressOpts{
			Offset: optional.NewFloat64(*offset),
		}
	}

	return p.bitcoinAPI(isTest).BitcoinApi.BtcGetTxByAddress(ctx, address, pageSize, opts)
}

func (p *Provider) BitcoinUTXO(ctx context.Context, hash string, index uint32, isTest bool) (
	client.BtcUtxo,
	*http.Response,
	error,
) {
	return p.bitcoinAPI(isTest).BitcoinApi.BtcGetUTXO(ctx, hash, float64(index))
}

func (p *Provider) BroadcastBitcoinTransaction(ctx context.Context, rawTX string, isTest bool) (
	client.TransactionHash,
	*http.Response,
	error,
) {
	return p.bitcoinAPI(isTest).BitcoinApi.BtcBroadcast(ctx, client.BroadcastKms{TxData: rawTX})
}

func (p *Provider) BitcoinTransaction(ctx context.Context, txHash string, isTest bool) (
	client.BtcTx,
	*http.Response,
	error,
) {
	return p.bitcoinAPI(isTest).BitcoinApi.BtcGetRawTransaction(ctx, txHash)
}

func (p *Provider) BitcoinBlockchainInfo(ctx context.Context, isTest bool) (
	client.BtcInfo,
	*http.Response,
	error,
) {
	return p.bitcoinAPI(isTest).BitcoinApi.BtcGetBlockChainInfo(ctx)
}
