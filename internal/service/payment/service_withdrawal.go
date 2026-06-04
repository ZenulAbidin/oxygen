package payment

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/oxygenpay/oxygen/internal/bus"
	"github.com/oxygenpay/oxygen/internal/db/repository"
	kmswallet "github.com/oxygenpay/oxygen/internal/kms/wallet"
	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/service/blockchain"
	"github.com/oxygenpay/oxygen/internal/service/wallet"
	"github.com/oxygenpay/oxygen/internal/util"
	"github.com/pkg/errors"
)

type CreateWithdrawalProps struct {
	BalanceID uuid.UUID
	AddressID uuid.UUID
	AmountRaw string // "0.123"
}

func (s *Service) ListWithdrawals(ctx context.Context, status Status, filterByIDs []int64) ([]*Payment, error) {
	results, err := s.repo.GetPaymentsByType(ctx, repository.GetPaymentsByTypeParams{
		Type:        string(TypeWithdrawal),
		Status:      string(status),
		FilterByIds: len(filterByIDs) > 0,
		ID:          util.MapSlice(filterByIDs, func(id int64) int32 { return int32(id) }),
		Limit:       200,
	})

	if err != nil {
		return nil, err
	}

	if len(filterByIDs) > 0 && len(results) != len(filterByIDs) {
		return nil, fmt.Errorf("withdrawals filter mismatch for status %q", status)
	}

	payments := make([]*Payment, len(results))
	for i := range results {
		pt, err := s.entryToPayment(results[i])
		if err != nil {
			return nil, err
		}

		payments[i] = pt
	}

	return payments, nil
}

func (s *Service) CreateWithdrawal(ctx context.Context, merchantID int64, props CreateWithdrawalProps) (*Payment, error) {
	// 1. Resolve address, balance & parse amount
	address, err := s.merchants.GetMerchantAddressByUUID(ctx, merchantID, props.AddressID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get merchant address")
	}

	balance, err := s.wallets.GetMerchantBalanceByUUID(ctx, merchantID, props.BalanceID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get merchant balance")
	}

	if string(address.Blockchain) != balance.Network {
		return nil, ErrAddressBalanceMismatch
	}

	ticker := balance.Amount.Ticker()

	amount, err := money.CryptoFromStringFloat(ticker, props.AmountRaw, balance.Amount.Decimals())
	if err != nil {
		return nil, err
	}
	if !amount.CompatibleTo(balance.Amount) {
		return nil, ErrAddressBalanceMismatch
	}

	currency, err := s.blockchain.GetCurrencyByTicker(ticker)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get currency %q", ticker)
	}
	if err := validateWithdrawalRuntimeCurrency(currency); err != nil {
		return nil, err
	}

	// 2. Check if balance has sufficient funds
	withdrawalFee, err := s.getWithdrawalFee(ctx, balance, currency)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get withdrawal fee")
	}
	if errCovers := balance.Covers(amount, withdrawalFee.CryptoFee); errCovers != nil {
		return nil, errors.WithMessagef(
			ErrWithdrawalInsufficientBalance,
			"balance of %s %s is less than requested %s %s + withdrawal fee of %s %s ($%s)",
			balance.Amount.String(),
			balance.Amount.Ticker(),
			amount.String(),
			amount.Ticker(),
			withdrawalFee.CryptoFee.String(),
			withdrawalFee.CryptoFee.Ticker(),
			withdrawalFee.USDFee.String(),
		)
	}

	// The minimum withdrawal follows the currently estimated withdrawal fee,
	// not a static USD floor.
	if !amount.CompatibleTo(withdrawalFee.CryptoFee) {
		return nil, money.ErrIncompatibleMoney
	}

	if amount.LessThan(withdrawalFee.CryptoFee) {
		return nil, errors.Wrapf(
			ErrWithdrawalAmountTooSmall,
			"minimum withdrawal amount is %s %s (estimated withdrawal fee)",
			withdrawalFee.CryptoFee.String(),
			withdrawalFee.CryptoFee.Ticker(),
		)
	}

	// 3. Create withdrawal
	publicID := uuid.New()
	isTest := balance.NetworkID != currency.NetworkID

	p, err := s.repo.CreatePayment(ctx, repository.CreatePaymentParams{
		PublicID:          publicID,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Type:              TypeWithdrawal.String(),
		Status:            StatusPending.String(),
		MerchantID:        merchantID,
		MerchantOrderUuid: publicID,
		Price:             repository.MoneyToNumeric(amount),
		Decimals:          int32(amount.Decimals()),
		Currency:          amount.Ticker(),
		Description:       repository.StringToNullable("Balance withdrawal"),
		IsTest:            isTest,
		Metadata: Metadata{
			MetaBalanceID: strconv.Itoa(int(balance.ID)),
			MetaAddressID: strconv.Itoa(int(address.ID)),
		}.ToJSONB(),
	})

	if err != nil {
		return nil, errors.Wrap(err, "unable to create payment")
	}

	err = s.publisher.Publish(bus.TopicWithdrawals, bus.WithdrawalCreatedEvent{
		MerchantID: p.MerchantID,
		PaymentID:  p.ID,
	})

	if err != nil {
		return nil, errors.Wrap(err, "unable to publish WithdrawalCreatedEvent event")
	}

	return s.entryToPayment(p)
}

type WithdrawalFee struct {
	CalculatedAt time.Time
	Blockchain   money.Blockchain
	Currency     string
	USDFee       money.Money
	CryptoFee    money.Money
	IsTest       bool
}

func (s *Service) GetWithdrawalFee(ctx context.Context, merchantID int64, balanceID uuid.UUID) (*WithdrawalFee, error) {
	balance, err := s.wallets.GetMerchantBalanceByUUID(ctx, merchantID, balanceID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get merchant balance")
	}

	// e.g. ETH_USDT
	currency, err := s.blockchain.GetCurrencyByTicker(balance.Currency)
	if err != nil {
		return nil, errors.Wrap(err, "unable to  get currency by ticker")
	}
	if err := validateWithdrawalRuntimeCurrency(currency); err != nil {
		return nil, err
	}

	return s.getWithdrawalFee(ctx, balance, currency)
}

func (s *Service) getWithdrawalFee(
	ctx context.Context,
	balance *wallet.Balance,
	currency money.CryptoCurrency,
) (*WithdrawalFee, error) {
	baseCurrency, err := s.blockchain.GetNativeCoin(currency.Blockchain)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get currency by ticker")
	}

	isTest := balance.NetworkID != currency.NetworkID

	if isUTXOBlockchain(currency.Blockchain) {
		networkFee, err := s.blockchain.CalculateFee(ctx, baseCurrency, currency, isTest)
		if err != nil {
			return nil, errors.Wrap(err, "unable to get fee")
		}

		utxoFee, err := networkFee.ToBitcoinFee()
		if err != nil {
			return nil, errors.Wrap(err, "unable to parse UTXO fee")
		}

		cryptoFee, err := currency.MakeAmount(utxoFee.TotalCostSats)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to make %s withdrawal fee", currency.Ticker)
		}

		conv, err := s.blockchain.CryptoToFiat(ctx, cryptoFee, money.USD)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to convert %s fee to USD", currency.Ticker)
		}

		return &WithdrawalFee{
			CalculatedAt: time.Now(),
			Blockchain:   currency.Blockchain,
			Currency:     currency.Ticker,
			IsTest:       isTest,
			USDFee:       conv.To,
			CryptoFee:    cryptoFee,
		}, nil
	}

	usdFee, err := s.blockchain.CalculateWithdrawalFeeUSD(ctx, baseCurrency, currency, isTest)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get fee")
	}

	conv, err := s.blockchain.FiatToCrypto(ctx, usdFee, currency)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get currency withdrawal fee in crypto")
	}

	return &WithdrawalFee{
		CalculatedAt: time.Now(),
		Blockchain:   currency.Blockchain,
		Currency:     currency.Ticker,
		IsTest:       isTest,
		USDFee:       usdFee,
		CryptoFee:    conv.To,
	}, nil
}

func isUTXOBlockchain(blockchain money.Blockchain) bool {
	switch kmswallet.Blockchain(blockchain) {
	case kmswallet.BTC, kmswallet.LTC:
		return true
	default:
		return false
	}
}

func validateWithdrawalRuntimeCurrency(currency money.CryptoCurrency) error {
	if err := blockchain.ValidateTransactionRuntimeBlockchain(currency.Blockchain); err != nil {
		return errors.Wrapf(ErrValidation, "%s withdrawals require dedicated runtime support", currency.Blockchain)
	}

	return nil
}
