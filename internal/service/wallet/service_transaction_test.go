package wallet

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oxygenpay/oxygen/internal/db/repository"
	kms "github.com/oxygenpay/oxygen/internal/kms/wallet"
	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/service/blockchain"
	kmswallet "github.com/oxygenpay/oxygen/pkg/api-kms/v1/client/wallet"
	kmsmock "github.com/oxygenpay/oxygen/pkg/api-kms/v1/mock"
	kmsmodel "github.com/oxygenpay/oxygen/pkg/api-kms/v1/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateSignedTransaction_BitcoinUsesUTXOPathWithoutNonceTracking(t *testing.T) {
	const rawTX = "010000000001..."

	recipient := "tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx"
	senderID := uuid.New()
	currency := money.CryptoCurrency{
		Blockchain:     money.Blockchain(kms.BTC),
		BlockchainName: "Bitcoin",
		NetworkID:      "mainnet",
		TestNetworkID:  "testnet",
		Type:           money.Coin,
		Ticker:         "BTC",
		Name:           "BTC",
		Decimals:       8,
	}

	amount, err := money.CryptoFromRaw("BTC", "123", 8)
	require.NoError(t, err)

	fee := blockchain.NewFee(currency, time.Now(), true, blockchain.BitcoinFee{
		FeeSatPerVByte: "2",
	})

	plan := blockchain.BitcoinTransactionPlan{
		Inputs: []blockchain.BitcoinUTXO{{
			Hash:       "1c99a4e6f1dcf8ef2e3db3d2db5ca7dd4f78edb5a8f42fb6f7f4d89af2b2b923",
			Index:      0,
			AmountSats: 1000,
			Address:    "tb1q82mck8gdey7wx78wtyl364v9gd7gyhg7k3w977",
		}},
		Outputs: []blockchain.BitcoinOutput{{
			Address:    recipient,
			AmountSats: 123,
		}},
		FeeSatPerVByte: 2,
		RBF:            true,
	}

	blockchainService := &bitcoinBlockchainStub{plan: plan}
	kmsClient := &kmsmock.ClientService{}
	kmsClient.On("CreateBitcoinTransaction", mock.MatchedBy(func(params *kmswallet.CreateBitcoinTransactionParams) bool {
		return params.WalletID == senderID.String() &&
			params.Data.IsTest &&
			params.Data.RBF &&
			params.Data.FeeSatPerVByte == 2 &&
			len(params.Data.Inputs) == 1 &&
			len(params.Data.Outputs) == 1
	})).Return(&kmswallet.CreateBitcoinTransactionCreated{
		Payload: &kmsmodel.BitcoinTransaction{RawTransaction: rawTX},
	}, nil).Once()

	store := &bitcoinReservationStorageStub{}
	s := &Service{kms: kmsClient, blockchain: blockchainService, store: store}

	txRaw, err := s.CreateSignedTransaction(
		context.Background(),
		&Wallet{ID: 1, UUID: senderID, Address: "tb1q82mck8gdey7wx78wtyl364v9gd7gyhg7k3w977", Blockchain: kms.BTC},
		recipient,
		currency,
		amount,
		fee,
		true,
	)

	require.NoError(t, err)
	assert.Equal(t, rawTX, txRaw)
	assert.Equal(t, "tb1q82mck8gdey7wx78wtyl364v9gd7gyhg7k3w977", blockchainService.senderAddress)
	assert.Equal(t, recipient, blockchainService.recipient)
	require.Len(t, store.reservations, 1)
	assert.Equal(t, "1c99a4e6f1dcf8ef2e3db3d2db5ca7dd4f78edb5a8f42fb6f7f4d89af2b2b923", store.reservations[0].TxHash)
	assert.Equal(t, bitcoinRawTXReservationID(rawTX), store.reservations[0].RawTxID.String)
	kmsClient.AssertExpectations(t)
}

func TestValidateAccountBasedTransactionCurrency(t *testing.T) {
	for _, tc := range []struct {
		name          string
		blockchain    kms.Blockchain
		expectErr     bool
		errorContains string
	}{
		{name: "ethereum", blockchain: kms.ETH},
		{name: "matic", blockchain: kms.MATIC},
		{name: "bsc", blockchain: kms.BSC},
		{name: "tron", blockchain: kms.TRON},
		{name: "bitcoin", blockchain: kms.BTC, expectErr: true, errorContains: "UTXO"},
		{name: "litecoin", blockchain: kms.LTC, expectErr: true, errorContains: "UTXO"},
		{name: "unknown", blockchain: kms.Blockchain("DOGE"), expectErr: true, errorContains: "unsupported currency"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAccountBasedTransactionCurrency(money.CryptoCurrency{
				Blockchain: money.Blockchain(tc.blockchain),
				Ticker:     tc.blockchain.String(),
			})

			if tc.expectErr {
				assert.ErrorIs(t, err, ErrUnsupportedTransactionPath)
				assert.ErrorContains(t, err, tc.errorContains)
				return
			}

			assert.NoError(t, err)
		})
	}
}

type bitcoinBlockchainStub struct {
	plan          blockchain.BitcoinTransactionPlan
	senderAddress string
	recipient     string
}

func (s *bitcoinBlockchainStub) PrepareBitcoinTransaction(
	_ context.Context,
	senderAddress string,
	recipient string,
	_ money.Money,
	_ blockchain.Fee,
	_ bool,
) (blockchain.BitcoinTransactionPlan, error) {
	s.senderAddress = senderAddress
	s.recipient = recipient

	return s.plan, nil
}

func (s *bitcoinBlockchainStub) PrepareBitcoinTransactionExcluding(
	_ context.Context,
	senderAddress string,
	recipient string,
	_ money.Money,
	_ blockchain.Fee,
	_ bool,
	_ []blockchain.BitcoinUTXOKey,
) (blockchain.BitcoinTransactionPlan, error) {
	s.senderAddress = senderAddress
	s.recipient = recipient

	return s.plan, nil
}

func (s *bitcoinBlockchainStub) PrepareBitcoinSweepTransactionExcluding(
	_ context.Context,
	senderAddress string,
	recipient string,
	_ blockchain.Fee,
	_ bool,
	_ []blockchain.BitcoinUTXOKey,
) (blockchain.BitcoinTransactionPlan, error) {
	s.senderAddress = senderAddress
	s.recipient = recipient

	return s.plan, nil
}

func (s *bitcoinBlockchainStub) MaxBitcoinTransactionAmountExcluding(
	context.Context,
	string,
	string,
	money.CryptoCurrency,
	blockchain.Fee,
	bool,
	money.Money,
	[]blockchain.BitcoinUTXOKey,
) (money.Money, error) {
	panic("unexpected call")
}

func (s *bitcoinBlockchainStub) GetExchangeRate(context.Context, string, string) (blockchain.ExchangeRate, error) {
	panic("unexpected call")
}

func (s *bitcoinBlockchainStub) Convert(context.Context, string, string, string) (blockchain.Conversion, error) {
	panic("unexpected call")
}

func (s *bitcoinBlockchainStub) FiatToFiat(context.Context, money.Money, money.FiatCurrency) (blockchain.Conversion, error) {
	panic("unexpected call")
}

func (s *bitcoinBlockchainStub) FiatToCrypto(context.Context, money.Money, money.CryptoCurrency) (blockchain.Conversion, error) {
	panic("unexpected call")
}

func (s *bitcoinBlockchainStub) CryptoToFiat(context.Context, money.Money, money.FiatCurrency) (blockchain.Conversion, error) {
	panic("unexpected call")
}

type bitcoinReservationStorageStub struct {
	repository.Storage
	nextID       int64
	reservations []repository.BitcoinUTXOReservation
}

func (s *bitcoinReservationStorageStub) RunTransaction(
	ctx context.Context,
	callback repository.TxCallback,
) error {
	return callback(ctx, s)
}

func (s *bitcoinReservationStorageStub) ListActiveBitcoinUTXOReservations(
	context.Context,
	repository.ListActiveBitcoinUTXOReservationsParams,
) ([]repository.BitcoinUTXOReservation, error) {
	return nil, nil
}

func (s *bitcoinReservationStorageStub) CreateBitcoinUTXOReservation(
	_ context.Context,
	arg repository.CreateBitcoinUTXOReservationParams,
) (repository.BitcoinUTXOReservation, error) {
	s.nextID++
	reservation := repository.BitcoinUTXOReservation{
		ID:          s.nextID,
		WalletID:    arg.WalletID,
		IsTest:      arg.IsTest,
		TxHash:      arg.TxHash,
		OutputIndex: arg.OutputIndex,
		AmountSats:  arg.AmountSats,
		Status:      "reserved",
		CreatedAt:   arg.CreatedAt,
		UpdatedAt:   arg.CreatedAt,
		ExpiresAt:   arg.ExpiresAt,
	}
	s.reservations = append(s.reservations, reservation)

	return reservation, nil
}

func (s *bitcoinReservationStorageStub) AttachBitcoinUTXOReservationRawTX(
	_ context.Context,
	arg repository.AttachBitcoinUTXOReservationRawTXParams,
) error {
	for i := range s.reservations {
		if s.reservations[i].ID == arg.ID {
			s.reservations[i].RawTxID.Valid = true
			s.reservations[i].RawTxID.String = arg.RawTxID
		}
	}

	return nil
}

func (s *bitcoinReservationStorageStub) ReleaseBitcoinUTXOReservation(
	context.Context,
	repository.ReleaseBitcoinUTXOReservationParams,
) error {
	return nil
}

func (s *bitcoinReservationStorageStub) ReleaseBitcoinUTXOReservationsByRawTX(
	context.Context,
	repository.ReleaseBitcoinUTXOReservationsByRawTXParams,
) error {
	return nil
}

func (s *bitcoinReservationStorageStub) MarkBitcoinUTXOReservationsBroadcastedByRawTX(
	context.Context,
	repository.MarkBitcoinUTXOReservationsBroadcastedByRawTXParams,
) error {
	return nil
}
