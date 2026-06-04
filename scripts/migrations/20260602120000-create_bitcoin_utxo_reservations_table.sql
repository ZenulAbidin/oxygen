-- +migrate Up
create table if not exists bitcoin_utxo_reservations
(
    id               bigserial constraint bitcoin_utxo_reservations_pkey primary key,
    wallet_id        bigint       not null references wallets (id),
    is_test          bool         not null,
    tx_hash          varchar(128) not null,
    output_index     bigint       not null check (output_index >= 0 and output_index <= 4294967295),
    amount_sats      bigint       not null check (amount_sats > 0),
    raw_tx_id        varchar(64)  null,
    transaction_hash varchar(128) null,
    status           varchar(16)  not null default 'reserved'
        constraint bitcoin_utxo_reservations_status_check check (status in ('reserved', 'broadcasted', 'released')),
    created_at       timestamp    not null,
    updated_at       timestamp    not null,
    expires_at       timestamp    not null,
    released_at      timestamp    null
);

create unique index bitcoin_utxo_reservations_active_unique
    on bitcoin_utxo_reservations (wallet_id, is_test, tx_hash, output_index)
    where status in ('reserved', 'broadcasted');

create index bitcoin_utxo_reservations_wallet_status
    on bitcoin_utxo_reservations (wallet_id, is_test, status);

create index bitcoin_utxo_reservations_raw_tx_id
    on bitcoin_utxo_reservations (raw_tx_id)
    where raw_tx_id is not null;

-- +migrate Down
drop index if exists bitcoin_utxo_reservations_raw_tx_id;
drop index if exists bitcoin_utxo_reservations_wallet_status;
drop index if exists bitcoin_utxo_reservations_active_unique;
drop table if exists bitcoin_utxo_reservations;
