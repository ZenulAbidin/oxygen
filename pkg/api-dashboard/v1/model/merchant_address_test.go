package model

import (
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
)

func TestMerchantAddressValidate_AllowsBTC(t *testing.T) {
	address := &MerchantAddress{
		Blockchain: "BTC",
	}

	assert.NoError(t, address.Validate(strfmt.Default))
}

func TestMerchantAddressValidate_AllowsLTC(t *testing.T) {
	address := &MerchantAddress{
		Blockchain: "LTC",
	}

	assert.NoError(t, address.Validate(strfmt.Default))
}
