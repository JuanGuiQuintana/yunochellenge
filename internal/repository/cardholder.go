package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/juanatsap/chargeback-api/internal/domain"
)

type pgCardholderRepository struct {
	db *pgxpool.Pool
}

// NewCardholderRepository returns a CardholderRepository backed by PostgreSQL.
func NewCardholderRepository(db *pgxpool.Pool) CardholderRepository {
	return &pgCardholderRepository{db: db}
}

// FindOrCreate uses INSERT ... ON CONFLICT to atomically find or create a cardholder.
// Concurrent calls with the same fingerprint will all return the same UUID because
// the ON CONFLICT clause ensures the row is returned regardless of which writer wins.
func (r *pgCardholderRepository) FindOrCreate(ctx context.Context, fingerprint domain.CardFingerprint, bin, last4 string) (uuid.UUID, error) {
	id := uuid.New()

	var resultID uuid.UUID
	err := r.db.QueryRow(ctx,
		`INSERT INTO cardholders (cardholder_id, bin, last4, card_fingerprint)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (card_fingerprint) DO UPDATE SET card_fingerprint = EXCLUDED.card_fingerprint
		 RETURNING cardholder_id`,
		id, bin, last4, string(fingerprint),
	).Scan(&resultID)

	if err != nil {
		return uuid.UUID{}, fmt.Errorf("cardholder.FindOrCreate: %w", err)
	}
	return resultID, nil
}
