package wallet

import (
	"io"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/google/uuid"
	"github.com/wemeetagain/go-hdwallet"
)

type BitcoinProvider struct {
	Blockchain   Blockchain
	CryptoReader io.Reader
}

var bitcoinNetworks = []*chaincfg.Params{
	&chaincfg.MainNetParams,
	&chaincfg.TestNet3Params,
}

var litecoinMainNetParams = chaincfg.Params{
	Name:             "litecoin-mainnet",
	Net:              wire.BitcoinNet(0xdbb6c0fb),
	Bech32HRPSegwit:  "ltc",
	PubKeyHashAddrID: 0x30,
	ScriptHashAddrID: 0x32,
	PrivateKeyID:     0xb0,
	HDPrivateKeyID:   [4]byte{0x04, 0x88, 0xad, 0xe4},
	HDPublicKeyID:    [4]byte{0x04, 0x88, 0xb2, 0x1e},
	HDCoinType:       2,
}

var litecoinTestNetParams = chaincfg.Params{
	Name:             "litecoin-testnet",
	Net:              wire.BitcoinNet(0xf1c8d2fd),
	Bech32HRPSegwit:  "tltc",
	PubKeyHashAddrID: 0x6f,
	ScriptHashAddrID: 0x3a,
	PrivateKeyID:     0xef,
	HDPrivateKeyID:   [4]byte{0x04, 0x35, 0x83, 0x94},
	HDPublicKeyID:    [4]byte{0x04, 0x35, 0x87, 0xcf},
	HDCoinType:       1,
}

var litecoinNetworks = []*chaincfg.Params{
	&litecoinMainNetParams,
	&litecoinTestNetParams,
}

func init() {
	for _, network := range litecoinNetworks {
		if err := chaincfg.Register(network); err != nil && err != chaincfg.ErrDuplicateNet {
			panic(err)
		}
	}
}

func (p *BitcoinProvider) Generate() *Wallet {
	seed := make([]byte, 256)
	if _, err := io.ReadFull(p.CryptoReader, seed); err != nil {
		return &Wallet{}
	}

	privateKey := hdwallet.MasterKey(seed)
	publicKey := privateKey.Pub()
	address, err := nativeSegWitAddressForBlockchain(p.blockchain(), publicKey, false)
	if err != nil {
		return &Wallet{}
	}

	return &Wallet{
		UUID:       uuid.New(),
		CreatedAt:  time.Now(),
		Blockchain: p.Blockchain,
		Address:    address,
		PublicKey:  publicKey.String(),
		PrivateKey: privateKey.String(),
	}
}

func (p *BitcoinProvider) GetBlockchain() Blockchain {
	return p.blockchain()
}

func (p *BitcoinProvider) ValidateAddress(address string) bool {
	return validateUTXOAddress(p.blockchain(), address)
}

func (p *BitcoinProvider) blockchain() Blockchain {
	if p.Blockchain == "" {
		return BTC
	}

	return p.Blockchain
}

func nativeSegWitAddress(publicKey *hdwallet.HDWallet, isTest bool) (string, error) {
	return nativeSegWitAddressForBlockchain(BTC, publicKey, isTest)
}

func nativeSegWitAddressForBlockchain(blockchain Blockchain, publicKey *hdwallet.HDWallet, isTest bool) (string, error) {
	witnessProgram := btcutil.Hash160(publicKey.Key)
	address, err := btcutil.NewAddressWitnessPubKeyHash(witnessProgram, utxoNetwork(blockchain, isTest))
	if err != nil {
		return "", err
	}

	return address.EncodeAddress(), nil
}

func validateBitcoinAddress(address string) bool {
	return validateUTXOAddress(BTC, address)
}

func validateBitcoinAddressForNetwork(address string, isTest bool) bool {
	return validateUTXOAddressForNetwork(BTC, address, isTest)
}

func validateUTXOAddress(blockchain Blockchain, address string) bool {
	for _, network := range utxoNetworks(blockchain) {
		if isNativeSegWitAddressForNetwork(address, network) {
			return true
		}
	}

	return false
}

func validateUTXOAddressForNetwork(blockchain Blockchain, address string, isTest bool) bool {
	return isNativeSegWitAddressForNetwork(address, utxoNetwork(blockchain, isTest))
}

func validateBitcoinAddressForNetworks(address string, networks ...*chaincfg.Params) bool {
	for _, network := range networks {
		if isNativeSegWitAddressForNetwork(address, network) {
			return true
		}
	}

	return false
}

func isNativeSegWitAddressForNetwork(address string, network *chaincfg.Params) bool {
	decoded, err := btcutil.DecodeAddress(address, network)
	if err != nil || !decoded.IsForNet(network) {
		return false
	}

	// btcutil also accepts raw pubkeys; re-encoding keeps validation limited to payment addresses.
	if !strings.EqualFold(decoded.EncodeAddress(), address) {
		return false
	}

	_, ok := decoded.(*btcutil.AddressWitnessPubKeyHash)
	return ok
}

func bitcoinNetwork(isTest bool) *chaincfg.Params {
	return utxoNetwork(BTC, isTest)
}

func utxoNetwork(blockchain Blockchain, isTest bool) *chaincfg.Params {
	switch blockchain {
	case LTC:
		if isTest {
			return &litecoinTestNetParams
		}

		return &litecoinMainNetParams
	default:
		if isTest {
			return &chaincfg.TestNet3Params
		}

		return &chaincfg.MainNetParams
	}
}

func utxoNetworks(blockchain Blockchain) []*chaincfg.Params {
	switch blockchain {
	case LTC:
		return litecoinNetworks
	default:
		return bitcoinNetworks
	}
}

func isUTXOBlockchain(blockchain Blockchain) bool {
	switch blockchain {
	case BTC, LTC:
		return true
	default:
		return false
	}
}
