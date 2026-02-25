package scoring

import (
	"time"

	"github.com/juanatsap/chargeback-api/internal/domain"
)

// ScoringInput aggregates all data needed by the components.
// Now is injected (not time.Now()) to make tests deterministic.
type ScoringInput struct {
	DisputeDeadline     time.Time
	AmountUSD           float64 // 0 if FX unavailable (score_breakdown will note fx_pending)
	ReasonCodeRiskLevel int     // 1-5
	MerchantRatio       float64 // e.g. 0.011 for 1.1%
	Now                 time.Time
}

// ComponentResult is what each component returns.
type ComponentResult struct {
	Name   string
	Score  int
	Inputs map[string]any
}

// ScoringComponent calculates a partial score (0-25).
type ScoringComponent interface {
	Calculate(input ScoringInput) ComponentResult
}

// ScoringEngine orchestrates the 4 independent components.
// Each component contributes 0-25 points for a total of 0-100.
type ScoringEngine struct {
	components []ScoringComponent
}

// NewScoringEngine creates an engine with the standard 4 components.
func NewScoringEngine() *ScoringEngine {
	return &ScoringEngine{
		components: []ScoringComponent{
			&TimeSensitivityComponent{},
			&AmountComponent{},
			&ReasonCodeComponent{},
			&MerchantRatioComponent{},
		},
	}
}

// Calculate computes the total risk score and returns the full breakdown.
func (e *ScoringEngine) Calculate(input ScoringInput) (int, domain.ScoreBreakdown) {
	results := make([]ComponentResult, len(e.components))
	for i, c := range e.components {
		results[i] = c.Calculate(input)
	}

	breakdown := domain.ScoreBreakdown{}
	total := 0

	for _, r := range results {
		switch r.Name {
		case "time_sensitivity":
			breakdown.TimeScore = domain.ComponentScore{Score: r.Score, Inputs: r.Inputs}
		case "amount":
			breakdown.AmountScore = domain.ComponentScore{Score: r.Score, Inputs: r.Inputs}
		case "reason_code":
			breakdown.ReasonCodeScore = domain.ComponentScore{Score: r.Score, Inputs: r.Inputs}
		case "merchant_ratio":
			breakdown.RatioScore = domain.ComponentScore{Score: r.Score, Inputs: r.Inputs}
		}
		total += r.Score
	}

	if total > 100 {
		total = 100
	}
	breakdown.Total = total

	return total, breakdown
}
