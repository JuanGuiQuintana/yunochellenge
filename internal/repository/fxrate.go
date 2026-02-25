package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/juanatsap/chargeback-api/internal/domain"
)

type pgFxRateRepository struct {
	db *pgxpool.Pool
}

// NewFxRateRepository returns a FxRateRepository backed by PostgreSQL.
func NewFxRateRepository(db *pgxpool.Pool) FxRateRepository {
	return &pgFxRateRepository{db: db}
}

// GetRate returns the USD exchange rate for the given currency on the given date.
// It tries an exact date match first, then falls back to the most recent known rate.
// USD short-circuits without a database query.
func (r *pgFxRateRepository) GetRate(ctx context.Context, currency string, date time.Time) (float64, error) {
	if currency == "USD" {
		return 1.0, nil
	}

	var rate float64

	err := r.db.QueryRow(ctx,
		`SELECT rate_to_usd FROM fx_rates WHERE currency = $1 AND date = $2::date`,
		currency, date,
	).Scan(&rate)

	if err == nil {
		return rate, nil
	}
	if err != pgx.ErrNoRows {
		return 0, fmt.Errorf("fxrate.GetRate exact: %w", err)
	}

	// Fallback to the most recent known rate for this currency.
	err = r.db.QueryRow(ctx,
		`SELECT rate_to_usd FROM fx_rates WHERE currency = $1 ORDER BY date DESC LIMIT 1`,
		currency,
	).Scan(&rate)

	if err == pgx.ErrNoRows {
		return 0, fmt.Errorf("fxrate for %s: %w", currency, domain.ErrFXNotAvailable)
	}
	if err != nil {
		return 0, fmt.Errorf("fxrate.GetRate fallback: %w", err)
	}
	return rate, nil
}
