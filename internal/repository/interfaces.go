package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/juanatsap/chargeback-api/internal/domain"
)

// ChargebackRepository defines persistence operations for the Chargeback aggregate.
type ChargebackRepository interface {
	Insert(ctx context.Context, cb domain.Chargeback) error
	FindByID(ctx context.Context, id uuid.UUID) (domain.Chargeback, error)
	FindByProcessorID(ctx context.Context, processorName, processorCBID string) (domain.Chargeback, bool, error)
	List(ctx context.Context, filter domain.ChargebackFilter) ([]domain.Chargeback, int, error)
	UpdateFlags(ctx context.Context, id uuid.UUID, flags []string) error
	Summary(ctx context.Context, merchantID *uuid.UUID) (domain.ChargebackSummary, error)
}

// MerchantRepository defines persistence operations for the Merchant aggregate.
type MerchantRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (domain.Merchant, error)
	UpdateRatioCache(ctx context.Context, merchantID uuid.UUID) error
	Stats(ctx context.Context, merchantID uuid.UUID) (domain.MerchantStats, error)
}

// CardholderRepository defines persistence operations for Cardholder identity resolution.
type CardholderRepository interface {
	FindOrCreate(ctx context.Context, fingerprint domain.CardFingerprint, bin, last4 string) (uuid.UUID, error)
}

// FxRateRepository defines lookups for currency exchange rates.
type FxRateRepository interface {
	GetRate(ctx context.Context, currency string, date time.Time) (float64, error)
}

// RateLimitRepository provides atomic rate-limit tracking per processor per minute window.
type RateLimitRepository interface {
	CheckAndIncrement(ctx context.Context, processorID uuid.UUID, window time.Time, limit int) (bool, error)
}

// ProcessorRepository defines lookups for payment processor configuration.
type ProcessorRepository interface {
	FindByName(ctx context.Context, name string) (domain.Processor, error)
}

// ReasonCodeRepository defines lookups for processor-specific reason code mappings.
type ReasonCodeRepository interface {
	FindByRawCode(ctx context.Context, processorID uuid.UUID, rawCode string) (domain.ReasonCode, error)
}
