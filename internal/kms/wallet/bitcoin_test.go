package wallet_test

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/oxygenpay/oxygen/internal/kms/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemeetagain/go-hdwallet"
)

func TestBitcoinProvider_Generate(t *testing.T) {
	const (
		mockAddress    = "bc1q82mck8gdey7wx78wtyl364v9gd7gyhg7pdsd7n"
		mockPubKey     = "xpub661MyMwAqRbcGRWWcY4qYw2QpdFnfQz31BLdkDqXvac9Cp4zk5J4NEqssAd3CEfSSUEsh183dN93xzPhbnprMKaq9E5BVkagZLFnqTVPBCy"
		mockPrivateKey = "xprv9s21ZrQH143K3wS3WWXqBo5gGbRJFxGBdxR2wqRvNF5AL1jrCXyopSXQ1tduFqKjJq4CbP3dPMH48JtKhMtm7zNLytntFN8NRsGaYJwJ3Ku"
	)

	p := &wallet.BitcoinProvider{
		Blockchain:   wallet.BTC,
		CryptoReader: &fakeReader{},
	}

	t.Run("Mock_GenerationSuccessful", func(t *testing.T) {
		w := p.Generate()

		assert.Equal(t, mockAddress, w.Address)
		assert.Equal(t, mockPubKey, w.PublicKey)
		assert.Equal(t, mockPrivateKey, w.PrivateKey)
		assert.Regexp(t, "^bc1q", w.Address)
	})

	t.Run("Mock_PrivateKeyAsStringToPublicKey", func(t *testing.T) {
		w := p.Generate()

		key, err := hdwallet.StringWallet(w.PrivateKey)
		require.NoError(t, err)

		publicKey := key.Pub().String()
		assert.Equal(t, publicKey, w.PublicKey)
	})

	t.Run("Mock_PrivateKeyAsStringToAddress", func(t *testing.T) {
		w := p.Generate()

		key, err := hdwallet.StringWallet(w.PrivateKey)
		require.NoError(t, err)

		address := nativeSegWitAddress(t, key)
		assert.Equal(t, address, w.Address)
	})

	t.Run("Real_GenerationSuccessful", func(t *testing.T) {
		p := &wallet.BitcoinProvider{
			Blockchain:   wallet.BTC,
			CryptoReader: cryptorand.Reader,
		}

		w := p.Generate()

		key, err := hdwallet.StringWallet(w.PrivateKey)
		require.NoError(t, err)

		publicKey := key.Pub().String()
		address := nativeSegWitAddress(t, key)

		assert.Equal(t, publicKey, w.PublicKey)
		assert.Equal(t, address, w.Address)
		assert.Regexp(t, "^bc1q", w.Address)
	})
}

func TestBitcoinProvider_GenerateLitecoinNativeSegWit(t *testing.T) {
	p := &wallet.BitcoinProvider{
		Blockchain:   wallet.LTC,
		CryptoReader: &fakeReader{},
	}

	w := p.Generate()

	assert.Equal(t, wallet.LTC, w.Blockchain)
	assert.Regexp(t, "^ltc1q", w.Address)
	assert.True(t, p.ValidateAddress(w.Address))
	assert.NoError(t, wallet.ValidateAddressForNetwork(wallet.LTC, w.Address, false))
	assert.ErrorIs(t, wallet.ValidateAddressForNetwork(wallet.BTC, w.Address, false), wallet.ErrInvalidAddress)
}

func TestBitcoinProvider_NewTransaction_NativeSegWit(t *testing.T) {
	p := &wallet.BitcoinProvider{
		Blockchain:   wallet.BTC,
		CryptoReader: &fakeReader{},
	}
	w := p.Generate()

	rawTX, err := p.NewTransaction(w, wallet.BitcoinTransactionParams{
		Inputs: []wallet.BitcoinUTXO{{
			Hash:       "1c99a4e6f1dcf8ef2e3db3d2db5ca7dd4f78edb5a8f42fb6f7f4d89af2b2b923",
			Index:      0,
			AmountSats: 10_000,
			Address:    w.Address,
		}},
		Outputs: []wallet.BitcoinOutput{{
			Address:    w.Address,
			AmountSats: 8_000,
		}},
		RBF: true,
	})
	require.NoError(t, err)
	require.NotEmpty(t, rawTX)

	rawBytes, err := hex.DecodeString(rawTX)
	require.NoError(t, err)

	var tx wire.MsgTx
	require.NoError(t, tx.Deserialize(bytes.NewReader(rawBytes)))

	require.Len(t, tx.TxIn, 1)
	require.Len(t, tx.TxOut, 1)
	assert.Equal(t, int32(2), tx.Version)
	assert.Equal(t, int64(8_000), tx.TxOut[0].Value)
	assert.Equal(t, uint32(0xfffffffd), tx.TxIn[0].Sequence)
	assert.Len(t, tx.TxIn[0].Witness, 2)
}

func TestBitcoinProvider_NewTransaction_LitecoinNativeSegWit(t *testing.T) {
	p := &wallet.BitcoinProvider{
		Blockchain:   wallet.LTC,
		CryptoReader: &fakeReader{},
	}
	w := p.Generate()

	rawTX, err := p.NewTransaction(w, wallet.BitcoinTransactionParams{
		Inputs: []wallet.BitcoinUTXO{{
			Hash:       "1c99a4e6f1dcf8ef2e3db3d2db5ca7dd4f78edb5a8f42fb6f7f4d89af2b2b923",
			Index:      0,
			AmountSats: 10_000,
			Address:    w.Address,
		}},
		Outputs: []wallet.BitcoinOutput{{
			Address:    w.Address,
			AmountSats: 8_000,
		}},
		RBF: true,
	})
	require.NoError(t, err)
	require.NotEmpty(t, rawTX)

	rawBytes, err := hex.DecodeString(rawTX)
	require.NoError(t, err)

	var tx wire.MsgTx
	require.NoError(t, tx.Deserialize(bytes.NewReader(rawBytes)))

	require.Len(t, tx.TxIn, 1)
	require.Len(t, tx.TxOut, 1)
	assert.Equal(t, int32(2), tx.Version)
	assert.Equal(t, int64(8_000), tx.TxOut[0].Value)
	assert.Equal(t, uint32(0xfffffffd), tx.TxIn[0].Sequence)
	assert.Len(t, tx.TxIn[0].Witness, 2)
}

func nativeSegWitAddress(t *testing.T, key *hdwallet.HDWallet) string {
	t.Helper()

	witnessProgram := btcutil.Hash160(key.Pub().Key)
	address, err := btcutil.NewAddressWitnessPubKeyHash(witnessProgram, &chaincfg.MainNetParams)
	require.NoError(t, err)

	return address.EncodeAddress()
}

func TestBitcoinProvider_ValidateAddress(t *testing.T) {
	p := &wallet.BitcoinProvider{}

	for _, tc := range []struct {
		addr          string
		expectInvalid bool
	}{
		{addr: "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq"},
		{addr: "bc1ql7c7u74ht6j02wt56csd43wfsnv5949xqwkx7h"},
		{addr: "37fiwTokZXVyao1iugda5cGAmkzfYAwNYW", expectInvalid: true},
		{addr: "1LQoWist8KkaUXSPKZHNvEyfrEkPHzSsCd", expectInvalid: true},
		{addr: "1FeexV6bAHb8ybZjqQMjJrcCrHGW9sb6uF", expectInvalid: true},
		{addr: "tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx"},
		{addr: "mipcBbFg9gMiCh81Kj8tqqdgoZub1ZJRfn", expectInvalid: true},
		{addr: "2NBFNJTktNa7GZusGbDbGKRZTxdK9VVez3n", expectInvalid: true},
		{addr: "2FeexV6bAHb8ybZjqQMjJrcCrHGW9sb6uF", expectInvalid: true},
		{addr: "1FeexV6bAHb8ybZjqQMjJrcCrHGW9sb6uF_", expectInvalid: true},
		{addr: "1FeexV6bAHb8ybZjqQMjJrcCrHGW9sb6uF_", expectInvalid: true},
		{addr: "1FeexV6bAHb8ybZjqQMjJH", expectInvalid: true},
		{addr: "wtf", expectInvalid: true},
	} {
		t.Run(tc.addr, func(t *testing.T) {
			assert.Equal(t, !tc.expectInvalid, p.ValidateAddress(tc.addr))
		})
	}
}

func TestValidateAddressForNetwork_Bitcoin(t *testing.T) {
	for _, tc := range []struct {
		name          string
		addr          string
		isTest        bool
		expectInvalid bool
	}{
		{
			name:   "mainnet address on mainnet",
			addr:   "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq",
			isTest: false,
		},
		{
			name:          "mainnet address on testnet",
			addr:          "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq",
			isTest:        true,
			expectInvalid: true,
		},
		{
			name:   "testnet bech32 on testnet",
			addr:   "tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx",
			isTest: true,
		},
		{
			name:          "testnet bech32 on mainnet",
			addr:          "tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx",
			isTest:        false,
			expectInvalid: true,
		},
		{
			name:          "testnet base58 on testnet",
			addr:          "mipcBbFg9gMiCh81Kj8tqqdgoZub1ZJRfn",
			isTest:        true,
			expectInvalid: true,
		},
		{
			name:          "testnet base58 on mainnet",
			addr:          "mipcBbFg9gMiCh81Kj8tqqdgoZub1ZJRfn",
			isTest:        false,
			expectInvalid: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := wallet.ValidateAddressForNetwork(wallet.BTC, tc.addr, tc.isTest)
			if tc.expectInvalid {
				assert.ErrorIs(t, err, wallet.ErrInvalidAddress)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestValidateAddressForNetwork_Litecoin(t *testing.T) {
	p := &wallet.BitcoinProvider{
		Blockchain:   wallet.LTC,
		CryptoReader: &fakeReader{},
	}
	mainnetWallet := p.Generate()

	for _, tc := range []struct {
		name          string
		addr          string
		isTest        bool
		expectInvalid bool
	}{
		{
			name:   "mainnet address on mainnet",
			addr:   mainnetWallet.Address,
			isTest: false,
		},
		{
			name:          "mainnet address on testnet",
			addr:          mainnetWallet.Address,
			isTest:        true,
			expectInvalid: true,
		},
		{
			name:   "testnet bech32 on testnet",
			addr:   "tltc1qyw3c0rvn6kk2c688y3dygvckn57525y8emy4p5",
			isTest: true,
		},
		{
			name:          "testnet bech32 on mainnet",
			addr:          "tltc1qyw3c0rvn6kk2c688y3dygvckn57525y8emy4p5",
			isTest:        false,
			expectInvalid: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := wallet.ValidateAddressForNetwork(wallet.LTC, tc.addr, tc.isTest)
			if tc.expectInvalid {
				assert.ErrorIs(t, err, wallet.ErrInvalidAddress)
				return
			}

			assert.NoError(t, err)
		})
	}
}
