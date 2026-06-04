package fakes

import (
	"testing"

	"github.com/oxygenpay/oxygen/internal/service/blockchain"
)

// Fakes global faker struct. Supported mocks:
// - Most of blockchain.Service
// - bus.PubSub
type Fakes struct {
	*Broadcaster
	*FeeCalculator
	*ConvertorProxy
	*Tracker
	*blockchain.CurrencyResolver
	*Bus
}

// New Fakes constructor
func New(t *testing.T, blockchainService *blockchain.Service) *Fakes {
	return &Fakes{
		Broadcaster:      newBroadcaster(t),
		FeeCalculator:    newFeeCalculator(t),
		ConvertorProxy:   newConvertorProxy(blockchainService),
		Tracker:          newTracker(t),
		CurrencyResolver: blockchainService.CurrencyResolver,
		Bus:              &Bus{},
	}
}
