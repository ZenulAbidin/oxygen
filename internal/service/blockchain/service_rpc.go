package blockchain

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
	kms "github.com/oxygenpay/oxygen/internal/kms/wallet"
	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/pkg/errors"
)

func (s *Service) evmRPC(ctx context.Context, blockchain money.Blockchain, isTest bool) (*ethclient.Client, error) {
	if s.providers.Chain != nil && s.providers.Chain.EVMRPCURL(blockchain, isTest) != "" {
		return s.providers.Chain.EVMRPC(ctx, blockchain, isTest)
	}

	if s.providers.Tatum == nil || !s.providers.Tatum.HasAPIKey(isTest) {
		return nil, errors.Errorf("RPC URL is not configured for %s test=%t", blockchain.String(), isTest)
	}

	switch kms.Blockchain(blockchain) {
	case kms.ETH:
		return s.providers.Tatum.EthereumRPC(ctx, isTest)
	case kms.MATIC:
		return s.providers.Tatum.MaticRPC(ctx, isTest)
	case kms.BSC:
		return s.providers.Tatum.BinanceSmartChainRPC(ctx, isTest)
	default:
		return nil, errors.Wrapf(ErrUnsupportedRuntime, "RPC is not implemented for %q", blockchain)
	}
}
