package tatum

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oxygenpay/oxygen/internal/money"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderSubscribeToWebhookSkipsPlaceholderCredentials(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()

	logger := zerolog.Nop()
	provider := New(Config{
		BasePath:   server.URL,
		APIKey:     "<tatum-api-key>",
		TestAPIKey: "<tatum-test-api-key>",
	}, nil, &logger)

	id, err := provider.SubscribeToWebhook(context.Background(), SubscriptionParams{
		Blockchain: money.Blockchain("BTC"),
		Address:    "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq",
		WebhookURL: "https://example.test/api/webhook/v1/tatum/mainnet/wallet-id",
	})

	require.NoError(t, err)
	assert.Equal(t, "disabled-btc-main-bc1qar0s", id)
	assert.LessOrEqual(t, len(id), 32)
	assert.False(t, called)
}

func TestProviderSubscribeToWebhookSkipsLocalWebhookURL(t *testing.T) {
	logger := zerolog.Nop()
	provider := New(Config{
		APIKey:     "real-main-key",
		TestAPIKey: "real-test-key",
	}, nil, &logger)

	id, err := provider.SubscribeToWebhook(context.Background(), SubscriptionParams{
		Blockchain: money.Blockchain("ETH"),
		Address:    "0xc2132d05d31c914a87c6611c10748aeb04b58e8f",
		IsTest:     true,
		WebhookURL: "http://127.0.0.1:8080/api/webhook/v1/tatum/5/wallet-id",
	})

	require.NoError(t, err)
	assert.Equal(t, "disabled-eth-test-0xc2132d", id)
	assert.LessOrEqual(t, len(id), 32)
}

func TestProviderHasAPIKeyRejectsDockerExampleValues(t *testing.T) {
	logger := zerolog.Nop()
	provider := New(Config{
		APIKey:     "<tatum-api-key>",
		TestAPIKey: "<tatum-test-api-key>",
	}, nil, &logger)

	assert.False(t, provider.HasMainAPIKey())
	assert.False(t, provider.HasTestAPIKey())
}
