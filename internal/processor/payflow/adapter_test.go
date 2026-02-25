package payflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/juanatsap/chargeback-api/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestAdapter_Normalize(t *testing.T) {
	ctx := context.Background()

	txnDateStr := "2024-03-05T14:30:00Z"
	notifiedDateStr := "2024-03-06T08:00:00Z"
	deadlineStr := "2024-03-13T08:00:00Z"

	txnDate, _ := time.Parse(time.RFC3339, txnDateStr)
	notifiedDate, _ := time.Parse(time.RFC3339, notifiedDateStr)
	deadline, _ := time.Parse(time.RFC3339, deadlineStr)

	validPayload := []byte(`{
		"chargeback_ref":   "PF-99887",
		"original_txn":     "ORIG-TXN-001",
		"merchant_number":  "MERCH-7890",
		"cardholder": {
			"card_bin":    "522222",
			"card_ending": "5555",
			"name":        "Carlos Mendez"
		},
		"dispute": {
			"amount":        150000,
			"currency_code": "BRL",
			"txn_datetime":  "2024-03-05T14:30:00Z",
			"received_at":   "2024-03-06T08:00:00Z",
			"deadline":      "2024-03-13T08:00:00Z",
			"reason_code":   "13.1"
		}
	}`)

	tests := []struct {
		name    string
		payload []byte
		wantErr bool
		errIs   error
	}{
		{
			name:    "Given valid PayFlow payload when normalizing then all DTO fields are populated correctly",
			payload: validPayload,
		},
		{
			name:    "Given invalid JSON when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{broken json`),
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
		{
			name: "Given missing chargeback_ref when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{
				"chargeback_ref":  "",
				"original_txn":    "ORIG-001",
				"merchant_number": "MERCH-01",
				"cardholder": {"card_bin":"522222","card_ending":"5555","name":"Test"},
				"dispute": {
					"amount":       50000,
					"currency_code":"BRL",
					"txn_datetime": "2024-03-05T14:30:00Z",
					"received_at":  "2024-03-06T08:00:00Z",
					"deadline":     "2024-03-13T08:00:00Z",
					"reason_code":  "13.1"
				}
			}`),
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
		{
			name: "Given zero dispute amount when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{
				"chargeback_ref":  "PF-001",
				"original_txn":    "ORIG-001",
				"merchant_number": "MERCH-01",
				"cardholder": {"card_bin":"522222","card_ending":"5555","name":"Test"},
				"dispute": {
					"amount":       0,
					"currency_code":"BRL",
					"txn_datetime": "2024-03-05T14:30:00Z",
					"received_at":  "2024-03-06T08:00:00Z",
					"deadline":     "2024-03-13T08:00:00Z",
					"reason_code":  "13.1"
				}
			}`),
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
		{
			name: "Given invalid txn_datetime format when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{
				"chargeback_ref":  "PF-002",
				"original_txn":    "ORIG-001",
				"merchant_number": "MERCH-01",
				"cardholder": {"card_bin":"522222","card_ending":"5555","name":"Test"},
				"dispute": {
					"amount":       50000,
					"currency_code":"BRL",
					"txn_datetime": "not-a-date",
					"received_at":  "2024-03-06T08:00:00Z",
					"deadline":     "2024-03-13T08:00:00Z",
					"reason_code":  "13.1"
				}
			}`),
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
		{
			name: "Given invalid received_at format when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{
				"chargeback_ref":  "PF-003",
				"original_txn":    "ORIG-001",
				"merchant_number": "MERCH-01",
				"cardholder": {"card_bin":"522222","card_ending":"5555","name":"Test"},
				"dispute": {
					"amount":       50000,
					"currency_code":"BRL",
					"txn_datetime": "2024-03-05T14:30:00Z",
					"received_at":  "bad-date",
					"deadline":     "2024-03-13T08:00:00Z",
					"reason_code":  "13.1"
				}
			}`),
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
		{
			name: "Given invalid deadline format when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{
				"chargeback_ref":  "PF-004",
				"original_txn":    "ORIG-001",
				"merchant_number": "MERCH-01",
				"cardholder": {"card_bin":"522222","card_ending":"5555","name":"Test"},
				"dispute": {
					"amount":       50000,
					"currency_code":"BRL",
					"txn_datetime": "2024-03-05T14:30:00Z",
					"received_at":  "2024-03-06T08:00:00Z",
					"deadline":     "bad-date",
					"reason_code":  "13.1"
				}
			}`),
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
	}

	adapter := New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dto, err := adapter.Normalize(ctx, tt.payload)

			if tt.wantErr {
				require.Error(t, err)
				require.True(t, errors.Is(err, tt.errIs), "expected error to wrap %v, got: %v", tt.errIs, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, "PF-99887", dto.ProcessorChargebackID)
			require.Equal(t, "ORIG-TXN-001", dto.TransactionID)
			require.Equal(t, "MERCH-7890", dto.MerchantID)
			require.Equal(t, "522222", dto.BIN)
			require.Equal(t, "5555", dto.Last4)
			require.Equal(t, "Carlos Mendez", dto.CardholderName)
			require.Equal(t, int64(150000), dto.Amount)
			require.Equal(t, "BRL", dto.Currency)
			require.Equal(t, txnDate.UTC(), dto.TransactionDate.UTC())
			require.Equal(t, notifiedDate.UTC(), dto.NotificationDate.UTC())
			require.Equal(t, deadline.UTC(), dto.DisputeDeadline.UTC())
			require.Equal(t, "13.1", dto.RawReasonCode)
			require.NotEmpty(t, dto.RawPayload)
		})
	}
}
