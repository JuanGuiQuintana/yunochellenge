package domain

import (
	"time"

	"github.com/google/uuid"
)

// Processor represents a payment processor that submits chargebacks via webhook.
type Processor struct {
	ProcessorID        uuid.UUID
	Name               string
	WebhookSecretHash  string
	RateLimitPerMinute int
	IsActive           bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
