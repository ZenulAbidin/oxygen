package wallet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"github.com/jackc/pgconn"
	"github.com/oxygenpay/oxygen/internal/db/repository"
	kms "github.com/oxygenpay/oxygen/internal/kms/wallet"
	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/service/blockchain"
	"github.com/oxygenpay/oxygen/internal/util"
	kmsclient "github.com/oxygenpay/oxygen/pkg/api-kms/v1/client/wallet"
	kmsmodel "github.com/oxygenpay/oxygen/pkg/api-kms/v1/model"
	"github.com/pkg/errors"
)

const (
	bitcoinUTXOReservationAttempts = 3
	bitcoinUTXOReservationTTL      = 30 * time.Minute
)

func (s *Service) CreateSignedTransaction(
	ctx context.Context,
	sender *Wallet,
	recipient string,
	currency money.CryptoCurrency,
	amount money.Money,
	fee blockchain.Fee,
	isTest bool,
) (string, error) {
	if isUTXOBlockchain(kms.Blockchain(currency.Blockchain)) {
		return s.createSignedBitcoinTransaction(ctx, sender, recipient, currency, amount, fee, isTest)
	}

	if err := validateAccountBasedTransactionCurrency(currency); err != nil {
		return "", err
	}

	nonce, err := s.IncrementPendingTransaction(ctx, sender.ID, isTest)
	if err != nil {
		return "", errors.Wrap(err, "unable to increment pending transactions counter")
	}

	txRaw, errCreate := s.createSignedTransaction(
		ctx,
		sender,
		recipient,
		currency,
		amount,
		fee,
		int64(nonce),
		isTest,
	)

	if errCreate != nil {
		if err := s.DecrementPendingTransaction(ctx, sender.ID, isTest); err != nil {
			return "", errors.Wrap(err, "unable to decrement pending transactions counter")
		}
	}

	return txRaw, errCreate
}

func (s *Service) createSignedBitcoinTransaction(
	ctx context.Context,
	sender *Wallet,
	recipient string,
	currency money.CryptoCurrency,
	amount money.Money,
	fee blockchain.Fee,
	isTest bool,
) (string, error) {
	if currency.Type != money.Coin || !isUTXOBlockchain(kms.Blockchain(currency.Blockchain)) {
		return "", errors.Wrapf(ErrUnsupportedTransactionPath, "%s requires native UTXO flow", currency.Ticker)
	}

	if s.store == nil {
		return "", errors.Wrap(ErrUnsupportedTransactionPath, "UTXO reservations require wallet storage")
	}

	var lastConflict error
	for attempt := 0; attempt < bitcoinUTXOReservationAttempts; attempt++ {
		excluded, err := s.activeBitcoinUTXOReservationKeys(ctx, sender.ID, isTest)
		if err != nil {
			return "", errors.Wrap(err, "unable to list reserved BTC UTXOs")
		}

		plan, err := s.blockchain.PrepareBitcoinTransactionExcluding(
			ctx,
			sender.Address,
			recipient,
			amount,
			fee,
			isTest,
			excluded,
		)
		if err != nil {
			return "", errors.Wrap(err, "unable to prepare BTC transaction")
		}

		reservationIDs, err := s.reserveBitcoinUTXOs(ctx, sender.ID, plan.Inputs, isTest)
		if isUniqueViolation(err) {
			lastConflict = err
			continue
		}
		if err != nil {
			return "", errors.Wrap(err, "unable to reserve BTC UTXOs")
		}

		rawTX, err := s.createSignedBitcoinTransactionFromPlan(ctx, sender, plan, isTest)
		if err != nil {
			if releaseErr := s.releaseBitcoinUTXOReservationIDs(ctx, reservationIDs); releaseErr != nil {
				return "", errors.Wrap(releaseErr, "unable to release BTC UTXO reservations after KMS failure")
			}

			return "", err
		}

		rawTXID := bitcoinRawTXReservationID(rawTX)
		if err := s.attachBitcoinUTXOReservationsRawTX(ctx, reservationIDs, rawTXID); err != nil {
			if releaseErr := s.releaseBitcoinUTXOReservationIDs(ctx, reservationIDs); releaseErr != nil {
				return "", errors.Wrap(releaseErr, "unable to release BTC UTXO reservations after attach failure")
			}

			return "", errors.Wrap(err, "unable to attach raw BTC transaction to UTXO reservations")
		}

		return rawTX, nil
	}

	return "", errors.Wrap(ErrUTXOReservationConflict, lastConflict.Error())
}

func (s *Service) createSignedBitcoinTransactionFromPlan(
	ctx context.Context,
	sender *Wallet,
	plan blockchain.BitcoinTransactionPlan,
	isTest bool,
) (string, error) {
	inputs := make([]*kmsmodel.BitcoinUTXO, 0, len(plan.Inputs))
	for _, input := range plan.Inputs {
		inputs = append(inputs, &kmsmodel.BitcoinUTXO{
			Hash:       input.Hash,
			Index:      int64(input.Index),
			AmountSats: input.AmountSats,
			Script:     input.Script,
			Address:    input.Address,
		})
	}

	outputs := make([]*kmsmodel.BitcoinOutput, 0, len(plan.Outputs))
	for _, output := range plan.Outputs {
		outputs = append(outputs, &kmsmodel.BitcoinOutput{
			Address:    output.Address,
			AmountSats: output.AmountSats,
		})
	}

	res, err := s.kms.CreateBitcoinTransaction(&kmsclient.CreateBitcoinTransactionParams{
		Context:  ctx,
		WalletID: sender.UUID.String(),
		Data: &kmsmodel.CreateBitcoinTransactionRequest{
			Inputs:         inputs,
			Outputs:        outputs,
			FeeSatPerVByte: plan.FeeSatPerVByte,
			IsTest:         isTest,
			RBF:            plan.RBF,
		},
	})
	if err != nil {
		return "", errors.Wrap(err, "unable to create BTC transaction")
	}

	return res.Payload.RawTransaction, nil
}

func (s *Service) activeBitcoinUTXOReservationKeys(
	ctx context.Context,
	walletID int64,
	isTest bool,
) ([]blockchain.BitcoinUTXOKey, error) {
	reservations, err := s.store.ListActiveBitcoinUTXOReservations(ctx, repository.ListActiveBitcoinUTXOReservationsParams{
		WalletID: walletID,
		IsTest:   isTest,
	})
	if err != nil {
		return nil, err
	}

	keys := make([]blockchain.BitcoinUTXOKey, 0, len(reservations))
	for _, reservation := range reservations {
		keys = append(keys, blockchain.BitcoinUTXOKey{
			Hash:  reservation.TxHash,
			Index: uint32(reservation.OutputIndex),
		})
	}

	return keys, nil
}

func (s *Service) reserveBitcoinUTXOs(
	ctx context.Context,
	walletID int64,
	inputs []blockchain.BitcoinUTXO,
	isTest bool,
) ([]int64, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(bitcoinUTXOReservationTTL)
	reservationIDs := make([]int64, 0, len(inputs))

	err := s.store.RunTransaction(ctx, func(ctx context.Context, q repository.Querier) error {
		for _, input := range inputs {
			reservation, err := q.CreateBitcoinUTXOReservation(ctx, repository.CreateBitcoinUTXOReservationParams{
				WalletID:    walletID,
				IsTest:      isTest,
				TxHash:      input.Hash,
				OutputIndex: int64(input.Index),
				AmountSats:  input.AmountSats,
				CreatedAt:   now,
				ExpiresAt:   expiresAt,
			})
			if err != nil {
				return err
			}

			reservationIDs = append(reservationIDs, reservation.ID)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return reservationIDs, nil
}

func (s *Service) attachBitcoinUTXOReservationsRawTX(ctx context.Context, reservationIDs []int64, rawTXID string) error {
	now := time.Now().UTC()
	for _, reservationID := range reservationIDs {
		if err := s.store.AttachBitcoinUTXOReservationRawTX(ctx, repository.AttachBitcoinUTXOReservationRawTXParams{
			ID:        reservationID,
			RawTxID:   rawTXID,
			UpdatedAt: now,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) releaseBitcoinUTXOReservationIDs(ctx context.Context, reservationIDs []int64) error {
	now := time.Now().UTC()
	for _, reservationID := range reservationIDs {
		if err := s.store.ReleaseBitcoinUTXOReservation(ctx, repository.ReleaseBitcoinUTXOReservationParams{
			ID:         reservationID,
			UpdatedAt:  now,
			ReleasedAt: now,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) ReleaseBitcoinTransactionReservations(ctx context.Context, walletID int64, rawTX string, isTest bool) error {
	now := time.Now().UTC()
	return s.store.ReleaseBitcoinUTXOReservationsByRawTX(ctx, repository.ReleaseBitcoinUTXOReservationsByRawTXParams{
		WalletID:   walletID,
		IsTest:     isTest,
		RawTxID:    bitcoinRawTXReservationID(rawTX),
		UpdatedAt:  now,
		ReleasedAt: now,
	})
}

func (s *Service) MarkBitcoinTransactionBroadcasted(
	ctx context.Context,
	walletID int64,
	rawTX string,
	transactionHash string,
	isTest bool,
) error {
	return s.store.MarkBitcoinUTXOReservationsBroadcastedByRawTX(
		ctx,
		repository.MarkBitcoinUTXOReservationsBroadcastedByRawTXParams{
			WalletID:        walletID,
			IsTest:          isTest,
			RawTxID:         bitcoinRawTXReservationID(rawTX),
			TransactionHash: transactionHash,
			UpdatedAt:       time.Now().UTC(),
		},
	)
}

func bitcoinRawTXReservationID(rawTX string) string {
	sum := sha256.Sum256([]byte(rawTX))
	return hex.EncodeToString(sum[:])
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func validateAccountBasedTransactionCurrency(currency money.CryptoCurrency) error {
	switch kms.Blockchain(currency.Blockchain) {
	case kms.ETH, kms.MATIC, kms.BSC, kms.TRON:
		return nil
	case kms.BTC, kms.LTC:
		return errors.Wrapf(ErrUnsupportedTransactionPath, "%s requires a dedicated UTXO transaction flow", currency.Ticker)
	default:
		return errors.Wrapf(ErrUnsupportedTransactionPath, "unsupported currency %s", currency.Ticker)
	}
}

func isUTXOBlockchain(blockchain kms.Blockchain) bool {
	switch blockchain {
	case kms.BTC, kms.LTC:
		return true
	default:
		return false
	}
}

//nolint:gocyclo
func (s *Service) createSignedTransaction(
	ctx context.Context,
	sender *Wallet,
	recipient string,
	currency money.CryptoCurrency,
	amount money.Money,
	fee blockchain.Fee,
	nonce int64,
	isTest bool,
) (string, error) {
	if currency.Blockchain == kms.ETH.ToMoneyBlockchain() {
		networkID, err := strconv.Atoi(currency.ChooseNetwork(isTest))
		if err != nil {
			return "", errors.Wrap(err, "unable to parse network id")
		}

		ethFee, err := fee.ToEthFee()
		if err != nil {
			return "", errors.Wrap(err, "fee is not ETH")
		}

		res, err := s.kms.CreateEthereumTransaction(&kmsclient.CreateEthereumTransactionParams{
			Context:  ctx,
			WalletID: sender.UUID.String(),
			Data: &kmsmodel.CreateEthereumTransactionRequest{
				Amount:            amount.StringRaw(),
				AssetType:         kmsmodel.AssetType(currency.Type),
				ContractAddress:   currency.ChooseContractAddress(isTest),
				Gas:               int64(ethFee.GasUnits),
				MaxFeePerGas:      ethFee.GasPrice,
				MaxPriorityPerGas: ethFee.PriorityFee,
				NetworkID:         int64(networkID),
				Nonce:             util.Ptr(nonce),
				Recipient:         recipient,
			},
		})

		if err != nil {
			return "", errors.Wrap(err, "unable to create ETH transaction")
		}

		return res.Payload.RawTransaction, nil
	}

	if currency.Blockchain == kms.MATIC.ToMoneyBlockchain() {
		networkID, err := strconv.Atoi(currency.ChooseNetwork(isTest))
		if err != nil {
			return "", errors.Wrap(err, "unable to parse network id")
		}

		maticFee, err := fee.ToMaticFee()
		if err != nil {
			return "", errors.Wrap(err, "fee is not MATIC")
		}

		res, err := s.kms.CreateMaticTransaction(&kmsclient.CreateMaticTransactionParams{
			Context:  ctx,
			WalletID: sender.UUID.String(),
			Data: &kmsmodel.CreateMaticTransactionRequest{
				Amount:            amount.StringRaw(),
				AssetType:         kmsmodel.AssetType(currency.Type),
				ContractAddress:   currency.ChooseContractAddress(isTest),
				Gas:               int64(maticFee.GasUnits),
				MaxFeePerGas:      maticFee.GasPrice,
				MaxPriorityPerGas: maticFee.PriorityFee,
				NetworkID:         int64(networkID),
				Nonce:             util.Ptr(nonce),
				Recipient:         recipient,
			},
		})

		if err != nil {
			return "", errors.Wrap(err, "unable to create MATIC transaction")
		}

		return res.Payload.RawTransaction, nil
	}

	if currency.Blockchain == kms.BSC.ToMoneyBlockchain() {
		networkID, err := strconv.Atoi(currency.ChooseNetwork(isTest))
		if err != nil {
			return "", errors.Wrap(err, "unable to parse network id")
		}

		bscFee, err := fee.ToBSCFee()
		if err != nil {
			return "", errors.Wrap(err, "fee is not BSC")
		}

		res, err := s.kms.CreateBSCTransaction(&kmsclient.CreateBSCTransactionParams{
			Context:  ctx,
			WalletID: sender.UUID.String(),
			Data: &kmsmodel.CreateBSCTransactionRequest{
				Amount:            amount.StringRaw(),
				AssetType:         kmsmodel.AssetType(currency.Type),
				ContractAddress:   currency.ChooseContractAddress(isTest),
				Gas:               int64(bscFee.GasUnits),
				MaxFeePerGas:      bscFee.GasPrice,
				MaxPriorityPerGas: bscFee.PriorityFee,
				NetworkID:         int64(networkID),
				Nonce:             util.Ptr(nonce),
				Recipient:         recipient,
			},
		})

		if err != nil {
			return "", errors.Wrap(err, "unable to create BSC transaction")
		}

		return res.Payload.RawTransaction, nil
	}

	if currency.Blockchain == kms.TRON.ToMoneyBlockchain() {
		tronFee, err := fee.ToTronFee()
		if err != nil {
			return "", errors.Wrap(err, "fee is not TRON")
		}

		res, err := s.kms.CreateTronTransaction(&kmsclient.CreateTronTransactionParams{
			Context:  ctx,
			WalletID: sender.UUID.String(),
			Data: &kmsmodel.CreateTronTransactionRequest{
				Amount:          amount.StringRaw(),
				AssetType:       kmsmodel.AssetType(currency.Type),
				ContractAddress: currency.ChooseContractAddress(isTest),
				FeeLimit:        int64(tronFee.FeeLimitSun),
				IsTest:          isTest,
				Recipient:       recipient,
			},
		})

		if err != nil {
			return "", errors.Wrap(err, "unable to create TRON transaction")
		}

		resAsBytes, err := json.Marshal(res.Payload)
		if err != nil {
			return "", errors.Wrap(err, "unable to marshal TRON transaction")
		}

		return string(resAsBytes), nil
	}

	return "", errors.New("unsupported currency " + currency.Ticker)
}
