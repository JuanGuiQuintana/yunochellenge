package globalpay

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

	txnDateStr := "2024-05-20T16:45:00Z"
	notifiedDateStr := "2024-05-21T09:00:00Z"
	deadlineStr := "2024-05-28T09:00:00Z"

	txnDate, _ := time.Parse(time.RFC3339, txnDateStr)
	notifiedDate, _ := time.Parse(time.RFC3339, notifiedDateStr)
	deadline, _ := time.Parse(time.RFC3339, deadlineStr)

	validPayload := []byte(`{
		"id":   "GP-555-ABC",
		"type": "chargeback.created",
		"data": {
			"reference":   "REF-88776",
			"merchant_id": "GP-MERCH-100",
			"card_details": {
				"first_six":        "404040",
				"last_four":        "6868",
				"cardholder_name":  "Maria Garcia"
			},
			"financial": {
				"transaction_amount":   250000,
				"transaction_currency": "MXN",
				"transaction_date":     "2024-05-20T16:45:00Z"
			},
			"dispute": {
				"notification_date": "2024-05-21T09:00:00Z",
				"response_deadline": "2024-05-28T09:00:00Z",
				"reason":            "13.2"
			}
		}
	}`)

	tests := []struct {
		name    string
		payload []byte
		wantErr bool
		errIs   error
	}{
		{
			name:    "Given valid GlobalPay payload when normalizing then all DTO fields are populated correctly",
			payload: validPayload,
		},
		{
			name:    "Given invalid JSON when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{bad json`),
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
		{
			name: "Given missing top-level id when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{
				"id":   "",
				"type": "chargeback.created",
				"data": {
					"reference":   "REF-001",
					"merchant_id": "GP-MERCH-01",
					"card_details": {"first_six":"404040","last_four":"6868","cardholder_name":"Test"},
					"financial": {
						"transaction_amount":   100000,
						"transaction_currency": "MXN",
						"transaction_date":     "2024-05-20T16:45:00Z"
					},
					"dispute": {
						"notification_date": "2024-05-21T09:00:00Z",
						"response_deadline": "2024-05-28T09:00:00Z",
						"reason":            "13.2"
					}
				}
			}`),
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
		{
			name: "Given zero transaction_amount when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{
				"id":   "GP-001",
				"type": "chargeback.created",
				"data": {
					"reference":   "REF-001",
					"merchant_id": "GP-MERCH-01",
					"card_details": {"first_six":"404040","last_four":"6868","cardholder_name":"Test"},
					"financial": {
						"transaction_amount":   0,
						"transaction_currency": "MXN",
						"transaction_date":     "2024-05-20T16:45:00Z"
					},
					"dispute": {
						"notification_date": "2024-05-21T09:00:00Z",
						"response_deadline": "2024-05-28T09:00:00Z",
						"reason":            "13.2"
					}
				}
			}`),
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
		{
			name: "Given invalid transaction_date format when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{
				"id":   "GP-002",
				"type": "chargeback.created",
				"data": {
					"reference":   "REF-002",
					"merchant_id": "GP-MERCH-01",
					"card_details": {"first_six":"404040","last_four":"6868","cardholder_name":"Test"},
					"financial": {
						"transaction_amount":   100000,
						"transaction_currency": "MXN",
						"transaction_date":     "not-a-date"
					},
					"dispute": {
						"notification_date": "2024-05-21T09:00:00Z",
						"response_deadline": "2024-05-28T09:00:00Z",
						"reason":            "13.2"
					}
				}
			}`),
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
		{
			name: "Given invalid notification_date format when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{
				"id":   "GP-003",
				"type": "chargeback.created",
				"data": {
					"reference":   "REF-003",
					"merchant_id": "GP-MERCH-01",
					"card_details": {"first_six":"404040","last_four":"6868","cardholder_name":"Test"},
					"financial": {
						"transaction_amount":   100000,
						"transaction_currency": "MXN",
						"transaction_date":     "2024-05-20T16:45:00Z"
					},
					"dispute": {
						"notification_date": "bad-date",
						"response_deadline": "2024-05-28T09:00:00Z",
						"reason":            "13.2"
					}
				}
			}`),
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
		{
			name: "Given invalid response_deadline format when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{
				"id":   "GP-004",
				"type": "chargeback.created",
				"data": {
					"reference":   "REF-004",
					"merchant_id": "GP-MERCH-01",
					"card_details": {"first_six":"404040","last_four":"6868","cardholder_name":"Test"},
					"financial": {
						"transaction_amount":   100000,
						"transaction_currency": "MXN",
						"transaction_date":     "2024-05-20T16:45:00Z"
					},
					"dispute": {
						"notification_date": "2024-05-21T09:00:00Z",
						"response_deadline": "bad-date",
						"reason":            "13.2"
					}
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
			require.Equal(t, "GP-555-ABC", dto.ProcessorChargebackID)
			require.Equal(t, "REF-88776", dto.TransactionID)
			require.Equal(t, "GP-MERCH-100", dto.MerchantID)
			require.Equal(t, "404040", dto.BIN)
			require.Equal(t, "6868", dto.Last4)
			require.Equal(t, "Maria Garcia", dto.CardholderName)
			require.Equal(t, int64(250000), dto.Amount)
			require.Equal(t, "MXN", dto.Currency)
			require.Equal(t, txnDate.UTC(), dto.TransactionDate.UTC())
			require.Equal(t, notifiedDate.UTC(), dto.NotificationDate.UTC())
			require.Equal(t, deadline.UTC(), dto.DisputeDeadline.UTC())
			require.Equal(t, "13.2", dto.RawReasonCode)
			require.NotEmpty(t, dto.RawPayload)
		})
	}
}
