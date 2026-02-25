package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/juanatsap/chargeback-api/internal/domain"
	"golang.org/x/sync/errgroup"
)

// PatternDetectionService runs 3 SQL COUNT queries in parallel to detect fraud patterns.
// It has direct pool access to run all 3 queries concurrently.
type PatternDetectionService struct {
	db *pgxpool.Pool
}

// NewPatternDetectionService constructs a PatternDetectionService with the given connection pool.
func NewPatternDetectionService(db *pgxpool.Pool) *PatternDetectionService {
	return &PatternDetectionService{db: db}
}

// DetectPatterns runs the 3 pattern detection queries concurrently and returns
// a slice of flag strings for any detected patterns.
func (s *PatternDetectionService) DetectPatterns(
	ctx context.Context,
	cardholderID uuid.UUID,
	merchantID uuid.UUID,
	reasonCodeID uuid.UUID,
) ([]string, error) {
	g, ctx := errgroup.WithContext(ctx)

	var repeatOffender, hotZone, clustering bool

	g.Go(func() error {
		var count int
		err := s.db.QueryRow(ctx,
			`SELECT COUNT(*) FROM chargebacks
             WHERE cardholder_id = $1
             AND notification_date >= NOW() - INTERVAL '30 days'`,
			cardholderID,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("pattern.repeatOffender: %w", err)
		}
		repeatOffender = count >= 3
		return nil
	})

	g.Go(func() error {
		var count int
		err := s.db.QueryRow(ctx,
			`SELECT COUNT(*) FROM chargebacks
             WHERE merchant_id = $1
             AND notification_date >= NOW() - INTERVAL '7 days'`,
			merchantID,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("pattern.hotZone: %w", err)
		}
		hotZone = count >= 5
		return nil
	})

	g.Go(func() error {
		var count int
		err := s.db.QueryRow(ctx,
			`SELECT COUNT(*) FROM chargebacks
             WHERE merchant_id = $1
             AND reason_code_id = $2
             AND notification_date >= NOW() - INTERVAL '14 days'`,
			merchantID, reasonCodeID,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("pattern.clustering: %w", err)
		}
		clustering = count >= 4
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var flags []string
	if repeatOffender {
		flags = append(flags, string(domain.FlagRepeatOffender))
	}
	if hotZone {
		flags = append(flags, string(domain.FlagMerchantHotZone))
	}
	if clustering {
		flags = append(flags, string(domain.FlagSuspiciousReasonClustering))
	}
	return flags, nil
}
