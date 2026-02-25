package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/juanatsap/chargeback-api/internal/domain"
)

type pgReasonCodeRepository struct {
	db *pgxpool.Pool
}

// NewReasonCodeRepository returns a ReasonCodeRepository backed by PostgreSQL.
func NewReasonCodeRepository(db *pgxpool.Pool) ReasonCodeRepository {
	return &pgReasonCodeRepository{db: db}
}

func (r *pgReasonCodeRepository) FindByRawCode(ctx context.Context, processorID uuid.UUID, rawCode string) (domain.ReasonCode, error) {
	var rc domain.ReasonCode
	err := r.db.QueryRow(ctx,
		`SELECT reason_code_id, processor_id, raw_code, normalized_code, category, risk_level, typical_win_rate, created_at
		 FROM reason_codes WHERE processor_id = $1 AND raw_code = $2`,
		processorID, rawCode,
	).Scan(&rc.ReasonCodeID, &rc.ProcessorID, &rc.RawCode, &rc.NormalizedCode, &rc.Category, &rc.RiskLevel, &rc.TypicalWinRate, &rc.CreatedAt)

	if err == pgx.ErrNoRows {
		return domain.ReasonCode{}, fmt.Errorf("reason_code %q for processor %s: %w", rawCode, processorID, domain.ErrNotFound)
	}
	if err != nil {
		return domain.ReasonCode{}, fmt.Errorf("reasoncode.FindByRawCode: %w", err)
	}
	return rc, nil
}
