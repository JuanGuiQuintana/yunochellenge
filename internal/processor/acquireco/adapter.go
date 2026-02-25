package acquireco

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/juanatsap/chargeback-api/internal/domain"
	"github.com/juanatsap/chargeback-api/internal/processor"
)

// Adapter normalizes AcquireCo webhook payloads into ChargebackDTO values.
type Adapter struct{}

// New returns a ProcessorAdapter backed by AcquireCo's JSON schema.
func New() processor.ProcessorAdapter {
	return &Adapter{}
}

type payload struct {
	DisputeID string `json:"dispute_id"`
	TxnRef    string `json:"txn_ref"`
	StoreID   string `json:"store_id"`
	Card      struct {
		BIN    string `json:"bin"`
		Last4  string `json:"last_four"`
		Holder string `json:"holder"`
	} `json:"card"`
	Charge struct {
		AmountCents int64  `json:"amount_cents"`
		Currency    string `json:"currency"`
	} `json:"charge"`
	Dates struct {
		Transaction string `json:"transaction"`
		Notified    string `json:"notified"`
		RespondBy   string `json:"respond_by"`
	} `json:"dates"`
	Reason string `json:"reason"`
}

func (a *Adapter) Normalize(ctx context.Context, raw []byte) (processor.ChargebackDTO, error) {
	var p payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return processor.ChargebackDTO{}, fmt.Errorf("acquireco.Normalize: %w: %w", domain.ErrInvalidInput, err)
	}

	if p.DisputeID == "" || p.Charge.AmountCents <= 0 {
		return processor.ChargebackDTO{}, fmt.Errorf("acquireco.Normalize: missing required fields: %w", domain.ErrInvalidInput)
	}

	txnDate, err := time.Parse(time.RFC3339, p.Dates.Transaction)
	if err != nil {
		return processor.ChargebackDTO{}, fmt.Errorf("acquireco.Normalize transaction_date: %w", domain.ErrInvalidInput)
	}
	notifiedDate, err := time.Parse(time.RFC3339, p.Dates.Notified)
	if err != nil {
		return processor.ChargebackDTO{}, fmt.Errorf("acquireco.Normalize notified_date: %w", domain.ErrInvalidInput)
	}
	deadline, err := time.Parse(time.RFC3339, p.Dates.RespondBy)
	if err != nil {
		return processor.ChargebackDTO{}, fmt.Errorf("acquireco.Normalize deadline: %w", domain.ErrInvalidInput)
	}

	return processor.ChargebackDTO{
		ProcessorChargebackID: p.DisputeID,
		TransactionID:         p.TxnRef,
		MerchantID:            p.StoreID,
		BIN:                   p.Card.BIN,
		Last4:                 p.Card.Last4,
		CardholderName:        p.Card.Holder,
		Amount:                p.Charge.AmountCents,
		Currency:              p.Charge.Currency,
		TransactionDate:       txnDate,
		NotificationDate:      notifiedDate,
		DisputeDeadline:       deadline,
		RawReasonCode:         p.Reason,
		RawPayload:            json.RawMessage(raw),
	}, nil
}
