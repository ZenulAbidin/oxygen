package fakes

import (
	"context"

	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/service/blockchain"
)

// ConvertorProxy represents proxy for real implementation of blockchain service that uses
// fake tatum http server that can we mocked as well.
type ConvertorProxy struct {
	conv *blockchain.Service
}

func newConvertorProxy(conv *blockchain.Service) *ConvertorProxy {
	return &ConvertorProxy{conv}
}

func (c *ConvertorProxy) GetExchangeRate(ctx context.Context, from, to string) (blockchain.ExchangeRate, error) {
	return c.conv.GetExchangeRate(ctx, from, to)
}

func (c *ConvertorProxy) Convert(ctx context.Context, from, to, amount string) (blockchain.Conversion, error) {
	return c.conv.Convert(ctx, from, to, amount)
}

func (c *ConvertorProxy) FiatToFiat(ctx context.Context, from money.Money, to money.FiatCurrency) (blockchain.Conversion, error) {
	return c.conv.FiatToFiat(ctx, from, to)
}

func (c *ConvertorProxy) FiatToCrypto(ctx context.Context, from money.Money, to money.CryptoCurrency) (blockchain.Conversion, error) {
	return c.conv.FiatToCrypto(ctx, from, to)
}

func (c *ConvertorProxy) CryptoToFiat(ctx context.Context, from money.Money, to money.FiatCurrency) (blockchain.Conversion, error) {
	return c.conv.CryptoToFiat(ctx, from, to)
}

func (c *ConvertorProxy) PrepareBitcoinTransaction(
	ctx context.Context,
	senderAddress string,
	recipient string,
	amount money.Money,
	fee blockchain.Fee,
	isTest bool,
) (blockchain.BitcoinTransactionPlan, error) {
	return c.conv.PrepareBitcoinTransaction(ctx, senderAddress, recipient, amount, fee, isTest)
}

func (c *ConvertorProxy) PrepareBitcoinTransactionExcluding(
	ctx context.Context,
	senderAddress string,
	recipient string,
	amount money.Money,
	fee blockchain.Fee,
	isTest bool,
	excluded []blockchain.BitcoinUTXOKey,
) (blockchain.BitcoinTransactionPlan, error) {
	return c.conv.PrepareBitcoinTransactionExcluding(ctx, senderAddress, recipient, amount, fee, isTest, excluded)
}

func (c *ConvertorProxy) PrepareBitcoinSweepTransactionExcluding(
	ctx context.Context,
	senderAddress string,
	recipient string,
	fee blockchain.Fee,
	isTest bool,
	excluded []blockchain.BitcoinUTXOKey,
) (blockchain.BitcoinTransactionPlan, error) {
	return c.conv.PrepareBitcoinSweepTransactionExcluding(ctx, senderAddress, recipient, fee, isTest, excluded)
}

func (c *ConvertorProxy) MaxBitcoinTransactionAmountExcluding(
	ctx context.Context,
	senderAddress string,
	recipient string,
	currency money.CryptoCurrency,
	fee blockchain.Fee,
	isTest bool,
	maxTotalCost money.Money,
	excluded []blockchain.BitcoinUTXOKey,
) (money.Money, error) {
	return c.conv.MaxBitcoinTransactionAmountExcluding(ctx, senderAddress, recipient, currency, fee, isTest, maxTotalCost, excluded)
}

func (c *ConvertorProxy) SpendableUTXOsExcluding(
	ctx context.Context,
	senderAddress string,
	fee blockchain.Fee,
	isTest bool,
	excluded []blockchain.BitcoinUTXOKey,
) ([]blockchain.BitcoinUTXO, error) {
	return c.conv.SpendableUTXOsExcluding(ctx, senderAddress, fee, isTest, excluded)
}
