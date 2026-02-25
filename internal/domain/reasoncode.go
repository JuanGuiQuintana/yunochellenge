package domain

import (
	"time"

	"github.com/google/uuid"
)

// RiskLevel is an ordinal classification of how risky a chargeback reason code is.
type RiskLevel int

const (
	RiskLevelMinimal  RiskLevel = 1
	RiskLevelLow      RiskLevel = 2
	RiskLevelMedium   RiskLevel = 3
	RiskLevelHigh     RiskLevel = 4
	RiskLevelCritical RiskLevel = 5
)

// ReasonCode maps a processor-specific chargeback code to a normalized category and risk level.
type ReasonCode struct {
	ReasonCodeID   uuid.UUID
	ProcessorID    uuid.UUID
	RawCode        string
	NormalizedCode string
	Category       string
	RiskLevel      RiskLevel
	TypicalWinRate float64
	CreatedAt      time.Time
}
