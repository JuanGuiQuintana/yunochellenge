package acquireco

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

	// Valid dates used consistently across test cases.
	txnDateStr := "2024-01-10T09:00:00Z"
	notifiedDateStr := "2024-01-15T10:00:00Z"
	deadlineStr := "2024-01-22T10:00:00Z"

	txnDate, _ := time.Parse(time.RFC3339, txnDateStr)
	notifiedDate, _ := time.Parse(time.RFC3339, notifiedDateStr)
	deadline, _ := time.Parse(time.RFC3339, deadlineStr)

	validPayload := []byte(`{
		"dispute_id": "ACQ-001",
		"txn_ref":    "TXN-XYZ",
		"store_id":   "STORE-42",
		"card": {
			"bin":        "411111",
			"last_four":  "4242",
			"holder":     "Jane Smith"
		},
		"charge": {
			"amount_cents": 89900,
			"currency":     "USD"
		},
		"dates": {
			"transaction": "2024-01-10T09:00:00Z",
			"notified":    "2024-01-15T10:00:00Z",
			"respond_by":  "2024-01-22T10:00:00Z"
		},
		"reason": "10.4"
	}`)

	tests := []struct {
		name    string
		payload []byte
		wantDTO func(t *testing.T, dto interface{})
		wantErr bool
		errIs   error
	}{
		{
			name:    "Given valid AcquireCo payload when normalizing then all DTO fields are populated correctly",
			payload: validPayload,
			wantDTO: func(t *testing.T, dtoIface interface{}) {
				// Cast checked separately below.
			},
		},
		{
			name:    "Given invalid JSON when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{not valid json`),
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
		{
			name: "Given missing dispute_id when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{
				"dispute_id": "",
				"txn_ref":    "TXN-XYZ",
				"store_id":   "STORE-42",
				"card": {"bin":"411111","last_four":"4242","holder":"Jane"},
				"charge": {"amount_cents":10000,"currency":"USD"},
				"dates": {
					"transaction": "2024-01-10T09:00:00Z",
					"notified":    "2024-01-15T10:00:00Z",
					"respond_by":  "2024-01-22T10:00:00Z"
				},
				"reason": "10.4"
			}`),
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
		{
			name: "Given zero amount_cents when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{
				"dispute_id": "ACQ-002",
				"txn_ref":    "TXN-XYZ",
				"store_id":   "STORE-42",
				"card": {"bin":"411111","last_four":"4242","holder":"Jane"},
				"charge": {"amount_cents":0,"currency":"USD"},
				"dates": {
					"transaction": "2024-01-10T09:00:00Z",
					"notified":    "2024-01-15T10:00:00Z",
					"respond_by":  "2024-01-22T10:00:00Z"
				},
				"reason": "10.4"
			}`),
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
		{
			name: "Given invalid transaction date format when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{
				"dispute_id": "ACQ-003",
				"txn_ref":    "TXN-XYZ",
				"store_id":   "STORE-42",
				"card": {"bin":"411111","last_four":"4242","holder":"Jane"},
				"charge": {"amount_cents":10000,"currency":"USD"},
				"dates": {
					"transaction": "not-a-date",
					"notified":    "2024-01-15T10:00:00Z",
					"respond_by":  "2024-01-22T10:00:00Z"
				},
				"reason": "10.4"
			}`),
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
		{
			name: "Given invalid notified date format when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{
				"dispute_id": "ACQ-004",
				"txn_ref":    "TXN-XYZ",
				"store_id":   "STORE-42",
				"card": {"bin":"411111","last_four":"4242","holder":"Jane"},
				"charge": {"amount_cents":10000,"currency":"USD"},
				"dates": {
					"transaction": "2024-01-10T09:00:00Z",
					"notified":    "bad-date",
					"respond_by":  "2024-01-22T10:00:00Z"
				},
				"reason": "10.4"
			}`),
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
		{
			name: "Given invalid deadline date format when normalizing then error wraps ErrInvalidInput",
			payload: []byte(`{
				"dispute_id": "ACQ-005",
				"txn_ref":    "TXN-XYZ",
				"store_id":   "STORE-42",
				"card": {"bin":"411111","last_four":"4242","holder":"Jane"},
				"charge": {"amount_cents":10000,"currency":"USD"},
				"dates": {
					"transaction": "2024-01-10T09:00:00Z",
					"notified":    "2024-01-15T10:00:00Z",
					"respond_by":  "bad-date"
				},
				"reason": "10.4"
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
			require.Equal(t, "ACQ-001", dto.ProcessorChargebackID)
			require.Equal(t, "TXN-XYZ", dto.TransactionID)
			require.Equal(t, "STORE-42", dto.MerchantID)
			require.Equal(t, "411111", dto.BIN)
			require.Equal(t, "4242", dto.Last4)
			require.Equal(t, "Jane Smith", dto.CardholderName)
			require.Equal(t, int64(89900), dto.Amount)
			require.Equal(t, "USD", dto.Currency)
			require.Equal(t, txnDate.UTC(), dto.TransactionDate.UTC())
			require.Equal(t, notifiedDate.UTC(), dto.NotificationDate.UTC())
			require.Equal(t, deadline.UTC(), dto.DisputeDeadline.UTC())
			require.Equal(t, "10.4", dto.RawReasonCode)
			require.NotEmpty(t, dto.RawPayload)
		})
	}

	// Suppress unused variable warnings for pre-parsed time values.
	_ = txnDate
	_ = notifiedDate
	_ = deadline
}
