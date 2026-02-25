package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/juanatsap/chargeback-api/internal/domain"
)

type pgMerchantRepository struct {
	db *pgxpool.Pool
}

// NewMerchantRepository returns a MerchantRepository backed by PostgreSQL.
func NewMerchantRepository(db *pgxpool.Pool) MerchantRepository {
	return &pgMerchantRepository{db: db}
}

func (r *pgMerchantRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.Merchant, error) {
	var m domain.Merchant
	err := r.db.QueryRow(ctx,
		`SELECT merchant_id, name, email, current_chargeback_ratio, total_transactions_30d,
		        total_chargebacks_30d, is_flagged_by_processor, created_at, updated_at
		 FROM merchants WHERE merchant_id = $1`,
		id,
	).Scan(&m.MerchantID, &m.Name, &m.Email, &m.CurrentChargebackRatio,
		&m.TotalTransactions30d, &m.TotalChargebacks30d, &m.IsFlaggedByProcessor,
		&m.CreatedAt, &m.UpdatedAt)

	if err == pgx.ErrNoRows {
		return domain.Merchant{}, fmt.Errorf("merchant %s: %w", id, domain.ErrNotFound)
	}
	if err != nil {
		return domain.Merchant{}, fmt.Errorf("merchant.FindByID: %w", err)
	}
	return m, nil
}

// UpdateRatioCache recalculates and persists the 30-day chargeback ratio for a merchant.
// The ratio is computed inline from the chargebacks table so it is always current.
func (r *pgMerchantRepository) UpdateRatioCache(ctx context.Context, merchantID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE merchants SET
		    total_chargebacks_30d = (
		        SELECT COUNT(*) FROM chargebacks
		        WHERE merchant_id = $1
		        AND notification_date >= NOW() - INTERVAL '30 days'
		    ),
		    current_chargeback_ratio = (
		        SELECT COUNT(*)::numeric FROM chargebacks
		        WHERE merchant_id = $1
		        AND notification_date >= NOW() - INTERVAL '30 days'
		    ) / NULLIF(total_transactions_30d, 0),
		    updated_at = NOW()
		 WHERE merchant_id = $1`,
		merchantID,
	)
	if err != nil {
		return fmt.Errorf("merchant.UpdateRatioCache: %w", err)
	}
	return nil
}

func (r *pgMerchantRepository) Stats(ctx context.Context, merchantID uuid.UUID) (domain.MerchantStats, error) {
	m, err := r.FindByID(ctx, merchantID)
	if err != nil {
		return domain.MerchantStats{}, err
	}

	stats := domain.MerchantStats{
		MerchantID:             m.MerchantID,
		Name:                   m.Name,
		Email:                  m.Email,
		CurrentChargebackRatio: m.CurrentChargebackRatio,
		TotalTransactions30d:   m.TotalTransactions30d,
		TotalChargebacks30d:    m.TotalChargebacks30d,
		IsFlaggedByProcessor:   m.IsFlaggedByProcessor,
		RiskLevel:              classifyRisk(m.CurrentChargebackRatio),
	}

	rows, err := r.db.Query(ctx,
		`SELECT date, chargeback_count, transaction_count, ratio
		 FROM merchant_ratio_snapshots
		 WHERE merchant_id = $1
		 ORDER BY date DESC
		 LIMIT 30`,
		merchantID,
	)
	if err != nil {
		return domain.MerchantStats{}, fmt.Errorf("merchant.Stats snapshots: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s domain.RatioSnapshot
		if err := rows.Scan(&s.Date, &s.ChargebackCount, &s.TransactionCount, &s.Ratio); err != nil {
			return domain.MerchantStats{}, fmt.Errorf("merchant.Stats scan: %w", err)
		}
		stats.RatioTrend = append(stats.RatioTrend, s)
	}

	return stats, rows.Err()
}

// classifyRisk maps a chargeback ratio to the Visa/Mastercard monitoring tier labels
// used in the MerchantStats.RiskLevel field.
func classifyRisk(ratio float64) string {
	switch {
	case ratio >= 0.015:
		return "termination"
	case ratio >= 0.010:
		return "critical"
	case ratio >= 0.009:
		return "monitoring"
	case ratio >= 0.007:
		return "alert"
	case ratio >= 0.005:
		return "yellow"
	default:
		return "healthy"
	}
}
