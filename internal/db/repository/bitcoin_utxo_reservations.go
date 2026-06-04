package repository

import (
	"context"
	"database/sql"
	"time"
)

const bitcoinUTXOReservationColumns = `
id, wallet_id, is_test, tx_hash, output_index, amount_sats, raw_tx_id, transaction_hash,
status, created_at, updated_at, expires_at, released_at`

type BitcoinUTXOReservation struct {
	ID              int64
	WalletID        int64
	IsTest          bool
	TxHash          string
	OutputIndex     int64
	AmountSats      int64
	RawTxID         sql.NullString
	TransactionHash sql.NullString
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ExpiresAt       time.Time
	ReleasedAt      sql.NullTime
}

type CreateBitcoinUTXOReservationParams struct {
	WalletID    int64
	IsTest      bool
	TxHash      string
	OutputIndex int64
	AmountSats  int64
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

func (q *Queries) CreateBitcoinUTXOReservation(
	ctx context.Context,
	arg CreateBitcoinUTXOReservationParams,
) (BitcoinUTXOReservation, error) {
	row := q.db.QueryRow(ctx, `
insert into bitcoin_utxo_reservations (
    wallet_id, is_test, tx_hash, output_index, amount_sats,
    status, created_at, updated_at, expires_at
) values ($1, $2, $3, $4, $5, 'reserved', $6, $6, $7)
returning `+bitcoinUTXOReservationColumns,
		arg.WalletID,
		arg.IsTest,
		arg.TxHash,
		arg.OutputIndex,
		arg.AmountSats,
		arg.CreatedAt,
		arg.ExpiresAt,
	)

	return scanBitcoinUTXOReservation(row)
}

type ListActiveBitcoinUTXOReservationsParams struct {
	WalletID int64
	IsTest   bool
}

func (q *Queries) ListActiveBitcoinUTXOReservations(
	ctx context.Context,
	arg ListActiveBitcoinUTXOReservationsParams,
) ([]BitcoinUTXOReservation, error) {
	rows, err := q.db.Query(ctx, `
select `+bitcoinUTXOReservationColumns+`
from bitcoin_utxo_reservations
where wallet_id = $1
  and is_test = $2
  and (
      status = 'broadcasted'
      or (status = 'reserved' and expires_at > now())
  )
order by id`, arg.WalletID, arg.IsTest)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reservations := make([]BitcoinUTXOReservation, 0)
	for rows.Next() {
		reservation, err := scanBitcoinUTXOReservation(rows)
		if err != nil {
			return nil, err
		}
		reservations = append(reservations, reservation)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reservations, nil
}

type AttachBitcoinUTXOReservationRawTXParams struct {
	ID        int64
	RawTxID   string
	UpdatedAt time.Time
}

func (q *Queries) AttachBitcoinUTXOReservationRawTX(
	ctx context.Context,
	arg AttachBitcoinUTXOReservationRawTXParams,
) error {
	_, err := q.db.Exec(ctx, `
update bitcoin_utxo_reservations
set raw_tx_id = $2, updated_at = $3
where id = $1 and status = 'reserved'`, arg.ID, arg.RawTxID, arg.UpdatedAt)

	return err
}

type ReleaseBitcoinUTXOReservationParams struct {
	ID         int64
	UpdatedAt  time.Time
	ReleasedAt time.Time
}

func (q *Queries) ReleaseBitcoinUTXOReservation(ctx context.Context, arg ReleaseBitcoinUTXOReservationParams) error {
	_, err := q.db.Exec(ctx, `
update bitcoin_utxo_reservations
set status = 'released', updated_at = $2, released_at = $3
where id = $1 and status = 'reserved'`, arg.ID, arg.UpdatedAt, arg.ReleasedAt)

	return err
}

type ReleaseBitcoinUTXOReservationsByRawTXParams struct {
	WalletID   int64
	IsTest     bool
	RawTxID    string
	UpdatedAt  time.Time
	ReleasedAt time.Time
}

func (q *Queries) ReleaseBitcoinUTXOReservationsByRawTX(
	ctx context.Context,
	arg ReleaseBitcoinUTXOReservationsByRawTXParams,
) error {
	_, err := q.db.Exec(ctx, `
update bitcoin_utxo_reservations
set status = 'released', updated_at = $4, released_at = $5
where wallet_id = $1
  and is_test = $2
  and raw_tx_id = $3
  and status = 'reserved'`, arg.WalletID, arg.IsTest, arg.RawTxID, arg.UpdatedAt, arg.ReleasedAt)

	return err
}

type MarkBitcoinUTXOReservationsBroadcastedByRawTXParams struct {
	WalletID        int64
	IsTest          bool
	RawTxID         string
	TransactionHash string
	UpdatedAt       time.Time
}

func (q *Queries) MarkBitcoinUTXOReservationsBroadcastedByRawTX(
	ctx context.Context,
	arg MarkBitcoinUTXOReservationsBroadcastedByRawTXParams,
) error {
	_, err := q.db.Exec(ctx, `
update bitcoin_utxo_reservations
set status = 'broadcasted', transaction_hash = $4, updated_at = $5
where wallet_id = $1
  and is_test = $2
  and raw_tx_id = $3
  and status = 'reserved'`, arg.WalletID, arg.IsTest, arg.RawTxID, arg.TransactionHash, arg.UpdatedAt)

	return err
}

type bitcoinUTXOReservationScanner interface {
	Scan(dest ...interface{}) error
}

func scanBitcoinUTXOReservation(row bitcoinUTXOReservationScanner) (BitcoinUTXOReservation, error) {
	var reservation BitcoinUTXOReservation
	err := row.Scan(
		&reservation.ID,
		&reservation.WalletID,
		&reservation.IsTest,
		&reservation.TxHash,
		&reservation.OutputIndex,
		&reservation.AmountSats,
		&reservation.RawTxID,
		&reservation.TransactionHash,
		&reservation.Status,
		&reservation.CreatedAt,
		&reservation.UpdatedAt,
		&reservation.ExpiresAt,
		&reservation.ReleasedAt,
	)
	return reservation, err
}
