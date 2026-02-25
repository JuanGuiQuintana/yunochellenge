package domain

import (
	"time"

	"github.com/google/uuid"
)

// SortField enumerates valid sort columns for chargeback queries.
type SortField string

const (
	SortByScore            SortField = "risk_score"
	SortByDisputeDeadline  SortField = "dispute_deadline"
	SortByAmount           SortField = "amount"
	SortByNotificationDate SortField = "notification_date"
)

// SortOrder controls ascending or descending sort direction.
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// FlagsMatch controls whether flag filtering requires any or all flags to match.
type FlagsMatch string

const (
	FlagsMatchAny FlagsMatch = "any"
	FlagsMatchAll FlagsMatch = "all"
)

// ChargebackFilter encapsulates all query parameters for listing chargebacks.
// Pointer fields are optional — nil means "no filter on this field".
type ChargebackFilter struct {
	MerchantID     *uuid.UUID
	ScoreMin       *int
	DeadlineHours  *int       // chargebacks expiring within N hours
	DeadlineBefore *time.Time
	Flags          []string   // filter by flags (OR or AND based on FlagsMatch)
	FlagsMatch     FlagsMatch // "any" (default) or "all"
	ReasonCode     *string    // normalized_code
	ProcessorName  *string
	Status         *ChargebackStatus
	Currency       *string
	AmountMin      *int64 // in cents
	AmountMax      *int64 // in cents
	SortBy         SortField
	SortOrder      SortOrder
	Page           int // 1-indexed, default 1
	PerPage        int // default 25, max 100
}

// Normalize applies defaults and clamps PerPage to the allowed range.
func (f *ChargebackFilter) Normalize() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 {
		f.PerPage = 25
	}
	if f.PerPage > 100 {
		f.PerPage = 100
	}
	if f.SortBy == "" {
		f.SortBy = SortByDisputeDeadline
	}
	if f.SortOrder == "" {
		f.SortOrder = SortAsc
	}
	if f.FlagsMatch == "" {
		f.FlagsMatch = FlagsMatchAny
	}
}

// Offset returns the SQL OFFSET value derived from the current page and page size.
func (f ChargebackFilter) Offset() int {
	return (f.Page - 1) * f.PerPage
}
