package blockchain

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	kmswallet "github.com/oxygenpay/oxygen/internal/kms/wallet"
	"github.com/oxygenpay/oxygen/internal/money"
	client "github.com/oxygenpay/tatum-sdk/tatum"
	"github.com/pkg/errors"
)

const (
	bitcoinAddressTxPageSize = 50
	bitcoinMaxAddressTxPages = 10

	bitcoinMainnetFeeSatPerVByte  = int64(10)
	bitcoinTestnetFeeSatPerVByte  = int64(2)
	litecoinMainnetFeeSatPerVByte = int64(2)
	litecoinTestnetFeeSatPerVByte = int64(1)
	bitcoinDustSats               = int64(546)
	bitcoinRequiredConfirmations  = int64(6)
	litecoinRequiredConfirmations = int64(12)
	ethRequiredConfirmations      = int64(12)
	maticRequiredConfirmations    = int64(30)
	bscRequiredConfirmations      = int64(15)
	tronRequiredConfirmations     = int64(10)
)

type BitcoinUTXO struct {
	Hash        string
	Index       uint32
	AmountSats  int64
	Script      string
	Address     string
	BlockNumber int64
}

type BitcoinUTXOKey struct {
	Hash  string
	Index uint32
}

type BitcoinOutput struct {
	Address    string
	AmountSats int64
}

type BitcoinTransactionPlan struct {
	Inputs            []BitcoinUTXO
	Outputs           []BitcoinOutput
	ChangeAddress     string
	FeeSatPerVByte    int64
	EstimatedVBytes   int64
	FeeSats           int64
	RequiredAmountSat int64
	RBF               bool
}

type BitcoinFee struct {
	FeeSatPerVByte string `json:"feeSatPerVByte"`
	EstimatedBytes string `json:"estimatedBytes"`
	TotalCostSats  string `json:"totalCostSats"`
	TotalCostBTC   string `json:"totalCostBtc"`
	TotalCostUSD   string `json:"totalCostUsd"`

	totalCostUSD money.Money
}

func (f *Fee) ToBitcoinFee() (BitcoinFee, error) {
	if fee, ok := f.raw.(BitcoinFee); ok {
		return fee, nil
	}

	return BitcoinFee{}, errors.New("invalid fee type assertion for BTC")
}

func (s *Service) bitcoinFee(ctx context.Context, baseCurrency, currency money.CryptoCurrency, isTest bool) (Fee, error) {
	blockchain := kmswallet.Blockchain(currency.Blockchain)
	if !isUTXOBlockchain(blockchain) || currency.Type != money.Coin {
		return Fee{}, errors.Wrap(ErrUnsupportedRuntime, "UTXO runtime supports native BTC/LTC coins only")
	}

	feeRate := utxoFeeRate(blockchain, isTest)
	estimatedVBytes := estimateBitcoinP2WPKHVSize(1, 2)
	feeSats := feeRate * estimatedVBytes

	totalCost, err := baseCurrency.MakeAmount(strconv.FormatInt(feeSats, 10))
	if err != nil {
		return Fee{}, errors.Wrapf(err, "unable to make %s fee amount", currency.Ticker)
	}

	conv, err := s.CryptoToFiat(ctx, totalCost, money.USD)
	if err != nil {
		return Fee{}, errors.Wrap(err, "unable to calculate total cost in USD")
	}

	return NewFee(currency, time.Now().UTC(), isTest, BitcoinFee{
		FeeSatPerVByte: strconv.FormatInt(feeRate, 10),
		EstimatedBytes: strconv.FormatInt(estimatedVBytes, 10),
		TotalCostSats:  totalCost.StringRaw(),
		TotalCostBTC:   totalCost.String(),
		TotalCostUSD:   conv.To.String(),

		totalCostUSD: conv.To,
	}), nil
}

func (s *Service) PrepareBitcoinTransaction(
	ctx context.Context,
	senderAddress string,
	recipient string,
	amount money.Money,
	fee Fee,
	isTest bool,
) (BitcoinTransactionPlan, error) {
	return s.PrepareBitcoinTransactionExcluding(ctx, senderAddress, recipient, amount, fee, isTest, nil)
}

func (s *Service) PrepareBitcoinSweepTransaction(
	ctx context.Context,
	senderAddress string,
	recipient string,
	fee Fee,
	isTest bool,
) (BitcoinTransactionPlan, error) {
	return s.PrepareBitcoinSweepTransactionExcluding(ctx, senderAddress, recipient, fee, isTest, nil)
}

func (s *Service) PrepareBitcoinSweepTransactionExcluding(
	ctx context.Context,
	senderAddress string,
	recipient string,
	fee Fee,
	isTest bool,
	excluded []BitcoinUTXOKey,
) (BitcoinTransactionPlan, error) {
	blockchain := kmswallet.Blockchain(fee.Currency.Blockchain)
	if !isUTXOBlockchain(blockchain) || fee.Currency.Type != money.Coin {
		return BitcoinTransactionPlan{}, errors.Wrap(ErrUnsupportedRuntime, "UTXO runtime supports native BTC/LTC coins only")
	}

	if err := kmswallet.ValidateAddressForNetwork(blockchain, senderAddress, isTest); err != nil {
		return BitcoinTransactionPlan{}, errors.Wrapf(err, "invalid %s sender address", blockchain)
	}

	if err := kmswallet.ValidateAddressForNetwork(blockchain, recipient, isTest); err != nil {
		return BitcoinTransactionPlan{}, errors.Wrapf(err, "invalid %s recipient address", blockchain)
	}

	feeBTC, err := fee.ToBitcoinFee()
	if err != nil {
		return BitcoinTransactionPlan{}, err
	}

	feeRate, err := strconv.ParseInt(feeBTC.FeeSatPerVByte, 10, 64)
	if err != nil || feeRate <= 0 {
		return BitcoinTransactionPlan{}, errors.Wrap(ErrValidation, "invalid UTXO fee rate")
	}

	utxos, err := s.utxoSpendableUTXOs(ctx, blockchain, senderAddress, isTest)
	if err != nil {
		return BitcoinTransactionPlan{}, err
	}

	excludedMap := make(map[BitcoinUTXOKey]struct{}, len(excluded))
	for _, key := range excluded {
		excludedMap[key] = struct{}{}
	}

	if len(excludedMap) > 0 {
		filtered := make([]BitcoinUTXO, 0, len(utxos))
		for _, utxo := range utxos {
			if _, ok := excludedMap[BitcoinUTXOKey{Hash: utxo.Hash, Index: utxo.Index}]; ok {
				continue
			}
			filtered = append(filtered, utxo)
		}
		utxos = filtered
	}

	selected, amountSats, feeSats, estimatedVBytes, err := selectBitcoinSweepUTXOs(utxos, feeRate)
	if err != nil {
		return BitcoinTransactionPlan{}, err
	}

	return BitcoinTransactionPlan{
		Inputs: selected,
		Outputs: []BitcoinOutput{{
			Address:    recipient,
			AmountSats: amountSats,
		}},
		ChangeAddress:     "",
		FeeSatPerVByte:    feeRate,
		EstimatedVBytes:   estimatedVBytes,
		FeeSats:           feeSats,
		RequiredAmountSat: amountSats,
		RBF:               true,
	}, nil
}

func (s *Service) PrepareBitcoinTransactionExcluding(
	ctx context.Context,
	senderAddress string,
	recipient string,
	amount money.Money,
	fee Fee,
	isTest bool,
	excluded []BitcoinUTXOKey,
) (BitcoinTransactionPlan, error) {
	blockchain := kmswallet.Blockchain(fee.Currency.Blockchain)
	if !isUTXOBlockchain(blockchain) || fee.Currency.Type != money.Coin {
		return BitcoinTransactionPlan{}, errors.Wrap(ErrUnsupportedRuntime, "UTXO runtime supports native BTC/LTC coins only")
	}

	if amount.Ticker() != fee.Currency.Ticker {
		return BitcoinTransactionPlan{}, errors.Wrapf(ErrValidation, "%s amount ticker mismatch: %s", fee.Currency.Ticker, amount.Ticker())
	}

	if err := kmswallet.ValidateAddressForNetwork(blockchain, senderAddress, isTest); err != nil {
		return BitcoinTransactionPlan{}, errors.Wrapf(err, "invalid %s sender address", blockchain)
	}

	if err := kmswallet.ValidateAddressForNetwork(blockchain, recipient, isTest); err != nil {
		return BitcoinTransactionPlan{}, errors.Wrapf(err, "invalid %s recipient address", blockchain)
	}

	feeBTC, err := fee.ToBitcoinFee()
	if err != nil {
		return BitcoinTransactionPlan{}, err
	}

	feeRate, err := strconv.ParseInt(feeBTC.FeeSatPerVByte, 10, 64)
	if err != nil || feeRate <= 0 {
		return BitcoinTransactionPlan{}, errors.Wrap(ErrValidation, "invalid UTXO fee rate")
	}

	amountSats, err := satoshiAmount(amount)
	if err != nil {
		return BitcoinTransactionPlan{}, err
	}

	if amountSats <= 0 {
		return BitcoinTransactionPlan{}, errors.Wrapf(ErrValidation, "%s amount must be positive", fee.Currency.Ticker)
	}

	if amountSats < bitcoinDustSats {
		return BitcoinTransactionPlan{}, errors.Wrapf(ErrValidation, "%s amount is below dust threshold %d sat", fee.Currency.Ticker, bitcoinDustSats)
	}

	utxos, err := s.utxoSpendableUTXOs(ctx, blockchain, senderAddress, isTest)
	if err != nil {
		return BitcoinTransactionPlan{}, err
	}

	excludedMap := make(map[BitcoinUTXOKey]struct{}, len(excluded))
	for _, key := range excluded {
		excludedMap[key] = struct{}{}
	}

	if len(excludedMap) > 0 {
		filtered := make([]BitcoinUTXO, 0, len(utxos))
		for _, utxo := range utxos {
			if _, ok := excludedMap[BitcoinUTXOKey{Hash: utxo.Hash, Index: utxo.Index}]; ok {
				continue
			}
			filtered = append(filtered, utxo)
		}
		utxos = filtered
	}

	selected, feeSats, changeSats, estimatedVBytes, err := selectBitcoinUTXOs(utxos, amountSats, feeRate)
	if err != nil {
		return BitcoinTransactionPlan{}, err
	}

	outputs := []BitcoinOutput{{
		Address:    recipient,
		AmountSats: amountSats,
	}}
	if changeSats >= bitcoinDustSats {
		outputs = append(outputs, BitcoinOutput{
			Address:    senderAddress,
			AmountSats: changeSats,
		})
	}

	return BitcoinTransactionPlan{
		Inputs:            selected,
		Outputs:           outputs,
		ChangeAddress:     senderAddress,
		FeeSatPerVByte:    feeRate,
		EstimatedVBytes:   estimatedVBytes,
		FeeSats:           feeSats,
		RequiredAmountSat: amountSats,
		RBF:               true,
	}, nil
}

func (s *Service) bitcoinSpendableUTXOs(ctx context.Context, address string, isTest bool) ([]BitcoinUTXO, error) {
	return s.utxoSpendableUTXOs(ctx, kmswallet.BTC, address, isTest)
}

func (s *Service) utxoSpendableUTXOs(
	ctx context.Context,
	blockchain kmswallet.Blockchain,
	address string,
	isTest bool,
) ([]BitcoinUTXO, error) {
	if s.providers.Chain != nil {
		return s.utxoSpendableUTXOsFromExplorer(ctx, blockchain, address, isTest)
	}

	if blockchain != kmswallet.BTC {
		return nil, errors.New("LTC UTXO lookup requires chain provider")
	}
	if s.providers.Tatum == nil || !s.providers.Tatum.HasAPIKey(isTest) {
		return nil, errors.New("BTC UTXO lookup requires chain provider or Tatum API key")
	}

	utxos := make([]BitcoinUTXO, 0)
	seen := make(map[string]struct{})

	for page := 0; page < bitcoinMaxAddressTxPages; page++ {
		offset := float64(page * bitcoinAddressTxPageSize)
		txs, _, err := s.providers.Tatum.BitcoinTransactionsByAddress(
			ctx,
			address,
			float64(bitcoinAddressTxPageSize),
			&offset,
			isTest,
		)
		if err != nil {
			return nil, errors.Wrap(err, "unable to list BTC address transactions")
		}

		if len(txs) == 0 {
			break
		}

		for _, tx := range txs {
			for outputIndex, output := range tx.Outputs {
				if output.Address != address || output.Value <= 0 {
					continue
				}

				key := tx.Hash + ":" + strconv.Itoa(outputIndex)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}

				utxo, _, err := s.providers.Tatum.BitcoinUTXO(ctx, tx.Hash, uint32(outputIndex), isTest)
				if isMissingBitcoinUTXO(err) {
					continue
				}
				if err != nil {
					return nil, errors.Wrap(err, "unable to verify BTC UTXO")
				}

				if utxo.Value <= 0 {
					continue
				}

				utxos = append(utxos, BitcoinUTXO{
					Hash:        tx.Hash,
					Index:       uint32(outputIndex),
					AmountSats:  int64(math.Round(utxo.Value)),
					Script:      utxo.Script,
					Address:     utxo.Address,
					BlockNumber: int64(math.Round(utxo.Height)),
				})
			}
		}

		if len(txs) < bitcoinAddressTxPageSize {
			break
		}
	}

	sort.SliceStable(utxos, func(i, j int) bool {
		if utxos[i].AmountSats == utxos[j].AmountSats {
			return utxos[i].Hash < utxos[j].Hash
		}

		return utxos[i].AmountSats > utxos[j].AmountSats
	})

	return utxos, nil
}

func (s *Service) utxoSpendableUTXOsFromExplorer(
	ctx context.Context,
	blockchain kmswallet.Blockchain,
	address string,
	isTest bool,
) ([]BitcoinUTXO, error) {
	utxos, err := s.providers.Chain.UTXOAddressUTXOs(ctx, blockchain, address, isTest)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to list %s address UTXOs", blockchain)
	}

	out := make([]BitcoinUTXO, 0, len(utxos))
	for _, utxo := range utxos {
		if utxo.Value <= 0 || utxo.TxID == "" {
			continue
		}

		blockNumber := int64(0)
		if utxo.Status.Confirmed {
			blockNumber = utxo.Status.BlockHeight
		}

		out = append(out, BitcoinUTXO{
			Hash:        utxo.TxID,
			Index:       utxo.Vout,
			AmountSats:  utxo.Value,
			Address:     address,
			BlockNumber: blockNumber,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].AmountSats == out[j].AmountSats {
			return out[i].Hash < out[j].Hash
		}

		return out[i].AmountSats > out[j].AmountSats
	})

	return out, nil
}

func isMissingBitcoinUTXO(err error) bool {
	if err == nil {
		return false
	}

	errSwagger, ok := err.(client.GenericSwaggerError)
	if !ok {
		return false
	}

	body := strings.ToLower(string(errSwagger.Body()))
	return strings.Contains(body, "not found") || strings.Contains(body, "404")
}

func selectBitcoinUTXOs(utxos []BitcoinUTXO, amountSats, feeRate int64) (
	[]BitcoinUTXO,
	int64,
	int64,
	int64,
	error,
) {
	if amountSats <= 0 || feeRate <= 0 {
		return nil, 0, 0, 0, errors.Wrap(ErrValidation, "invalid BTC amount or fee rate")
	}

	selected := make([]BitcoinUTXO, 0, len(utxos))
	var inputTotal int64

	for _, utxo := range utxos {
		if utxo.AmountSats <= 0 {
			continue
		}

		selected = append(selected, utxo)
		inputTotal += utxo.AmountSats

		estimatedWithChange := estimateBitcoinP2WPKHVSize(len(selected), 2)
		feeWithChange := estimatedWithChange * feeRate
		changeSats := inputTotal - amountSats - feeWithChange
		if changeSats >= bitcoinDustSats {
			return selected, feeWithChange, changeSats, estimatedWithChange, nil
		}

		estimatedWithoutChange := estimateBitcoinP2WPKHVSize(len(selected), 1)
		feeWithoutChange := inputTotal - amountSats
		if feeWithoutChange >= estimatedWithoutChange*feeRate && feeWithoutChange >= 0 {
			return selected, feeWithoutChange, 0, estimatedWithoutChange, nil
		}
	}

	return nil, 0, 0, 0, errors.Wrap(ErrInsufficientFunds, "not enough BTC UTXOs to cover amount and network fee")
}

func selectBitcoinSweepUTXOs(utxos []BitcoinUTXO, feeRate int64) (
	[]BitcoinUTXO,
	int64,
	int64,
	int64,
	error,
) {
	if feeRate <= 0 {
		return nil, 0, 0, 0, errors.Wrap(ErrValidation, "invalid BTC fee rate")
	}

	selected := make([]BitcoinUTXO, 0, len(utxos))
	var inputTotal int64
	perInputFee := int64(68) * feeRate

	for _, utxo := range utxos {
		if utxo.AmountSats <= perInputFee {
			continue
		}

		selected = append(selected, utxo)
		inputTotal += utxo.AmountSats
	}

	if len(selected) == 0 {
		return nil, 0, 0, 0, errors.Wrap(ErrInsufficientFunds, "not enough BTC UTXOs to sweep")
	}

	estimatedVBytes := estimateBitcoinP2WPKHVSize(len(selected), 1)
	feeSats := estimatedVBytes * feeRate
	amountSats := inputTotal - feeSats
	if amountSats < bitcoinDustSats {
		return nil, 0, 0, 0, errors.Wrap(ErrInsufficientFunds, "sweep amount is below BTC dust threshold after network fee")
	}

	return selected, amountSats, feeSats, estimatedVBytes, nil
}

func estimateBitcoinP2WPKHVSize(inputs int, outputs int) int64 {
	return int64(10 + (68 * inputs) + (31 * outputs))
}

func bitcoinFeeRate(isTest bool) int64 {
	if isTest {
		return bitcoinTestnetFeeSatPerVByte
	}

	return bitcoinMainnetFeeSatPerVByte
}

func satoshiAmount(amount money.Money) (int64, error) {
	bigInt, decimals := amount.BigInt()
	if amount.Type() != money.Crypto || decimals != 8 {
		return 0, errors.Wrap(ErrValidation, "UTXO amount must be crypto with 8 decimals")
	}

	if !bigInt.IsInt64() {
		return 0, errors.Wrap(ErrValidation, "UTXO amount is too large")
	}

	return bigInt.Int64(), nil
}

func (s *Service) getBitcoinReceipt(
	ctx context.Context,
	nativeCoin money.CryptoCurrency,
	txID string,
	isTest bool,
) (*TransactionReceipt, error) {
	blockchain := kmswallet.Blockchain(nativeCoin.Blockchain)
	if s.providers.Chain != nil {
		return s.getBitcoinReceiptFromExplorer(ctx, nativeCoin, txID, isTest)
	}

	if blockchain != kmswallet.BTC {
		return nil, errors.New("LTC receipt lookup requires chain provider")
	}
	if s.providers.Tatum == nil || !s.providers.Tatum.HasAPIKey(isTest) {
		return nil, errors.New("BTC receipt lookup requires chain provider or Tatum API key")
	}

	tx, _, err := s.providers.Tatum.BitcoinTransaction(ctx, txID, isTest)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get BTC transaction")
	}

	info, _, err := s.providers.Tatum.BitcoinBlockchainInfo(ctx, isTest)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get BTC blockchain info")
	}

	networkFee, err := nativeCoin.MakeAmount(strconv.FormatInt(int64(math.Round(tx.Fee)), 10))
	if err != nil {
		return nil, errors.Wrap(err, "unable to calculate BTC network fee")
	}

	confirmations := int64(0)
	if tx.BlockNumber > 0 && info.Blocks >= tx.BlockNumber {
		confirmations = int64(math.Round(info.Blocks-tx.BlockNumber)) + 1
	}

	sender, recipient := "", ""
	if len(tx.Inputs) > 0 && tx.Inputs[0].Coin != nil {
		sender = tx.Inputs[0].Coin.Address
	}
	if len(tx.Outputs) > 0 {
		recipient = tx.Outputs[0].Address
	}

	return &TransactionReceipt{
		Blockchain:    nativeCoin.Blockchain,
		IsTest:        isTest,
		Sender:        sender,
		Recipient:     recipient,
		Hash:          txID,
		NetworkFee:    networkFee,
		Success:       true,
		Confirmations: confirmations,
		IsConfirmed:   confirmations >= utxoRequiredConfirmations(blockchain),
	}, nil
}

func (s *Service) getBitcoinReceiptFromExplorer(
	ctx context.Context,
	nativeCoin money.CryptoCurrency,
	txID string,
	isTest bool,
) (*TransactionReceipt, error) {
	blockchain := kmswallet.Blockchain(nativeCoin.Blockchain)
	tx, err := s.providers.Chain.UTXOTransaction(ctx, blockchain, txID, isTest)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get %s transaction", blockchain)
	}

	networkFee, err := nativeCoin.MakeAmount(strconv.FormatInt(tx.Fee, 10))
	if err != nil {
		return nil, errors.Wrapf(err, "unable to calculate %s network fee", blockchain)
	}

	var confirmations int64
	if tx.Status.Confirmed && tx.Status.BlockHeight > 0 {
		tip, err := s.providers.Chain.UTXOTipHeight(ctx, blockchain, isTest)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to get %s tip height", blockchain)
		}

		if tip >= tx.Status.BlockHeight {
			confirmations = tip - tx.Status.BlockHeight + 1
		}
	}

	sender, recipient := "", ""
	if len(tx.Vin) > 0 && tx.Vin[0].PrevOut != nil {
		sender = tx.Vin[0].PrevOut.ScriptPubKeyAddress
	}
	if len(tx.Vout) > 0 {
		recipient = tx.Vout[0].ScriptPubKeyAddress
	}

	return &TransactionReceipt{
		Blockchain:    nativeCoin.Blockchain,
		IsTest:        isTest,
		Sender:        sender,
		Recipient:     recipient,
		Hash:          txID,
		NetworkFee:    networkFee,
		Success:       true,
		Confirmations: confirmations,
		IsConfirmed:   confirmations >= utxoRequiredConfirmations(blockchain),
	}, nil
}

func isUTXOBlockchain(blockchain kmswallet.Blockchain) bool {
	switch blockchain {
	case kmswallet.BTC, kmswallet.LTC:
		return true
	default:
		return false
	}
}

func utxoRequiredConfirmations(blockchain kmswallet.Blockchain) int64 {
	switch blockchain {
	case kmswallet.LTC:
		return litecoinRequiredConfirmations
	default:
		return bitcoinRequiredConfirmations
	}
}

func utxoFeeRate(blockchain kmswallet.Blockchain, isTest bool) int64 {
	switch blockchain {
	case kmswallet.LTC:
		if isTest {
			return litecoinTestnetFeeSatPerVByte
		}
		return litecoinMainnetFeeSatPerVByte
	default:
		return bitcoinFeeRate(isTest)
	}
}
