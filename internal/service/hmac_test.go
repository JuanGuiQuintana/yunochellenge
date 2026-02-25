package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

// computeHMAC is a test helper that produces the correct HMAC-SHA256 hex signature.
func computeHMAC(secret, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestHMACValidator_Validate(t *testing.T) {
	validator := &HMACValidator{}

	secret := []byte("super-secret-key")
	body := []byte(`{"dispute_id":"ACQ-001","amount_cents":89900}`)
	validSignature := computeHMAC(secret, body)

	tests := []struct {
		name      string
		secret    []byte
		signature string
		body      []byte
		want      bool
	}{
		{
			name:      "Given valid signature when validating then returns true",
			secret:    secret,
			signature: validSignature,
			body:      body,
			want:      true,
		},
		{
			name:      "Given tampered body when validating then returns false",
			secret:    secret,
			signature: validSignature,
			body:      []byte(`{"dispute_id":"ACQ-001","amount_cents":99999}`),
			want:      false,
		},
		{
			name:      "Given wrong secret when validating then returns false",
			secret:    []byte("wrong-secret"),
			signature: validSignature,
			body:      body,
			want:      false,
		},
		{
			name:      "Given non-hex signature when validating then returns false",
			secret:    secret,
			signature: "this-is-not-hex!",
			body:      body,
			want:      false,
		},
		{
			name:      "Given empty signature when validating then returns false",
			secret:    secret,
			signature: "",
			body:      body,
			want:      false,
		},
		{
			name:      "Given signature with wrong length when validating then returns false",
			secret:    secret,
			signature: "deadbeef",
			body:      body,
			want:      false,
		},
		{
			name:      "Given valid signature for empty body when validating then returns true",
			secret:    secret,
			signature: computeHMAC(secret, []byte{}),
			body:      []byte{},
			want:      true,
		},
		{
			name:      "Given all-zeros signature when validating then returns false",
			secret:    secret,
			signature: "0000000000000000000000000000000000000000000000000000000000000000",
			body:      body,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.Validate(tt.secret, tt.signature, tt.body)

			require.Equal(t, tt.want, got)
		})
	}
}
