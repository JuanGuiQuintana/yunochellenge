package processor

import (
	"context"
	"encoding/json"
	"time"
)

// ChargebackDTO is the normalized output that every adapter produces.
// It's a value object — immutable once constructed.
type ChargebackDTO struct {
	ProcessorChargebackID string
	TransactionID         string
	MerchantID            string          // as string — service layer resolves to UUID
	BIN                   string
	Last4                 string
	CardholderName        string
	Amount                int64           // in cents, never float
	Currency              string          // ISO 4217, uppercase
	TransactionDate       time.Time
	NotificationDate      time.Time
	DisputeDeadline       time.Time
	RawReasonCode         string
	RawPayload            json.RawMessage // original payload preserved
}

// ProcessorAdapter normalizes a raw processor webhook payload into a ChargebackDTO.
// Each implementation knows the schema JSON of its specific processor.
// Adapters must NOT touch the database or compute scores.
type ProcessorAdapter interface {
	Normalize(ctx context.Context, raw []byte) (ChargebackDTO, error)
}
