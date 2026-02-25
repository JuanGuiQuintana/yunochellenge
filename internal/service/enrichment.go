package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/juanatsap/chargeback-api/internal/domain"
	"github.com/juanatsap/chargeback-api/internal/repository"
)

// EnrichmentService resolves FX rates and calculates amount_usd.
type EnrichmentService struct {
	fxRepo repository.FxRateRepository
}

// NewEnrichmentService constructs an EnrichmentService with the given FX rate repository.
func NewEnrichmentService(fxRepo repository.FxRateRepository) *EnrichmentService {
	return &EnrichmentService{fxRepo: fxRepo}
}

// EnrichAmount returns the USD equivalent of amount (in cents) and whether FX is pending.
// amountCents is the original amount in the original currency's smallest unit.
// Returns (amountUSD, fxPending, error).
// If FX rate is unavailable, returns (nil, true, nil) — not an error, just a pending state.
func (s *EnrichmentService) EnrichAmount(ctx context.Context, amountCents int64, currency string, date time.Time) (amountUSD *float64, fxPending bool, err error) {
	rate, err := s.fxRepo.GetRate(ctx, currency, date)
	if err != nil {
		if errors.Is(err, domain.ErrFXNotAvailable) {
			return nil, true, nil // pending — not an error for the caller
		}
		return nil, false, fmt.Errorf("enrichment.EnrichAmount: %w", err)
	}

	// Convert cents to major unit, then apply FX rate.
	amountMajor := float64(amountCents) / 100.0
	usd := amountMajor * rate
	return &usd, false, nil
}
