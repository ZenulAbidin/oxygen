package kms

import (
	"testing"

	"github.com/oxygenpay/oxygen/internal/config"
	httpServer "github.com/oxygenpay/oxygen/internal/server/http"
)

func TestServerConfigForEmbeddedKMSUsesLoopback(t *testing.T) {
	app := &App{
		config: &config.Config{
			KMS: config.KMS{
				IsEmbedded: true,
				Server: httpServer.Config{
					Address: "0.0.0.0",
					Port:    "80",
				},
			},
		},
	}

	cfg := app.serverConfig()

	if cfg.Address != embeddedKMSAddress {
		t.Fatalf("expected embedded KMS address %q, got %q", embeddedKMSAddress, cfg.Address)
	}
	if cfg.Port != embeddedKMSPort {
		t.Fatalf("expected embedded KMS port %q, got %q", embeddedKMSPort, cfg.Port)
	}
}

func TestServerConfigForStandaloneKMSKeepsConfiguredAddress(t *testing.T) {
	app := &App{
		config: &config.Config{
			KMS: config.KMS{
				Server: httpServer.Config{
					Address: "0.0.0.0",
					Port:    "8080",
				},
			},
		},
	}

	cfg := app.serverConfig()

	if cfg.Address != "0.0.0.0" {
		t.Fatalf("expected standalone KMS address %q, got %q", "0.0.0.0", cfg.Address)
	}
	if cfg.Port != "8080" {
		t.Fatalf("expected standalone KMS port %q, got %q", "8080", cfg.Port)
	}
}
