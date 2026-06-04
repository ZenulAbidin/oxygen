package wallet_test

import (
	"testing"

	"github.com/oxygenpay/oxygen/internal/service/wallet"
	"github.com/oxygenpay/oxygen/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcquireLockSkipsLegacyBitcoinWallets(t *testing.T) {
	tc := test.NewIntegrationTest(t)

	user, _ := tc.Must.CreateSampleUser(t)
	merchant, _ := tc.Must.CreateMerchant(t, user.ID)
	btc := tc.Must.GetCurrency(t, "BTC")

	legacy := tc.Must.CreateWallet(t, "BTC", "1A8Po3UQaibD733LjxLCe4ydtcq2RmDNEc", "legacy-pub-key", wallet.TypeInbound)
	native := tc.Must.CreateWallet(t, "BTC", "bc1q4lv4th9eca7wlrqjs7uz6dqzz9jltrpkm2jryy", "native-pub-key", wallet.TypeInbound)

	acquired, err := tc.Services.Wallet.AcquireLock(tc.Context, merchant.ID, btc, false)
	require.NoError(t, err)

	assert.NotEqual(t, legacy.ID, acquired.ID)
	assert.Equal(t, native.ID, acquired.ID)
	assert.Equal(t, native.Address, acquired.Address)
}

func TestAcquireFreshLockNeverReusesExistingInboundWallet(t *testing.T) {
	tc := test.NewIntegrationTest(t)

	user, _ := tc.Must.CreateSampleUser(t)
	merchant, _ := tc.Must.CreateMerchant(t, user.ID)
	eth := tc.Must.GetCurrency(t, "ETH")

	existing := tc.Must.CreateWallet(t, "ETH", "0x-existing-inbound", "existing-pub-key", wallet.TypeInbound)
	tc.SetupCreateWallet("ETH", "0x-fresh-inbound", "fresh-pub-key")

	acquired, err := tc.Services.Wallet.AcquireFreshLock(tc.Context, merchant.ID, eth, false)
	require.NoError(t, err)

	assert.NotEqual(t, existing.ID, acquired.ID)
	assert.NotEqual(t, existing.Address, acquired.Address)
	assert.Equal(t, "0x-fresh-inbound", acquired.Address)
	tc.AssertTableRows(t, "wallets", 2)
	tc.AssertTableRowsByMerchant(t, merchant.ID, "wallet_locks", 1)
}

func TestEnsureOutboundWalletRejectsLegacyBitcoinWallet(t *testing.T) {
	tc := test.NewIntegrationTest(t)

	legacy := tc.Must.CreateWallet(t, "BTC", "1A8Po3UQaibD733LjxLCe4ydtcq2RmDNEc", "legacy-pub-key", wallet.TypeOutbound)

	acquired, created, err := tc.Services.Wallet.EnsureOutboundWallet(tc.Context, legacy.Blockchain)

	require.ErrorIs(t, err, wallet.ErrUnsupportedAddressFormat)
	assert.Nil(t, acquired)
	assert.False(t, created)
}
