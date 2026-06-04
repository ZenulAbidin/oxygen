package processing

import (
	kmswallet "github.com/oxygenpay/oxygen/internal/kms/wallet"
	"github.com/oxygenpay/oxygen/internal/money"
)

func isUTXOBlockchain(blockchain money.Blockchain) bool {
	switch kmswallet.Blockchain(blockchain) {
	case kmswallet.BTC, kmswallet.LTC:
		return true
	default:
		return false
	}
}
