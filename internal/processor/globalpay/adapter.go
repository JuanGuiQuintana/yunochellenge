package globalpay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/juanatsap/chargeback-api/internal/domain"
	"github.com/juanatsap/chargeback-api/internal/processor"
)

// Adapter normalizes GlobalPay webhook payloads into ChargebackDTO values.
type Adapter struct{}

// New returns a ProcessorAdapter backed by GlobalPay's JSON schema.
func New() processor.ProcessorAdapter {
	return &Adapter{}
}

type payload struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Reference  string `json:"reference"`
		MerchantID string `json:"merchant_id"`
		CardDetails struct {
			FirstSix       string `json:"first_six"`
			LastFour       string `json:"last_four"`
			CardholderName string `json:"cardholder_name"`
		} `json:"card_details"`
		Financial struct {
			TransactionAmount   int64  `json:"transaction_amount"`
			TransactionCurrency string `json:"transaction_currency"`
			TransactionDate     string `json:"transaction_date"`
		} `json:"financial"`
		Dispute struct {
			NotificationDate string `json:"notification_date"`
			ResponseDeadline string `json:"response_deadline"`
			Reason           string `json:"reason"`
		} `json:"dispute"`
	} `json:"data"`
}

func (a *Adapter) Normalize(ctx context.Context, raw []byte) (processor.ChargebackDTO, error) {
	var p payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return processor.ChargebackDTO{}, fmt.Errorf("globalpay.Normalize: %w: %w", domain.ErrInvalidInput, err)
	}

	if p.ID == "" || p.Data.Financial.TransactionAmount <= 0 {
		return processor.ChargebackDTO{}, fmt.Errorf("globalpay.Normalize: missing required fields: %w", domain.ErrInvalidInput)
	}

	txnDate, err := time.Parse(time.RFC3339, p.Data.Financial.TransactionDate)
	if err != nil {
		return processor.ChargebackDTO{}, fmt.Errorf("globalpay.Normalize transaction_date: %w", domain.ErrInvalidInput)
	}
	notifiedDate, err := time.Parse(time.RFC3339, p.Data.Dispute.NotificationDate)
	if err != nil {
		return processor.ChargebackDTO{}, fmt.Errorf("globalpay.Normalize notification_date: %w", domain.ErrInvalidInput)
	}
	deadline, err := time.Parse(time.RFC3339, p.Data.Dispute.ResponseDeadline)
	if err != nil {
		return processor.ChargebackDTO{}, fmt.Errorf("globalpay.Normalize deadline: %w", domain.ErrInvalidInput)
	}

	return processor.ChargebackDTO{
		ProcessorChargebackID: p.ID,
		TransactionID:         p.Data.Reference,
		MerchantID:            p.Data.MerchantID,
		BIN:                   p.Data.CardDetails.FirstSix,
		Last4:                 p.Data.CardDetails.LastFour,
		CardholderName:        p.Data.CardDetails.CardholderName,
		Amount:                p.Data.Financial.TransactionAmount,
		Currency:              p.Data.Financial.TransactionCurrency,
		TransactionDate:       txnDate,
		NotificationDate:      notifiedDate,
		DisputeDeadline:       deadline,
		RawReasonCode:         p.Data.Dispute.Reason,
		RawPayload:            json.RawMessage(raw),
	}, nil
}
