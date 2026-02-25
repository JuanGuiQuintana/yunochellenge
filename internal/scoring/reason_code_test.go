package scoring

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReasonCodeComponent_Calculate(t *testing.T) {
	tests := []struct {
		name      string
		riskLevel int
		wantScore int
	}{
		{
			name:      "Given risk level 1 when calculating then score is 4",
			riskLevel: 1,
			wantScore: 4,
		},
		{
			name:      "Given risk level 2 when calculating then score is 8",
			riskLevel: 2,
			wantScore: 8,
		},
		{
			name:      "Given risk level 3 when calculating then score is 14",
			riskLevel: 3,
			wantScore: 14,
		},
		{
			name:      "Given risk level 4 when calculating then score is 20",
			riskLevel: 4,
			wantScore: 20,
		},
		{
			name:      "Given risk level 5 when calculating then score is 25",
			riskLevel: 5,
			wantScore: 25,
		},
		{
			// Unknown risk level (0) → treated as minimal, same as level 1.
			name:      "Given unknown risk level 0 when calculating then score defaults to 4",
			riskLevel: 0,
			wantScore: 4,
		},
		{
			// Out-of-range risk level (6) → treated as default minimal.
			name:      "Given out-of-range risk level 6 when calculating then score defaults to 4",
			riskLevel: 6,
			wantScore: 4,
		},
		{
			// Negative risk level → default minimal.
			name:      "Given negative risk level when calculating then score defaults to 4",
			riskLevel: -1,
			wantScore: 4,
		},
	}

	component := &ReasonCodeComponent{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := ScoringInput{ReasonCodeRiskLevel: tt.riskLevel}

			result := component.Calculate(input)

			require.Equal(t, "reason_code", result.Name)
			require.Equal(t, tt.wantScore, result.Score)
			require.NotNil(t, result.Inputs)
			require.Equal(t, tt.riskLevel, result.Inputs["risk_level"])
		})
	}
}
