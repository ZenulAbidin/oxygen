package wallet

import (
	"bytes"
	"encoding/hex"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"github.com/wemeetagain/go-hdwallet"
)

const (
	bitcoinTxVersion      int32  = 2
	bitcoinRBFSequence    uint32 = 0xfffffffd
	bitcoinOutputDustSats int64  = 546
)

type BitcoinUTXO struct {
	Hash       string
	Index      uint32
	AmountSats int64
	Script     string
	Address    string
}

type BitcoinOutput struct {
	Address    string
	AmountSats int64
}

type BitcoinTransactionParams struct {
	Inputs  []BitcoinUTXO
	Outputs []BitcoinOutput
	IsTest  bool
	RBF     bool
}

func (p BitcoinTransactionParams) validate(blockchain Blockchain, walletAddress string) error {
	if len(p.Inputs) == 0 || len(p.Outputs) == 0 {
		return ErrInvalidAmount
	}

	if !isNativeSegWitAddress(blockchain, walletAddress, p.IsTest) {
		return errors.Wrapf(ErrInvalidAddress, "sender must be a native SegWit %s address", blockchain)
	}

	var inputTotal int64
	for _, input := range p.Inputs {
		if input.Hash == "" || input.AmountSats <= 0 {
			return ErrInvalidAmount
		}

		if input.Address != "" && input.Address != walletAddress {
			return errors.Wrap(ErrInvalidAddress, "input does not belong to sender")
		}

		inputTotal += input.AmountSats
	}

	var outputTotal int64
	for _, output := range p.Outputs {
		if output.AmountSats < bitcoinOutputDustSats {
			return ErrInvalidAmount
		}

		if err := ValidateAddressForNetwork(blockchain, output.Address, p.IsTest); err != nil {
			return err
		}

		outputTotal += output.AmountSats
	}

	if inputTotal <= outputTotal {
		return ErrInsufficientBalance
	}

	return nil
}

func (p *BitcoinProvider) NewTransaction(w *Wallet, params BitcoinTransactionParams) (string, error) {
	if w.Blockchain != p.Blockchain {
		return "", errors.Wrapf(
			ErrUnknownBlockchain,
			"This wallet (%s) doesn't support transactions for %s",
			w.Blockchain,
			p.Blockchain,
		)
	}

	if err := params.validate(p.Blockchain, w.Address); err != nil {
		return "", err
	}

	key, err := hdwallet.StringWallet(w.PrivateKey)
	if err != nil {
		return "", errors.Wrap(err, "unable to decode BTC xprv")
	}

	privateKeyBytes := key.Key
	if len(privateKeyBytes) == 33 && privateKeyBytes[0] == 0 {
		privateKeyBytes = privateKeyBytes[1:]
	}
	if len(privateKeyBytes) != 32 {
		return "", errors.New("invalid BTC private key length")
	}

	privateKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), privateKeyBytes)
	network := utxoNetwork(p.Blockchain, params.IsTest)

	tx := wire.NewMsgTx(bitcoinTxVersion)
	sequence := wire.MaxTxInSequenceNum
	if params.RBF {
		sequence = bitcoinRBFSequence
	}

	for _, input := range params.Inputs {
		hash, err := chainhash.NewHashFromStr(input.Hash)
		if err != nil {
			return "", errors.Wrap(err, "invalid BTC input hash")
		}

		txIn := wire.NewTxIn(wire.NewOutPoint(hash, input.Index), nil, nil)
		txIn.Sequence = sequence
		tx.AddTxIn(txIn)
	}

	for _, output := range params.Outputs {
		address, err := btcutil.DecodeAddress(output.Address, network)
		if err != nil || !address.IsForNet(network) {
			return "", ErrInvalidAddress
		}

		script, err := txscript.PayToAddrScript(address)
		if err != nil {
			return "", errors.Wrap(err, "unable to build BTC output script")
		}

		tx.AddTxOut(wire.NewTxOut(output.AmountSats, script))
	}

	scriptCode, err := p2wpkhScriptCode(p.Blockchain, key.Pub().Key, params.IsTest)
	if err != nil {
		return "", err
	}

	sigHashes := txscript.NewTxSigHashes(tx)
	for i, input := range params.Inputs {
		witness, err := txscript.WitnessSignature(
			tx,
			sigHashes,
			i,
			input.AmountSats,
			scriptCode,
			txscript.SigHashAll,
			privateKey,
			true,
		)
		if err != nil {
			return "", errors.Wrap(err, "unable to sign BTC input")
		}

		tx.TxIn[i].Witness = witness
	}

	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		return "", errors.Wrap(err, "unable to serialize BTC transaction")
	}

	return hex.EncodeToString(buf.Bytes()), nil
}

func p2wpkhScriptCode(blockchain Blockchain, compressedPublicKey []byte, isTest bool) ([]byte, error) {
	pubKeyHash := btcutil.Hash160(compressedPublicKey)
	address, err := btcutil.NewAddressPubKeyHash(pubKeyHash, utxoNetwork(blockchain, isTest))
	if err != nil {
		return nil, err
	}

	return txscript.PayToAddrScript(address)
}

func isNativeSegWitBitcoinAddress(address string, isTest bool) bool {
	return isNativeSegWitAddress(BTC, address, isTest)
}

func isNativeSegWitAddress(blockchain Blockchain, address string, isTest bool) bool {
	return isNativeSegWitAddressForNetwork(address, utxoNetwork(blockchain, isTest))
}
