package middleware

import (
	"os"
	"testing"

	"github.com/ilyakaznacheev/cleanenv"
)

func TestCookieSecureDefaultsToTrue(t *testing.T) {
	unsetEnv(t, "SESSION_COOKIE_SECURE")
	unsetEnv(t, "CSRF_COOKIE_SECURE")

	cfg := struct {
		Session SessionConfig `yaml:"session"`
		CSRF    CSRFConfig    `yaml:"csrf"`
	}{}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		t.Fatalf("read config from environment: %v", err)
	}

	if !cfg.Session.CookieSecure {
		t.Fatal("session cookie secure should default to true")
	}

	if !cfg.CSRF.CookieSecure {
		t.Fatal("CSRF cookie secure should default to true")
	}
}

func TestCookieSecureCanBeDisabledByEnvironment(t *testing.T) {
	t.Setenv("SESSION_COOKIE_SECURE", "false")
	t.Setenv("CSRF_COOKIE_SECURE", "false")

	cfg := struct {
		Session SessionConfig `yaml:"session"`
		CSRF    CSRFConfig    `yaml:"csrf"`
	}{}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		t.Fatalf("read config from environment: %v", err)
	}

	if cfg.Session.CookieSecure {
		t.Fatal("session cookie secure should allow explicit false override")
	}

	if cfg.CSRF.CookieSecure {
		t.Fatal("CSRF cookie secure should allow explicit false override")
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()

	value, ok := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}

	t.Cleanup(func() {
		if ok {
			if err := os.Setenv(key, value); err != nil {
				t.Fatalf("restore %s: %v", key, err)
			}
		}
	})
}
