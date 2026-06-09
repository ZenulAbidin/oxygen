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

	if !amount.CompatibleTo(withdrawalFee.MinimumAmount) || !amount.CompatibleTo(withdrawalFee.MaximumAmount) {
		return nil, money.ErrIncompatibleMoney
	}

	if amount.LessThan(withdrawalFee.MinimumAmount) {
		return nil, errors.Wrapf(
			ErrWithdrawalAmountTooSmall,
			"minimum withdrawal amount is %s %s",
			withdrawalFee.MinimumAmount.String(),
			withdrawalFee.MinimumAmount.Ticker(),
		)
	}

	if amount.GreaterThan(withdrawalFee.MaximumAmount) {
		return nil, errors.WithMessagef(
			ErrWithdrawalInsufficientBalance,
			"maximum withdrawal amount is %s %s for currently spendable funds",
			withdrawalFee.MaximumAmount.String(),
			withdrawalFee.MaximumAmount.Ticker(),
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
	CalculatedAt  time.Time
	Blockchain    money.Blockchain
	Currency      string
	USDFee        money.Money
	CryptoFee     money.Money
	MinimumAmount money.Money
	MaximumAmount money.Money
	IsTest        bool
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
	minimumAmount, err := minimumWithdrawalAmount(currency)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to make %s minimum withdrawal amount", currency.Ticker)
	}

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

		limit, err := s.maxUTXOWithdrawalLimit(ctx, balance, currency, networkFee, isTest, "")
		if err != nil {
			return nil, errors.Wrapf(err, "unable to calculate %s max withdrawal amount", currency.Ticker)
		}
		if !limit.CryptoFee.IsZero() {
			cryptoFee = limit.CryptoFee
		}

		conv, err := s.blockchain.CryptoToFiat(ctx, cryptoFee, money.USD)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to convert %s fee to USD", currency.Ticker)
		}

		return &WithdrawalFee{
			CalculatedAt:  time.Now(),
			Blockchain:    currency.Blockchain,
			Currency:      currency.Ticker,
			IsTest:        isTest,
			USDFee:        conv.To,
			CryptoFee:     cryptoFee,
			MinimumAmount: minimumAmount,
			MaximumAmount: limit.MaximumAmount,
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

	maximumAmount, err := balance.Amount.Sub(conv.To)
	if errors.Is(err, money.ErrNegative) {
		maximumAmount, err = currency.MakeAmount("0")
	}
	if err != nil {
		return nil, errors.Wrapf(err, "unable to calculate %s max withdrawal amount", currency.Ticker)
	}

	return &WithdrawalFee{
		CalculatedAt:  time.Now(),
		Blockchain:    currency.Blockchain,
		Currency:      currency.Ticker,
		IsTest:        isTest,
		USDFee:        usdFee,
		CryptoFee:     conv.To,
		MinimumAmount: minimumAmount,
		MaximumAmount: maximumAmount,
	}, nil
}

func minimumWithdrawalAmount(currency money.CryptoCurrency) (money.Money, error) {
	if isUTXOBlockchain(currency.Blockchain) {
		return currency.MakeAmount(strconv.FormatInt(blockchain.UTXODustSats, 10))
	}

	return currency.MakeAmount("0")
}

type utxoWithdrawalSource struct {
	Wallet  *wallet.Wallet
	Balance *wallet.Balance
}

type utxoWithdrawalLimit struct {
	MaximumAmount money.Money
	CryptoFee     money.Money
}

func (s *Service) maxUTXOWithdrawalLimit(
	ctx context.Context,
	merchantBalance *wallet.Balance,
	currency money.CryptoCurrency,
	fee blockchain.Fee,
	isTest bool,
	recipient string,
) (utxoWithdrawalLimit, error) {
	maximum, err := currency.MakeAmount("0")
	if err != nil {
		return utxoWithdrawalLimit{}, err
	}
	maximumFee, err := currency.MakeAmount("0")
	if err != nil {
		return utxoWithdrawalLimit{}, err
	}

	consolidatedInputs := make([]blockchain.BitcoinUTXO, 0)
	sources, err := s.listUTXOWithdrawalSources(ctx, merchantBalance, currency, isTest)
	if err != nil {
		return utxoWithdrawalLimit{}, err
	}

	for _, source := range sources {
		maxTotalCost := source.Balance.Amount
		if merchantBalance.Amount.LessThan(maxTotalCost) {
			maxTotalCost = merchantBalance.Amount
		}
		if maxTotalCost.IsZero() {
			continue
		}

		utxos, err := s.wallets.SpendableUTXOs(ctx, source.Wallet, fee, isTest)
		if err != nil {
			s.logger.Warn().Err(err).
				Int64("wallet_id", source.Wallet.ID).
				Str("wallet_type", string(source.Wallet.Type)).
				Str("currency", currency.Ticker).
				Msg("skipping UTXO withdrawal source: unable to list spendable UTXOs")
			continue
		}

		if len(utxos) == 0 {
			continue
		}

		spendable, spendableFee, err := blockchain.MaxBitcoinTransactionAmountAndFeeFromUTXOs(currency, fee, maxTotalCost, utxos)
		if err != nil {
			s.logger.Warn().Err(err).
				Int64("wallet_id", source.Wallet.ID).
				Str("wallet_type", string(source.Wallet.Type)).
				Str("currency", currency.Ticker).
				Msg("skipping direct UTXO withdrawal source")
		} else if spendable.GreaterThan(maximum) {
			maximum = spendable
			maximumFee = spendableFee
		}

		if source.Wallet.Type == wallet.TypeOutbound {
			consolidatedInputs = append(consolidatedInputs, utxos...)
			continue
		}

		sweepAmount, err := blockchain.BitcoinSweepAmountFromUTXOs(currency, fee, utxos)
		if err != nil {
			s.logger.Warn().Err(err).
				Int64("wallet_id", source.Wallet.ID).
				Str("wallet_type", string(source.Wallet.Type)).
				Str("currency", currency.Ticker).
				Msg("skipping UTXO consolidation source")
			continue
		}

		amountSats, err := withdrawalUTXOAmountSats(sweepAmount)
		if err != nil {
			return utxoWithdrawalLimit{}, err
		}
		if amountSats < blockchain.UTXODustSats {
			continue
		}

		consolidatedInputs = append(consolidatedInputs, blockchain.BitcoinUTXO{
			Hash:       fmt.Sprintf("consolidated-%d", source.Wallet.ID),
			Index:      uint32(len(consolidatedInputs)),
			AmountSats: amountSats,
		})
	}

	if len(consolidatedInputs) == 0 {
		return utxoWithdrawalLimit{MaximumAmount: maximum, CryptoFee: maximumFee}, nil
	}

	consolidatedMaximum, consolidatedFee, err := blockchain.MaxBitcoinTransactionAmountAndFeeFromUTXOs(
		currency,
		fee,
		merchantBalance.Amount,
		consolidatedInputs,
	)
	if err != nil {
		if errors.Is(err, blockchain.ErrInsufficientFunds) {
			return utxoWithdrawalLimit{MaximumAmount: maximum, CryptoFee: maximumFee}, nil
		}

		return utxoWithdrawalLimit{}, err
	}

	if consolidatedMaximum.GreaterThan(maximum) {
		maximum = consolidatedMaximum
		maximumFee = consolidatedFee
	}

	return utxoWithdrawalLimit{MaximumAmount: maximum, CryptoFee: maximumFee}, nil
}

func withdrawalUTXOAmountSats(amount money.Money) (int64, error) {
	if amount.Decimals() != 8 {
		return 0, errors.Wrapf(ErrValidation, "%s UTXO amount must use 8 decimals", amount.Ticker())
	}

	value, err := strconv.ParseInt(amount.StringRaw(), 10, 64)
	if err != nil {
		return 0, errors.Wrapf(ErrValidation, "unable to parse %s UTXO amount", amount.Ticker())
	}

	return value, nil
}

func (s *Service) listUTXOWithdrawalSources(
	ctx context.Context,
	merchantBalance *wallet.Balance,
	currency money.CryptoCurrency,
	isTest bool,
) ([]utxoWithdrawalSource, error) {
	var (
		sources   []utxoWithdrawalSource
		start     int64
		networkID = currency.ChooseNetwork(isTest)
	)

	for {
		wallets, nextID, err := s.wallets.List(ctx, wallet.Pagination{
			Start:              start,
			Limit:              300,
			FilterByBlockchain: kmswallet.Blockchain(currency.Blockchain),
		})
		if err != nil {
			return nil, errors.Wrap(err, "unable to list UTXO wallets")
		}

		for _, sourceWallet := range wallets {
			if sourceWallet.Type != wallet.TypeOutbound && sourceWallet.Type != wallet.TypeInbound {
				continue
			}

			balances, err := s.wallets.ListBalances(ctx, wallet.EntityTypeWallet, sourceWallet.ID, false)
			if err != nil {
				return nil, errors.Wrap(err, "unable to list UTXO wallet balances")
			}

			for _, balance := range balances {
				if balance.Currency != currency.Ticker || balance.NetworkID != networkID || balance.Amount.IsZero() {
					continue
				}
				if !balance.Amount.CompatibleTo(merchantBalance.Amount) {
					continue
				}

				sources = append(sources, utxoWithdrawalSource{
					Wallet:  sourceWallet,
					Balance: balance,
				})
			}
		}

		if nextID == nil {
			break
		}
		start = *nextID
	}

	return sources, nil
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
