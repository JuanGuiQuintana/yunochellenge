package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/juanatsap/chargeback-api/internal/domain"
	"github.com/juanatsap/chargeback-api/internal/processor"
	"github.com/juanatsap/chargeback-api/internal/repository"
	"github.com/juanatsap/chargeback-api/internal/scoring"
)

// IngestResult is returned by IngestService.Ingest.
type IngestResult struct {
	ChargebackID   uuid.UUID
	RiskScore      int
	ScoreBreakdown domain.ScoreBreakdown
	Flags          []string
	IsDuplicate    bool
}

// IngestService orchestrates the full ingestion flow.
type IngestService struct {
	adapters   map[string]processor.ProcessorAdapter
	cbRepo     repository.ChargebackRepository
	mRepo      repository.MerchantRepository
	chRepo     repository.CardholderRepository
	fxRepo     repository.FxRateRepository
	rlRepo     repository.RateLimitRepository
	procRepo   repository.ProcessorRepository
	rcRepo     repository.ReasonCodeRepository
	scoring    *scoring.ScoringEngine
	enrichment *EnrichmentService
	patterns   *PatternDetectionService
	hmac       *HMACValidator
}

// NewIngestService constructs the service with all dependencies.
func NewIngestService(
	adapters map[string]processor.ProcessorAdapter,
	cbRepo repository.ChargebackRepository,
	mRepo repository.MerchantRepository,
	chRepo repository.CardholderRepository,
	fxRepo repository.FxRateRepository,
	rlRepo repository.RateLimitRepository,
	procRepo repository.ProcessorRepository,
	rcRepo repository.ReasonCodeRepository,
	scoringEngine *scoring.ScoringEngine,
	enrichment *EnrichmentService,
	patterns *PatternDetectionService,
) *IngestService {
	return &IngestService{
		adapters:   adapters,
		cbRepo:     cbRepo,
		mRepo:      mRepo,
		chRepo:     chRepo,
		fxRepo:     fxRepo,
		rlRepo:     rlRepo,
		procRepo:   procRepo,
		rcRepo:     rcRepo,
		scoring:    scoringEngine,
		enrichment: enrichment,
		patterns:   patterns,
		hmac:       &HMACValidator{},
	}
}

// Ingest processes a webhook from the given processor.
// Returns ErrUnauthorized if HMAC is invalid.
// Returns ErrRateLimitExceeded if the processor is over quota.
// Returns IngestResult.IsDuplicate=true if the chargeback already exists (idempotent).
func (s *IngestService) Ingest(ctx context.Context, processorName, signature string, body []byte) (IngestResult, error) {
	// Step 1: Resolve processor.
	proc, err := s.procRepo.FindByName(ctx, processorName)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return IngestResult{}, fmt.Errorf("ingest: unknown processor %q: %w", processorName, domain.ErrNotFound)
		}
		return IngestResult{}, fmt.Errorf("ingest.FindProcessor: %w", err)
	}

	// Step 2: Validate HMAC signature (skip if no secret configured).
	if proc.WebhookSecretHash != "" && signature != "" {
		if !s.hmac.Validate([]byte(proc.WebhookSecretHash), signature, body) {
			return IngestResult{}, domain.ErrUnauthorized
		}
	}

	// Step 3: Rate limiting.
	windowMinute := time.Now().UTC().Truncate(time.Minute)
	exceeded, err := s.rlRepo.CheckAndIncrement(ctx, proc.ProcessorID, windowMinute, proc.RateLimitPerMinute)
	if err != nil {
		return IngestResult{}, fmt.Errorf("ingest.RateLimit: %w", err)
	}
	if exceeded {
		return IngestResult{}, domain.ErrRateLimitExceeded
	}

	// Step 4: Normalize payload via adapter.
	adapter, ok := s.adapters[processorName]
	if !ok {
		return IngestResult{}, fmt.Errorf("ingest: no adapter for processor %q: %w", processorName, domain.ErrNotFound)
	}
	dto, err := adapter.Normalize(ctx, body)
	if err != nil {
		return IngestResult{}, fmt.Errorf("ingest.Normalize: %w", err)
	}

	// Step 5: Deduplication.
	existing, found, err := s.cbRepo.FindByProcessorID(ctx, processorName, dto.ProcessorChargebackID)
	if err != nil {
		return IngestResult{}, fmt.Errorf("ingest.Deduplicate: %w", err)
	}
	if found {
		return IngestResult{
			ChargebackID:   existing.ChargebackID,
			RiskScore:      existing.RiskScore,
			ScoreBreakdown: existing.ScoreBreakdown,
			Flags:          existing.Flags,
			IsDuplicate:    true,
		}, nil
	}

	// Step 6: Resolve merchant UUID from string ID.
	merchantID, err := s.resolveMerchantID(ctx, dto.MerchantID)
	if err != nil {
		return IngestResult{}, fmt.Errorf("ingest.ResolveMerchant: %w", err)
	}

	// Step 7: Resolve/create cardholder.
	fingerprint := domain.Compute(dto.BIN, dto.Last4, dto.CardholderName)
	cardholderID, err := s.chRepo.FindOrCreate(ctx, fingerprint, dto.BIN, dto.Last4)
	if err != nil {
		return IngestResult{}, fmt.Errorf("ingest.FindOrCreateCardholder: %w", err)
	}

	// Step 8: Resolve reason code.
	rc, err := s.rcRepo.FindByRawCode(ctx, proc.ProcessorID, dto.RawReasonCode)
	if err != nil {
		return IngestResult{}, fmt.Errorf("ingest.FindReasonCode: %w", err)
	}

	// Step 9: Enrich with FX rate.
	merchant, err := s.mRepo.FindByID(ctx, merchantID)
	if err != nil {
		return IngestResult{}, fmt.Errorf("ingest.FindMerchant: %w", err)
	}

	amountUSD, fxPending, err := s.enrichment.EnrichAmount(ctx, dto.Amount, dto.Currency, dto.NotificationDate)
	if err != nil {
		return IngestResult{}, fmt.Errorf("ingest.EnrichAmount: %w", err)
	}

	// Step 10: Calculate score.
	var usdValue float64
	if amountUSD != nil {
		usdValue = *amountUSD
	}
	scoreInput := scoring.ScoringInput{
		DisputeDeadline:     dto.DisputeDeadline,
		AmountUSD:           usdValue,
		ReasonCodeRiskLevel: int(rc.RiskLevel),
		MerchantRatio:       merchant.CurrentChargebackRatio,
		Now:                 time.Now(),
	}
	riskScore, breakdown := s.scoring.Calculate(scoreInput)
	breakdown.FXPending = fxPending

	// Step 11: Build and persist chargeback.
	chargebackID := uuid.New()
	cb := domain.Chargeback{
		ChargebackID:          chargebackID,
		ProcessorChargebackID: dto.ProcessorChargebackID,
		ProcessorID:           proc.ProcessorID,
		MerchantID:            merchantID,
		TransactionID:         dto.TransactionID,
		CardholderID:          cardholderID,
		ReasonCodeID:          rc.ReasonCodeID,
		Amount:                dto.Amount,
		Currency:              dto.Currency,
		AmountUSD:             amountUSD,
		TransactionDate:       dto.TransactionDate,
		NotificationDate:      dto.NotificationDate,
		DisputeDeadline:       dto.DisputeDeadline,
		Status:                domain.StatusOpen,
		RawPayload:            []byte(dto.RawPayload),
		RiskScore:             riskScore,
		ScoreBreakdown:        breakdown,
		Flags:                 []string{},
	}

	if err := s.cbRepo.Insert(ctx, cb); err != nil {
		return IngestResult{}, fmt.Errorf("ingest.Insert: %w", err)
	}

	// Step 12: Update merchant ratio cache (async-ish, after INSERT).
	if err := s.mRepo.UpdateRatioCache(ctx, merchantID); err != nil {
		// Non-fatal — log but don't fail the request.
		_ = err
	}

	// Step 13: Detect patterns (post-INSERT so the new chargeback is counted).
	flags, err := s.patterns.DetectPatterns(ctx, cardholderID, merchantID, rc.ReasonCodeID)
	if err != nil {
		// Non-fatal — return the result without flags rather than failing.
		flags = []string{}
	}

	// Step 14: Update flags if any detected.
	if len(flags) > 0 {
		if err := s.cbRepo.UpdateFlags(ctx, chargebackID, flags); err != nil {
			_ = err // Non-fatal.
		}
	}

	return IngestResult{
		ChargebackID:   chargebackID,
		RiskScore:      riskScore,
		ScoreBreakdown: breakdown,
		Flags:          flags,
		IsDuplicate:    false,
	}, nil
}

// resolveMerchantID tries to parse the DTO merchant ID as a UUID directly.
// If it's not a UUID, it returns ErrInvalidInput — the processor is expected to send UUIDs.
func (s *IngestService) resolveMerchantID(ctx context.Context, merchantIDStr string) (uuid.UUID, error) {
	id, err := uuid.Parse(merchantIDStr)
	if err == nil {
		return id, nil
	}
	return uuid.UUID{}, fmt.Errorf("merchant ID %q is not a valid UUID: %w", merchantIDStr, domain.ErrInvalidInput)
}

// QueryService wraps the query-side repositories for the API handlers.
// It is a thin facade over the repository layer.
type QueryService struct {
	cbRepo   repository.ChargebackRepository
	mRepo    repository.MerchantRepository
	procRepo repository.ProcessorRepository
}

// NewQueryService constructs a QueryService with the given repositories.
func NewQueryService(
	cbRepo repository.ChargebackRepository,
	mRepo repository.MerchantRepository,
	procRepo repository.ProcessorRepository,
) *QueryService {
	return &QueryService{
		cbRepo:   cbRepo,
		mRepo:    mRepo,
		procRepo: procRepo,
	}
}

// GetChargeback retrieves a single chargeback by its UUID.
func (q *QueryService) GetChargeback(ctx context.Context, id uuid.UUID) (domain.Chargeback, error) {
	return q.cbRepo.FindByID(ctx, id)
}

// ListChargebacks returns a filtered, paginated list of chargebacks and the total count.
func (q *QueryService) ListChargebacks(ctx context.Context, filter domain.ChargebackFilter) ([]domain.Chargeback, int, error) {
	filter.Normalize()
	return q.cbRepo.List(ctx, filter)
}

// Summary returns aggregate counts and metrics across chargebacks, optionally scoped to a merchant.
func (q *QueryService) Summary(ctx context.Context, merchantID *uuid.UUID) (domain.ChargebackSummary, error) {
	return q.cbRepo.Summary(ctx, merchantID)
}

// GetMerchant retrieves a merchant by its UUID.
func (q *QueryService) GetMerchant(ctx context.Context, id uuid.UUID) (domain.Merchant, error) {
	return q.mRepo.FindByID(ctx, id)
}

// GetMerchantStats retrieves chargeback ratio statistics and trend data for a merchant.
func (q *QueryService) GetMerchantStats(ctx context.Context, id uuid.UUID) (domain.MerchantStats, error) {
	return q.mRepo.Stats(ctx, id)
}

// ListProcessors returns all registered processors. Currently returns nil (stretch feature).
func (q *QueryService) ListProcessors(ctx context.Context) ([]domain.Processor, error) {
	return nil, nil
}

// splitAndTrim parses a comma-separated string into a trimmed, non-empty slice of strings.
func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
