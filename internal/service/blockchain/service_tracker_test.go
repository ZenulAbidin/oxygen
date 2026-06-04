package blockchain

import (
	"testing"

	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/oxygenpay/oxygen/internal/provider/trongrid"
	"github.com/oxygenpay/oxygen/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTronIncomingFromTransaction(t *testing.T) {
	currency := money.CryptoCurrency{
		Blockchain:    money.Blockchain("TRON"),
		NetworkID:     "mainnet",
		TestNetworkID: "testnet",
		Type:          money.Coin,
		Ticker:        "TRON",
		Decimals:      6,
	}

	ownerHex := "41b3dcf27c251da9363f1a4888257c16676cf54edf"
	toHex := "4199409c7014a738224159a8d3e12cc90163ce6db2"
	recipient := util.TronHexToBase58(toHex)

	tx := trongrid.AccountTransaction{
		TxID:        "tron-tx",
		BlockNumber: 123,
	}
	tx.Ret = append(tx.Ret, struct {
		ContractRet string `json:"contractRet"`
	}{ContractRet: "SUCCESS"})
	tx.RawData.Contract = append(tx.RawData.Contract, struct {
		Type      string `json:"type"`
		Parameter struct {
			Value struct {
				Amount       int64  `json:"amount"`
				OwnerAddress string `json:"owner_address"`
				ToAddress    string `json:"to_address"`
			} `json:"value"`
		} `json:"parameter"`
	}{Type: "TransferContract"})
	tx.RawData.Contract[0].Parameter.Value.Amount = 5_000_000
	tx.RawData.Contract[0].Parameter.Value.OwnerAddress = ownerHex
	tx.RawData.Contract[0].Parameter.Value.ToAddress = toHex

	incoming, ok, err := tronIncomingFromTransaction(tx, recipient, currency, false)

	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "tron-tx", incoming.TransactionID)
	assert.Equal(t, "5000000", incoming.Amount.StringRaw())
	assert.Equal(t, util.TronHexToBase58(ownerHex), incoming.SenderAddress)
	assert.Equal(t, "mainnet", incoming.NetworkID)
	assert.Equal(t, int64(123), incoming.BlockNumber)
}

func TestTronIncomingFromTransactionSkipsActivationTopUp(t *testing.T) {
	currency := money.CryptoCurrency{
		Blockchain:    money.Blockchain("TRON"),
		NetworkID:     "mainnet",
		TestNetworkID: "testnet",
		Type:          money.Coin,
		Ticker:        "TRON",
		Decimals:      6,
	}

	toHex := "4199409c7014a738224159a8d3e12cc90163ce6db2"
	recipient := util.TronHexToBase58(toHex)

	tx := trongrid.AccountTransaction{TxID: "activation"}
	tx.RawData.Contract = append(tx.RawData.Contract, struct {
		Type      string `json:"type"`
		Parameter struct {
			Value struct {
				Amount       int64  `json:"amount"`
				OwnerAddress string `json:"owner_address"`
				ToAddress    string `json:"to_address"`
			} `json:"value"`
		} `json:"parameter"`
	}{Type: "TransferContract"})
	tx.RawData.Contract[0].Parameter.Value.Amount = 1_000_000
	tx.RawData.Contract[0].Parameter.Value.ToAddress = toHex

	_, ok, err := tronIncomingFromTransaction(tx, recipient, currency, false)

	require.NoError(t, err)
	assert.False(t, ok)
}
