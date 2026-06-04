package processing

import (
	"context"

	"github.com/oxygenpay/oxygen/internal/service/blockchain"
	"github.com/oxygenpay/oxygen/internal/service/transaction"
	"github.com/pkg/errors"
)

type ObservedTransaction struct {
	TransactionID         string
	SenderAddress         string
	Amount                string
	AmountFormatted       string
	Confirmations         int64
	RequiredConfirmations int64
	IsConfirmed           bool
	IsMempool             bool
	ExplorerLink          string
}

func (s *Service) ObserveIncomingTransaction(
	ctx context.Context,
	tx *transaction.Transaction,
) (*ObservedTransaction, error) {
	if !canObserveIncomingTransaction(tx) {
		return nil, nil
	}

	incoming, err := s.blockchain.ListIncomingTransactions(ctx, tx.RecipientAddress, tx.Currency, tx.IsTest)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list incoming blockchain transactions")
	}

	for _, candidate := range incoming {
		ok, err := s.canUseObservedCandidate(ctx, tx, candidate)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		return s.observedTransactionFromCandidate(ctx, tx, candidate)
	}

	return nil, nil
}

func canObserveIncomingTransaction(tx *transaction.Transaction) bool {
	return tx != nil &&
		tx.Type == transaction.TypeIncoming &&
		tx.Status == transaction.StatusPending &&
		tx.HashID == nil &&
		tx.RecipientWalletID != nil
}

func (s *Service) canUseObservedCandidate(
	ctx context.Context,
	tx *transaction.Transaction,
	candidate blockchain.IncomingTransaction,
) (bool, error) {
	if candidate.TransactionID == "" ||
		candidate.NetworkID != tx.NetworkID() ||
		candidate.Currency.Ticker != tx.Currency.Ticker {
		return false, nil
	}

	existing, err := s.transactions.GetByHash(ctx, candidate.NetworkID, candidate.TransactionID)
	switch {
	case err == nil:
		return existing.ID == tx.ID, nil
	case errors.Is(err, transaction.ErrNotFound):
		return true, nil
	default:
		return false, errors.Wrap(err, "unable to check transaction hash")
	}
}

func (s *Service) observedTransactionFromCandidate(
	ctx context.Context,
	tx *transaction.Transaction,
	candidate blockchain.IncomingTransaction,
) (*ObservedTransaction, error) {
	receipt, err := s.blockchain.GetTransactionReceipt(
		ctx,
		tx.Currency.Blockchain,
		candidate.TransactionID,
		tx.IsTest,
	)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get transaction receipt")
	}

	requiredConfirmations, err := blockchain.RequiredConfirmations(tx.Currency.Blockchain)
	if err != nil {
		return nil, err
	}

	explorerLink, err := blockchain.CreateExplorerTXLink(
		tx.Currency.Blockchain,
		tx.NetworkID(),
		candidate.TransactionID,
	)
	if err != nil {
		explorerLink = ""
	}

	return &ObservedTransaction{
		TransactionID:         candidate.TransactionID,
		SenderAddress:         candidate.SenderAddress,
		Amount:                candidate.Amount.StringRaw(),
		AmountFormatted:       candidate.Amount.String(),
		Confirmations:         receipt.Confirmations,
		RequiredConfirmations: requiredConfirmations,
		IsConfirmed:           receipt.IsConfirmed,
		IsMempool:             candidate.IsMempool,
		ExplorerLink:          explorerLink,
	}, nil
}
