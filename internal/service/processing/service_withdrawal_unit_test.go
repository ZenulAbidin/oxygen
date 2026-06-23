package processing

import (
	"testing"

	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/service/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectUTXOConsolidationCandidatesStopsWhenTargetIsCovered(t *testing.T) {
	target := money.MustCryptoFromRaw("BTC", "100", 8)
	candidates := []utxoWithdrawalSource{
		utxoWithdrawalCandidate("BTC", "70"),
		utxoWithdrawalCandidate("BTC", "40"),
		utxoWithdrawalCandidate("BTC", "30"),
	}

	selected, err := selectUTXOConsolidationCandidates(candidates, target)

	require.NoError(t, err)
	assert.Len(t, selected, 2)
}

func TestSelectUTXOConsolidationCandidatesAppliesPerWithdrawalCap(t *testing.T) {
	target := money.MustCryptoFromRaw("BTC", "100000", 8)
	candidates := make([]utxoWithdrawalSource, 0, maxUTXOWithdrawalConsolidationSources+3)
	for i := 0; i < maxUTXOWithdrawalConsolidationSources+3; i++ {
		candidates = append(candidates, utxoWithdrawalCandidate("BTC", "1"))
	}

	selected, err := selectUTXOConsolidationCandidates(candidates, target)

	require.NoError(t, err)
	assert.Len(t, selected, maxUTXOWithdrawalConsolidationSources)
}

func utxoWithdrawalCandidate(ticker, rawAmount string) utxoWithdrawalSource {
	return utxoWithdrawalSource{
		Wallet: &wallet.Wallet{},
		Balance: &wallet.Balance{
			Currency: ticker,
			Amount:   money.MustCryptoFromRaw(ticker, rawAmount, 8),
		},
	}
}
