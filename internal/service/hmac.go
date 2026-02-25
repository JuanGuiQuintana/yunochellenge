package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// HMACValidator validates webhook signatures using HMAC-SHA256.
// Uses constant-time comparison to prevent timing attacks.
type HMACValidator struct{}

// Validate returns true if the signature matches HMAC-SHA256(secret, body).
// signature is expected as a hex-encoded string.
func (v *HMACValidator) Validate(secret []byte, signature string, body []byte) bool {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expected := mac.Sum(nil)

	got, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	return hmac.Equal(expected, got)
}
