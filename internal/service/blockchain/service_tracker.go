package blockchain

import (
	"context"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	kms "github.com/oxygenpay/oxygen/internal/kms/wallet"
	"github.com/oxygenpay/oxygen/internal/money"
	chainprovider "github.com/oxygenpay/oxygen/internal/provider/chain"
	"github.com/oxygenpay/oxygen/internal/provider/trongrid"
	"github.com/oxygenpay/oxygen/internal/util"
	"github.com/pkg/errors"
)

var erc20TransferTopic = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

const defaultTrackerEVMScanBlocks = int64(1200)

type Tracker interface {
	ListIncomingTransactions(
		ctx context.Context,
		recipient string,
		currency money.CryptoCurrency,
		isTest bool,
	) ([]IncomingTransaction, error)
}

type IncomingTransaction struct {
	Currency      money.CryptoCurrency
	Amount        money.Money
	SenderAddress string
	TransactionID string
	NetworkID     string
	BlockNumber   int64
	IsMempool     bool
}

func (s *Service) ListIncomingTransactions(
	ctx context.Context,
	recipient string,
	currency money.CryptoCurrency,
	isTest bool,
) ([]IncomingTransaction, error) {
	if recipient == "" {
		return nil, errors.Wrap(ErrValidation, "recipient address is empty")
	}

	if err := validateTransactionRuntimeBlockchain(currency.Blockchain); err != nil {
		return nil, err
	}

	switch kms.Blockchain(currency.Blockchain) {
	case kms.BTC, kms.LTC:
		return s.listBitcoinIncomingTransactions(ctx, recipient, currency, isTest)
	case kms.ETH, kms.MATIC, kms.BSC:
		return s.listEVMIncomingTransactions(ctx, recipient, currency, isTest)
	case kms.TRON:
		return s.listTronIncomingTransactions(ctx, recipient, currency, isTest)
	default:
		return nil, errors.Wrapf(ErrUnsupportedRuntime, "incoming tracker is not implemented for %q", currency.Blockchain)
	}
}

func (s *Service) listBitcoinIncomingTransactions(
	ctx context.Context,
	recipient string,
	currency money.CryptoCurrency,
	isTest bool,
) ([]IncomingTransaction, error) {
	blockchain := kms.Blockchain(currency.Blockchain)
	if s.providers.Chain != nil {
		return s.listBitcoinIncomingTransactionsFromExplorer(ctx, recipient, currency, isTest)
	}

	if blockchain != kms.BTC {
		return nil, errors.New("LTC tracker requires chain provider")
	}
	if s.providers.Tatum == nil || !s.providers.Tatum.HasAPIKey(isTest) {
		return nil, errors.New("BTC tracker requires chain provider or Tatum API key")
	}

	txs, _, err := s.providers.Tatum.BitcoinTransactionsByAddress(
		ctx,
		recipient,
		float64(bitcoinAddressTxPageSize),
		nil,
		isTest,
	)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list BTC address transactions")
	}

	out := make([]IncomingTransaction, 0, len(txs))
	for _, tx := range txs {
		var totalSats int64
		for _, output := range tx.Outputs {
			if output.Address == recipient && output.Value > 0 {
				totalSats += int64(output.Value)
			}
		}
		if totalSats <= 0 {
			continue
		}

		amount, err := currency.MakeAmount(strconv.FormatInt(totalSats, 10))
		if err != nil {
			return nil, err
		}

		sender := ""
		if len(tx.Inputs) > 0 && tx.Inputs[0].Coin != nil {
			sender = tx.Inputs[0].Coin.Address
		}

		out = append(out, IncomingTransaction{
			Currency:      currency,
			Amount:        amount,
			SenderAddress: sender,
			TransactionID: tx.Hash,
			NetworkID:     currency.ChooseNetwork(isTest),
			BlockNumber:   int64(tx.BlockNumber),
			IsMempool:     tx.BlockNumber <= 0,
		})
	}

	return out, nil
}

func (s *Service) listBitcoinIncomingTransactionsFromExplorer(
	ctx context.Context,
	recipient string,
	currency money.CryptoCurrency,
	isTest bool,
) ([]IncomingTransaction, error) {
	blockchain := kms.Blockchain(currency.Blockchain)
	mempoolTxs, mempoolErr := s.providers.Chain.UTXOAddressTransactions(ctx, blockchain, recipient, isTest, true)
	confirmedTxs, confirmedErr := s.providers.Chain.UTXOAddressTransactions(ctx, blockchain, recipient, isTest, false)
	switch {
	case mempoolErr != nil && confirmedErr != nil:
		return nil, errors.Wrapf(
			mempoolErr,
			"unable to list %s mempool transactions; confirmed lookup also failed: %v",
			blockchain,
			confirmedErr,
		)
	case mempoolErr != nil:
		s.logger.Warn().
			Err(mempoolErr).
			Str("blockchain", string(blockchain)).
			Str("recipient", recipient).
			Msg("unable to list UTXO mempool transactions")
	case confirmedErr != nil:
		s.logger.Warn().
			Err(confirmedErr).
			Str("blockchain", string(blockchain)).
			Str("recipient", recipient).
			Msg("unable to list UTXO confirmed transactions")
	}

	txs := append(mempoolTxs, confirmedTxs...)
	out := make([]IncomingTransaction, 0, len(txs))
	seen := make(map[string]struct{}, len(txs))

	for _, tx := range txs {
		if _, exists := seen[tx.TxID]; exists {
			continue
		}
		seen[tx.TxID] = struct{}{}

		incoming, ok, err := bitcoinIncomingFromExplorer(tx, recipient, currency, isTest)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		out = append(out, incoming)
	}

	return out, nil
}

func bitcoinIncomingFromExplorer(
	tx chainprovider.BitcoinTransaction,
	recipient string,
	currency money.CryptoCurrency,
	isTest bool,
) (IncomingTransaction, bool, error) {
	var totalSats int64
	for _, output := range tx.Vout {
		if output.ScriptPubKeyAddress == recipient && output.Value > 0 {
			totalSats += output.Value
		}
	}

	if totalSats <= 0 {
		return IncomingTransaction{}, false, nil
	}

	amount, err := currency.MakeAmount(strconv.FormatInt(totalSats, 10))
	if err != nil {
		return IncomingTransaction{}, false, err
	}

	sender := ""
	for _, input := range tx.Vin {
		if input.PrevOut == nil || input.PrevOut.ScriptPubKeyAddress == "" {
			continue
		}
		sender = input.PrevOut.ScriptPubKeyAddress
		break
	}

	return IncomingTransaction{
		Currency:      currency,
		Amount:        amount,
		SenderAddress: sender,
		TransactionID: tx.TxID,
		NetworkID:     currency.ChooseNetwork(isTest),
		BlockNumber:   tx.Status.BlockHeight,
		IsMempool:     !tx.Status.Confirmed,
	}, true, nil
}

func (s *Service) listTronIncomingTransactions(
	ctx context.Context,
	recipient string,
	currency money.CryptoCurrency,
	isTest bool,
) ([]IncomingTransaction, error) {
	if s.providers.Trongrid == nil {
		return nil, errors.New("Trongrid provider is not configured")
	}

	if currency.Type == money.Token {
		return s.listTronTokenIncomingTransactions(ctx, recipient, currency, isTest)
	}

	txs, err := s.providers.Trongrid.ListIncomingTransactions(ctx, recipient, isTest)
	if err != nil {
		return nil, err
	}

	out := make([]IncomingTransaction, 0, len(txs))
	for _, tx := range txs {
		incoming, ok, err := tronIncomingFromTransaction(tx, recipient, currency, isTest)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		out = append(out, incoming)
	}

	return out, nil
}

func (s *Service) listTronTokenIncomingTransactions(
	ctx context.Context,
	recipient string,
	currency money.CryptoCurrency,
	isTest bool,
) ([]IncomingTransaction, error) {
	contract := currency.ChooseContractAddress(isTest)
	txs, err := s.providers.Trongrid.ListIncomingTRC20Transactions(ctx, recipient, contract, isTest)
	if err != nil {
		return nil, err
	}

	out := make([]IncomingTransaction, 0, len(txs))
	for _, tx := range txs {
		if tx.TransactionID == "" ||
			!strings.EqualFold(tx.To, recipient) ||
			!strings.EqualFold(tx.TokenInfo.Address, contract) {
			continue
		}

		amount, err := currency.MakeAmount(tx.Value)
		if err != nil {
			return nil, err
		}

		out = append(out, IncomingTransaction{
			Currency:      currency,
			Amount:        amount,
			SenderAddress: tx.From,
			TransactionID: tx.TransactionID,
			NetworkID:     currency.ChooseNetwork(isTest),
			BlockNumber:   tx.BlockNumber,
		})
	}

	return out, nil
}

func tronIncomingFromTransaction(
	tx trongrid.AccountTransaction,
	recipient string,
	currency money.CryptoCurrency,
	isTest bool,
) (IncomingTransaction, bool, error) {
	if tx.TxID == "" || !tronTransactionSucceeded(tx) || len(tx.RawData.Contract) == 0 {
		return IncomingTransaction{}, false, nil
	}

	contract := tx.RawData.Contract[0]
	if contract.Type != "TransferContract" || contract.Parameter.Value.Amount <= 0 {
		return IncomingTransaction{}, false, nil
	}

	sender := util.TronHexToBase58(contract.Parameter.Value.OwnerAddress)
	to := util.TronHexToBase58(contract.Parameter.Value.ToAddress)
	if to != recipient {
		return IncomingTransaction{}, false, nil
	}

	amount, err := currency.MakeAmount(strconv.FormatInt(contract.Parameter.Value.Amount, 10))
	if err != nil {
		return IncomingTransaction{}, false, err
	}

	// Mirror webhook behavior: one-TRX account activation top-ups are unexpected noise.
	if currency.Type == money.Coin && amount.StringRaw() == "1000000" {
		return IncomingTransaction{}, false, nil
	}

	return IncomingTransaction{
		Currency:      currency,
		Amount:        amount,
		SenderAddress: sender,
		TransactionID: tx.TxID,
		NetworkID:     currency.ChooseNetwork(isTest),
		BlockNumber:   tx.BlockNumber,
	}, true, nil
}

func tronTransactionSucceeded(tx trongrid.AccountTransaction) bool {
	if len(tx.Ret) == 0 {
		return true
	}

	for _, ret := range tx.Ret {
		if ret.ContractRet == "SUCCESS" {
			return true
		}
	}

	return false
}

func (s *Service) listEVMIncomingTransactions(
	ctx context.Context,
	recipient string,
	currency money.CryptoCurrency,
	isTest bool,
) ([]IncomingTransaction, error) {
	rpc, err := s.evmRPC(ctx, currency.Blockchain, isTest)
	if err != nil {
		return nil, err
	}
	defer rpc.Close()

	if currency.Type == money.Token {
		return s.listEVMTokenIncomingTransactions(ctx, rpc, recipient, currency, isTest)
	}

	return s.listEVMNativeIncomingTransactions(ctx, rpc, recipient, currency, isTest)
}

func (s *Service) listEVMNativeIncomingTransactions(
	ctx context.Context,
	rpc ethClient,
	recipient string,
	currency money.CryptoCurrency,
	isTest bool,
) ([]IncomingTransaction, error) {
	latest, err := rpc.BlockNumber(ctx)
	if err != nil {
		return nil, err
	}

	from := evmScanFromBlock(latest, s.evmScanBlocks())
	recipientAddress := common.HexToAddress(recipient)
	out := make([]IncomingTransaction, 0)

	for blockNumber := int64(latest); blockNumber >= from; blockNumber-- {
		block, err := rpc.BlockByNumber(ctx, big.NewInt(blockNumber))
		if err != nil {
			return nil, err
		}

		for _, tx := range block.Transactions() {
			if tx.To() == nil || *tx.To() != recipientAddress || tx.Value().Sign() <= 0 {
				continue
			}

			amount, err := currency.MakeAmountFromBigInt(tx.Value())
			if err != nil {
				return nil, err
			}

			sender, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
			if err != nil {
				return nil, errors.Wrap(err, "unable to resolve transaction sender")
			}

			out = append(out, IncomingTransaction{
				Currency:      currency,
				Amount:        amount,
				SenderAddress: sender.String(),
				TransactionID: tx.Hash().String(),
				NetworkID:     currency.ChooseNetwork(isTest),
				BlockNumber:   blockNumber,
			})
		}
	}

	return out, nil
}

func (s *Service) listEVMTokenIncomingTransactions(
	ctx context.Context,
	rpc ethClient,
	recipient string,
	currency money.CryptoCurrency,
	isTest bool,
) ([]IncomingTransaction, error) {
	latest, err := rpc.BlockNumber(ctx)
	if err != nil {
		return nil, err
	}

	from := evmScanFromBlock(latest, s.evmScanBlocks())
	contract := common.HexToAddress(currency.ChooseContractAddress(isTest))
	recipientTopic := common.BytesToHash(common.HexToAddress(recipient).Bytes())

	logs, err := rpc.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: big.NewInt(from),
		ToBlock:   big.NewInt(int64(latest)),
		Addresses: []common.Address{contract},
		Topics: [][]common.Hash{
			{erc20TransferTopic},
			nil,
			{recipientTopic},
		},
	})
	if err != nil {
		return nil, err
	}

	out := make([]IncomingTransaction, 0, len(logs))
	for _, entry := range logs {
		if len(entry.Topics) < 3 || len(entry.Data) == 0 {
			continue
		}

		amount, err := currency.MakeAmountFromBigInt(new(big.Int).SetBytes(entry.Data))
		if err != nil {
			return nil, err
		}

		out = append(out, IncomingTransaction{
			Currency:      currency,
			Amount:        amount,
			SenderAddress: common.BytesToAddress(entry.Topics[1].Bytes()).String(),
			TransactionID: entry.TxHash.String(),
			NetworkID:     currency.ChooseNetwork(isTest),
			BlockNumber:   int64(entry.BlockNumber),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].BlockNumber > out[j].BlockNumber
	})

	return out, nil
}

func (s *Service) evmScanBlocks() int64 {
	if s.providers.Chain == nil {
		return defaultTrackerEVMScanBlocks
	}

	return s.providers.Chain.EVMScanBlocks()
}

func evmScanFromBlock(latest uint64, scanBlocks int64) int64 {
	if scanBlocks <= 0 {
		scanBlocks = defaultTrackerEVMScanBlocks
	}

	from := int64(latest) - scanBlocks + 1
	if from < 0 {
		return 0
	}

	return from
}

type ethClient interface {
	BlockNumber(ctx context.Context) (uint64, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
}
