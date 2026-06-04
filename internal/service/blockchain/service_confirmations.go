package blockchain

import (
	kms "github.com/oxygenpay/oxygen/internal/kms/wallet"
	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/pkg/errors"
)

func RequiredConfirmations(blockchain money.Blockchain) (int64, error) {
	switch kms.Blockchain(blockchain) {
	case kms.BTC:
		return bitcoinRequiredConfirmations, nil
	case kms.LTC:
		return litecoinRequiredConfirmations, nil
	case kms.ETH:
		return ethRequiredConfirmations, nil
	case kms.MATIC:
		return maticRequiredConfirmations, nil
	case kms.BSC:
		return bscRequiredConfirmations, nil
	case kms.TRON:
		return tronRequiredConfirmations, nil
	default:
		return 0, errors.Wrapf(ErrUnsupportedRuntime, "unsupported blockchain %q", blockchain)
	}
}
