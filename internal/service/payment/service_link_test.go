package payment

import (
	"testing"

	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/util"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestCreateLinkPropsValidate(t *testing.T) {
	t.Run("payment link defaults currency from price", func(t *testing.T) {
		props := CreateLinkProps{
			Name:           "fixed link",
			Price:          lo.Must(money.USD.MakeAmount("25")),
			SuccessAction:  SuccessActionShowMessage,
			SuccessMessage: util.Ptr("Thank you"),
		}

		props.fillDefaults()

		assert.NoError(t, props.validate())
		assert.Equal(t, LinkTypePayment, props.Type)
		assert.Equal(t, money.USD, props.Currency)
	})

	t.Run("payment link rejects mismatched currency", func(t *testing.T) {
		props := CreateLinkProps{
			Type:           LinkTypePayment,
			Name:           "fixed link",
			Currency:       money.EUR,
			Price:          lo.Must(money.USD.MakeAmount("25")),
			SuccessAction:  SuccessActionShowMessage,
			SuccessMessage: util.Ptr("Thank you"),
		}

		assert.ErrorContains(t, props.validate(), "price currency should match link currency")
	})

	t.Run("donation link accepts currency without price", func(t *testing.T) {
		props := CreateLinkProps{
			Type:           LinkTypeDonation,
			Name:           "donation link",
			Currency:       money.USD,
			SuccessAction:  SuccessActionShowMessage,
			SuccessMessage: util.Ptr("Thank you"),
		}

		assert.NoError(t, props.validate())
	})
}

func TestLinkPaymentAmount(t *testing.T) {
	t.Run("fixed link uses configured price", func(t *testing.T) {
		price := lo.Must(money.USD.MakeAmount("25"))
		link := &Link{
			Type:  LinkTypePayment,
			Price: price,
		}

		actual, err := linkPaymentAmount(link)

		assert.NoError(t, err)
		assert.True(t, price.Equals(actual))
	})

	t.Run("fixed link rejects supplied amount", func(t *testing.T) {
		link := &Link{
			Type:  LinkTypePayment,
			Price: lo.Must(money.USD.MakeAmount("25")),
		}

		_, err := linkPaymentAmount(link, lo.Must(money.USD.MakeAmount("30")))

		assert.ErrorIs(t, err, ErrLinkValidation)
	})

	t.Run("donation link uses supplied amount", func(t *testing.T) {
		amount := lo.Must(money.USD.MakeAmount("30"))
		link := &Link{
			Type:     LinkTypeDonation,
			Currency: money.USD,
		}

		actual, err := linkPaymentAmount(link, amount)

		assert.NoError(t, err)
		assert.True(t, amount.Equals(actual))
	})

	t.Run("donation link requires amount", func(t *testing.T) {
		link := &Link{
			Type:     LinkTypeDonation,
			Currency: money.USD,
		}

		_, err := linkPaymentAmount(link)

		assert.ErrorIs(t, err, ErrLinkValidation)
	})

	t.Run("donation link rejects zero amount", func(t *testing.T) {
		link := &Link{
			Type:     LinkTypeDonation,
			Currency: money.USD,
		}

		_, err := linkPaymentAmount(link, lo.Must(money.USD.MakeAmount("0")))

		assert.ErrorIs(t, err, ErrLinkValidation)
	})

	t.Run("donation link rejects mismatched currency", func(t *testing.T) {
		link := &Link{
			Type:     LinkTypeDonation,
			Currency: money.USD,
		}

		_, err := linkPaymentAmount(link, lo.Must(money.EUR.MakeAmount("30")))

		assert.ErrorIs(t, err, ErrLinkValidation)
	})
}
