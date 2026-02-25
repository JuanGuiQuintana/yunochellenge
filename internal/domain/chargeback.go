package domain

import (
	"time"

	"github.com/google/uuid"
)

// ChargebackStatus represents the lifecycle state of a chargeback dispute.
type ChargebackStatus string

const (
	StatusOpen        ChargebackStatus = "open"
	StatusUnderReview ChargebackStatus = "under_review"
	StatusWon         ChargebackStatus = "won"
	StatusLost        ChargebackStatus = "lost"
	StatusExpired     ChargebackStatus = "expired"
)

// Flag identifies a risk signal attached to a chargeback.
type Flag string

const (
	FlagRepeatOffender             Flag = "repeat_offender"
	FlagMerchantHotZone            Flag = "merchant_hot_zone"
	FlagSuspiciousReasonClustering Flag = "suspicious_reason_clustering"
	FlagFXPending                  Flag = "fx_pending"
)

// Chargeback is the central aggregate of the domain.
type Chargeback struct {
	ChargebackID          uuid.UUID
	ProcessorChargebackID string
	ProcessorID           uuid.UUID
	MerchantID            uuid.UUID
	TransactionID         string
	CardholderID          uuid.UUID
	ReasonCodeID          uuid.UUID
	Amount                int64    // in cents
	Currency              string
	AmountUSD             *float64 // nil if FX unavailable
	TransactionDate       time.Time
	NotificationDate      time.Time
	DisputeDeadline       time.Time
	Status                ChargebackStatus
	RawPayload            []byte // JSONB raw
	RiskScore             int
	ScoreBreakdown        ScoreBreakdown
	Flags                 []string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ChargebackSummary aggregates counts and metrics for the /summary endpoint.
type ChargebackSummary struct {
	TotalOpen      int
	TotalResponded int
	TotalWon       int
	TotalLost      int
	TotalExpired   int
	AvgRiskScore   float64
	ExpiringIn24h  int
	ExpiringIn48h  int
	ExpiringIn72h  int
}
