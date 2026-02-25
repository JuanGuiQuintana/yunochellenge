package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCardFingerprint_Compute(t *testing.T) {
	t.Run("Given same inputs when computing twice then fingerprints are identical", func(t *testing.T) {
		fp1 := Compute("411111", "1234", "John Doe")
		fp2 := Compute("411111", "1234", "John Doe")

		require.Equal(t, fp1, fp2)
		require.NotEmpty(t, fp1)
	})

	t.Run("Given uppercase name when computing then matches lowercase version", func(t *testing.T) {
		fpUpper := Compute("411111", "1234", "JOHN DOE")
		fpLower := Compute("411111", "1234", "john doe")

		require.Equal(t, fpUpper, fpLower)
	})

	t.Run("Given mixed-case name when computing then matches uppercase version", func(t *testing.T) {
		fpMixed := Compute("411111", "1234", "John Doe")
		fpUpper := Compute("411111", "1234", "JOHN DOE")

		require.Equal(t, fpMixed, fpUpper)
	})

	t.Run("Given name with leading whitespace when computing then matches trimmed version", func(t *testing.T) {
		fpPadded := Compute("411111", "1234", "  John Doe  ")
		fpClean := Compute("411111", "1234", "John Doe")

		require.Equal(t, fpPadded, fpClean)
	})

	t.Run("Given name with trailing whitespace when computing then matches trimmed version", func(t *testing.T) {
		fpTrailing := Compute("411111", "1234", "John Doe   ")
		fpClean := Compute("411111", "1234", "John Doe")

		require.Equal(t, fpTrailing, fpClean)
	})

	t.Run("Given different BIN when computing then fingerprints differ", func(t *testing.T) {
		fp1 := Compute("411111", "1234", "John Doe")
		fp2 := Compute("522222", "1234", "John Doe")

		require.NotEqual(t, fp1, fp2)
	})

	t.Run("Given different last4 when computing then fingerprints differ", func(t *testing.T) {
		fp1 := Compute("411111", "1234", "John Doe")
		fp2 := Compute("411111", "9999", "John Doe")

		require.NotEqual(t, fp1, fp2)
	})

	t.Run("Given different cardholder name when computing then fingerprints differ", func(t *testing.T) {
		fp1 := Compute("411111", "1234", "John Doe")
		fp2 := Compute("411111", "1234", "Jane Smith")

		require.NotEqual(t, fp1, fp2)
	})

	t.Run("Given valid inputs when computing then fingerprint is 64 hex characters", func(t *testing.T) {
		fp := Compute("411111", "1234", "John Doe")

		// SHA-256 produces 32 bytes → 64 hex chars.
		require.Len(t, string(fp), 64)
	})

	t.Run("Given all three fields differ when computing then fingerprints differ", func(t *testing.T) {
		fp1 := Compute("411111", "1111", "Alice")
		fp2 := Compute("522222", "2222", "Bob")

		require.NotEqual(t, fp1, fp2)
	})
}
