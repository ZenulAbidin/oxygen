package fakes

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/service/blockchain"
	"github.com/pkg/errors"
)

type Tracker struct {
	t        *testing.T
	mu       sync.RWMutex
	incoming map[string][]blockchain.IncomingTransaction
}

func newTracker(t *testing.T) *Tracker {
	return &Tracker{
		t:        t,
		incoming: make(map[string][]blockchain.IncomingTransaction),
	}
}

func (m *Tracker) ListIncomingTransactions(
	_ context.Context,
	recipient string,
	currency money.CryptoCurrency,
	isTest bool,
) ([]blockchain.IncomingTransaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.key(recipient, currency, isTest)
	result, exists := m.incoming[key]
	if !exists {
		return nil, errors.New("unexpected call of (*TrackerMock).ListIncomingTransactions with args " + key)
	}

	return result, nil
}

func (m *Tracker) SetupListIncomingTransactions(
	recipient string,
	currency money.CryptoCurrency,
	isTest bool,
	txs []blockchain.IncomingTransaction,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.incoming[m.key(recipient, currency, isTest)] = txs
}

func (m *Tracker) key(recipient string, currency money.CryptoCurrency, isTest bool) string {
	return fmt.Sprintf("%s/%s/%t", recipient, currency.Ticker, isTest)
}
