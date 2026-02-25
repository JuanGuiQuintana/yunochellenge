package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CardFingerprint is a value object: SHA-256 of "BIN|LAST4|NORMALIZED_NAME".
// The cardholder name is never stored in plaintext.
type CardFingerprint string

// Compute creates a deterministic fingerprint from card data.
// Name is uppercased and trimmed before hashing.
func Compute(bin, last4, name string) CardFingerprint {
	normalized := strings.ToUpper(strings.TrimSpace(name))
	raw := bin + "|" + last4 + "|" + normalized
	hash := sha256.Sum256([]byte(raw))
	return CardFingerprint(hex.EncodeToString(hash[:]))
}

// Cardholder identifies a card used in a disputed transaction without storing PII.
type Cardholder struct {
	CardholderID    uuid.UUID
	BIN             string
	Last4           string
	CardFingerprint CardFingerprint
	CreatedAt       time.Time
}
