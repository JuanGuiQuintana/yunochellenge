package payflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/juanatsap/chargeback-api/internal/domain"
	"github.com/juanatsap/chargeback-api/internal/processor"
)

// Adapter normalizes PayFlow webhook payloads into ChargebackDTO values.
type Adapter struct{}

// New returns a ProcessorAdapter backed by PayFlow's JSON schema.
func New() processor.ProcessorAdapter {
	return &Adapter{}
}

type payload struct {
	ChargebackRef  string `json:"chargeback_ref"`
	OriginalTxn    string `json:"original_txn"`
	MerchantNumber string `json:"merchant_number"`
	Cardholder     struct {
		CardBIN    string `json:"card_bin"`
		CardEnding string `json:"card_ending"`
		Name       string `json:"name"`
	} `json:"cardholder"`
	Dispute struct {
		Amount       int64  `json:"amount"`
		CurrencyCode string `json:"currency_code"`
		TxnDatetime  string `json:"txn_datetime"`
		ReceivedAt   string `json:"received_at"`
		Deadline     string `json:"deadline"`
		ReasonCode   string `json:"reason_code"`
	} `json:"dispute"`
}

func (a *Adapter) Normalize(ctx context.Context, raw []byte) (processor.ChargebackDTO, error) {
	var p payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return processor.ChargebackDTO{}, fmt.Errorf("payflow.Normalize: %w: %w", domain.ErrInvalidInput, err)
	}

	if p.ChargebackRef == "" || p.Dispute.Amount <= 0 {
		return processor.ChargebackDTO{}, fmt.Errorf("payflow.Normalize: missing required fields: %w", domain.ErrInvalidInput)
	}

	txnDate, err := time.Parse(time.RFC3339, p.Dispute.TxnDatetime)
	if err != nil {
		return processor.ChargebackDTO{}, fmt.Errorf("payflow.Normalize txn_datetime: %w", domain.ErrInvalidInput)
	}
	notifiedDate, err := time.Parse(time.RFC3339, p.Dispute.ReceivedAt)
	if err != nil {
		return processor.ChargebackDTO{}, fmt.Errorf("payflow.Normalize received_at: %w", domain.ErrInvalidInput)
	}
	deadline, err := time.Parse(time.RFC3339, p.Dispute.Deadline)
	if err != nil {
		return processor.ChargebackDTO{}, fmt.Errorf("payflow.Normalize deadline: %w", domain.ErrInvalidInput)
	}

	return processor.ChargebackDTO{
		ProcessorChargebackID: p.ChargebackRef,
		TransactionID:         p.OriginalTxn,
		MerchantID:            p.MerchantNumber,
		BIN:                   p.Cardholder.CardBIN,
		Last4:                 p.Cardholder.CardEnding,
		CardholderName:        p.Cardholder.Name,
		Amount:                p.Dispute.Amount,
		Currency:              p.Dispute.CurrencyCode,
		TransactionDate:       txnDate,
		NotificationDate:      notifiedDate,
		DisputeDeadline:       deadline,
		RawReasonCode:         p.Dispute.ReasonCode,
		RawPayload:            json.RawMessage(raw),
	}, nil
}
