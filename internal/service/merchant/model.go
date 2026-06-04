package merchant

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/oxygenpay/oxygen/internal/kms/wallet"
	"github.com/pkg/errors"
)

type Merchant struct {
	ID        int64
	UUID      uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string
	Website   string
	CreatorID int64
	settings  Settings
}

const (
	PropertyWebhookURL      = "webhook.url"
	PropertySignatureSecret = "webhook.secret"
	PropertyPaymentMethods  = "payment.methods"

	PropertyDefaultPaymentExpirationMinutes = "payment.default_expiration_minutes"
	DefaultPaymentExpirationMinutes         = int64(20)
	MinPaymentExpirationMinutes             = int64(1)
	MaxPaymentExpirationMinutes             = int64(1440)
)

func (m *Merchant) Settings() Settings {
	return m.settings
}

type Property string
type Settings map[Property]string

func (s Settings) WebhookURL() string {
	return s[PropertyWebhookURL]
}

func (s Settings) WebhookSignatureSecret() string {
	return s[PropertySignatureSecret]
}

func (s Settings) PaymentMethods() []string {
	raw := s[PropertyPaymentMethods]
	if raw == "" {
		return nil
	}

	return strings.Split(raw, ",")
}

func (s Settings) DefaultPaymentExpirationMinutes() int64 {
	raw := s[PropertyDefaultPaymentExpirationMinutes]
	if raw == "" {
		return DefaultPaymentExpirationMinutes
	}

	minutes, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || ValidatePaymentExpirationMinutes(minutes) != nil {
		return DefaultPaymentExpirationMinutes
	}

	return minutes
}

func (s Settings) PaymentExpirationPeriod() time.Duration {
	return time.Duration(s.DefaultPaymentExpirationMinutes()) * time.Minute
}

func ValidatePaymentExpirationMinutes(minutes int64) error {
	if minutes < MinPaymentExpirationMinutes || minutes > MaxPaymentExpirationMinutes {
		return errors.Errorf(
			"default expiration must be between %d and %d minutes",
			MinPaymentExpirationMinutes,
			MaxPaymentExpirationMinutes,
		)
	}

	return nil
}

func (s Settings) toJSONB() pgtype.JSONB {
	if len(s) == 0 {
		return pgtype.JSONB{Status: pgtype.Null}
	}

	raw, _ := json.Marshal(s)

	return pgtype.JSONB{Bytes: raw, Status: pgtype.Present}
}

type Address struct {
	ID             int64
	UUID           uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Name           string
	MerchantID     int64
	Blockchain     wallet.Blockchain
	BlockchainName string
	Address        string
}
