package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgRateLimitRepository struct {
	db *pgxpool.Pool
}

// NewRateLimitRepository returns a RateLimitRepository backed by PostgreSQL.
func NewRateLimitRepository(db *pgxpool.Pool) RateLimitRepository {
	return &pgRateLimitRepository{db: db}
}

// CheckAndIncrement atomically increments the request count for the given processor
// and minute window, then returns true if the new count exceeds the limit.
func (r *pgRateLimitRepository) CheckAndIncrement(ctx context.Context, processorID uuid.UUID, window time.Time, limit int) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`INSERT INTO ingest_rate_limits (processor_id, window_minute, request_count)
		 VALUES ($1, $2, 1)
		 ON CONFLICT (processor_id, window_minute)
		 DO UPDATE SET request_count = ingest_rate_limits.request_count + 1
		 RETURNING request_count`,
		processorID, window,
	).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("ratelimit.CheckAndIncrement: %w", err)
	}
	return count > limit, nil
}
