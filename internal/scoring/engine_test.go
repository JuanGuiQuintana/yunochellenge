package scoring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestScoringEngine_Calculate(t *testing.T) {
	// Fixed reference point for deterministic time-based scoring.
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		input         ScoringInput
		wantTotal     int
		wantTimeScore int
		wantAmtScore  int
		wantRCScore   int
		wantRatScore  int
	}{
		{
			// Reference case from the spec:
			// deadline=20h => 25, amount=$800 => 20, risk_level=4 => 20, ratio=1.1% => 22 → total=87
			name: "Given reference case when deadline 20h amount 800 risk 4 ratio 1.1pct then total is 87",
			input: ScoringInput{
				DisputeDeadline:     now.Add(20 * time.Hour),
				AmountUSD:           800,
				ReasonCodeRiskLevel: 4,
				MerchantRatio:       0.011,
				Now:                 now,
			},
			wantTotal:     87,
			wantTimeScore: 25,
			wantAmtScore:  20,
			wantRCScore:   20,
			wantRatScore:  22,
		},
		{
			// All components at maximum (25 each) → total must be 100, not 102 or any value > 100.
			name: "Given all components at maximum when calculated then total is capped at 100",
			input: ScoringInput{
				DisputeDeadline:     now.Add(1 * time.Hour),  // <= 24h → 25
				AmountUSD:           1500,                    // >= 1000 → 25
				ReasonCodeRiskLevel: 5,                       // level 5 → 25
				MerchantRatio:       0.02,                    // >= 1.5% → 25
				Now:                 now,
			},
			wantTotal:     100,
			wantTimeScore: 25,
			wantAmtScore:  25,
			wantRCScore:   25,
			wantRatScore:  25,
		},
		{
			// All components at minimum → total should be 1+3+4+2 = 10.
			name: "Given all components at minimum when calculated then total is 10",
			input: ScoringInput{
				DisputeDeadline:     now.Add(200 * time.Hour), // > 144h → 2
				AmountUSD:           10,                       // < 50 → 3
				ReasonCodeRiskLevel: 1,                        // level 1 → 4
				MerchantRatio:       0.001,                    // < 0.3% → 1
				Now:                 now,
			},
			wantTotal:     10,
			wantTimeScore: 2,
			wantAmtScore:  3,
			wantRCScore:   4,
			wantRatScore:  1,
		},
		{
			// Verify time score contribution at the 48h boundary.
			name: "Given deadline 48h amount 200 risk 3 ratio 0.5pct when calculated then correct component scores",
			input: ScoringInput{
				DisputeDeadline:     now.Add(48 * time.Hour), // <= 48h → 22
				AmountUSD:           200,                     // >= 200 → 15
				ReasonCodeRiskLevel: 3,                       // level 3 → 14
				MerchantRatio:       0.005,                   // >= 0.5% → 8
				Now:                 now,
			},
			wantTotal:     59,
			wantTimeScore: 22,
			wantAmtScore:  15,
			wantRCScore:   14,
			wantRatScore:  8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewScoringEngine()

			total, breakdown := engine.Calculate(tt.input)

			require.Equal(t, tt.wantTotal, total)
			require.Equal(t, tt.wantTotal, breakdown.Total)

			require.Equal(t, tt.wantTimeScore, breakdown.TimeScore.Score)
			require.Equal(t, tt.wantAmtScore, breakdown.AmountScore.Score)
			require.Equal(t, tt.wantRCScore, breakdown.ReasonCodeScore.Score)
			require.Equal(t, tt.wantRatScore, breakdown.RatioScore.Score)

			require.LessOrEqual(t, total, 100)
			require.GreaterOrEqual(t, total, 0)
		})
	}
}
