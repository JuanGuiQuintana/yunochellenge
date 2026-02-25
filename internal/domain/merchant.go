package domain

import (
	"time"

	"github.com/google/uuid"
)

// Merchant represents a payment processing merchant subject to chargeback monitoring.
type Merchant struct {
	MerchantID             uuid.UUID
	Name                   string
	Email                  string
	CurrentChargebackRatio float64
	TotalTransactions30d   int
	TotalChargebacks30d    int
	IsFlaggedByProcessor   bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// MerchantStats extends Merchant with trend and classification data for the stats endpoint.
type MerchantStats struct {
	MerchantID             uuid.UUID
	Name                   string
	Email                  string
	CurrentChargebackRatio float64
	TotalTransactions30d   int
	TotalChargebacks30d    int
	IsFlaggedByProcessor   bool
	RatioTrend             []RatioSnapshot
	RiskLevel              string // "healthy", "yellow", "alert", "monitoring", "critical", "termination"
}

// RatioSnapshot captures a merchant's chargeback ratio at a point in time.
type RatioSnapshot struct {
	Date             time.Time
	ChargebackCount  int
	TransactionCount int
	Ratio            float64
}
