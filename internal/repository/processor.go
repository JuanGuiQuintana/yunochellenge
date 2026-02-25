package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/juanatsap/chargeback-api/internal/domain"
)

type pgProcessorRepository struct {
	db    *pgxpool.Pool
	mu    sync.RWMutex
	cache map[string]domain.Processor
}

// NewProcessorRepository returns a ProcessorRepository backed by PostgreSQL with an
// in-memory read-through cache. Processors are immutable in practice, so the cache
// is never invalidated — a process restart clears it if needed.
func NewProcessorRepository(db *pgxpool.Pool) ProcessorRepository {
	return &pgProcessorRepository{
		db:    db,
		cache: make(map[string]domain.Processor),
	}
}

func (r *pgProcessorRepository) FindByName(ctx context.Context, name string) (domain.Processor, error) {
	r.mu.RLock()
	if p, ok := r.cache[name]; ok {
		r.mu.RUnlock()
		return p, nil
	}
	r.mu.RUnlock()

	var p domain.Processor
	err := r.db.QueryRow(ctx,
		`SELECT processor_id, name, webhook_secret_hash, rate_limit_per_minute, is_active, created_at, updated_at
		 FROM processors WHERE name = $1 AND is_active = true`,
		name,
	).Scan(&p.ProcessorID, &p.Name, &p.WebhookSecretHash, &p.RateLimitPerMinute, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)

	if err == pgx.ErrNoRows {
		return domain.Processor{}, fmt.Errorf("processor %q: %w", name, domain.ErrNotFound)
	}
	if err != nil {
		return domain.Processor{}, fmt.Errorf("processor.FindByName: %w", err)
	}

	r.mu.Lock()
	r.cache[name] = p
	r.mu.Unlock()

	return p, nil
}
