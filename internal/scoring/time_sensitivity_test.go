package scoring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimeSensitivityComponent_Calculate(t *testing.T) {
	// A fixed "now" is injected via ScoringInput.Now to keep tests deterministic.
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		hoursRemaining float64
		wantScore      int
	}{
		{
			name:           "Given 1h remaining when calculating then score is 25",
			hoursRemaining: 1,
			wantScore:      25,
		},
		{
			name:           "Given exactly 24h remaining when calculating then score is 25",
			hoursRemaining: 24,
			wantScore:      25,
		},
		{
			name:           "Given 25h remaining when calculating then score is 22",
			hoursRemaining: 25,
			wantScore:      22,
		},
		{
			name:           "Given exactly 48h remaining when calculating then score is 22",
			hoursRemaining: 48,
			wantScore:      22,
		},
		{
			name:           "Given 49h remaining when calculating then score is 18",
			hoursRemaining: 49,
			wantScore:      18,
		},
		{
			name:           "Given exactly 72h remaining when calculating then score is 18",
			hoursRemaining: 72,
			wantScore:      18,
		},
		{
			name:           "Given 73h remaining when calculating then score is 14",
			hoursRemaining: 73,
			wantScore:      14,
		},
		{
			name:           "Given exactly 96h remaining when calculating then score is 14",
			hoursRemaining: 96,
			wantScore:      14,
		},
		{
			name:           "Given 97h remaining when calculating then score is 10",
			hoursRemaining: 97,
			wantScore:      10,
		},
		{
			name:           "Given exactly 120h remaining when calculating then score is 10",
			hoursRemaining: 120,
			wantScore:      10,
		},
		{
			name:           "Given 121h remaining when calculating then score is 6",
			hoursRemaining: 121,
			wantScore:      6,
		},
		{
			name:           "Given exactly 144h remaining when calculating then score is 6",
			hoursRemaining: 144,
			wantScore:      6,
		},
		{
			name:           "Given 145h remaining when calculating then score is 2",
			hoursRemaining: 145,
			wantScore:      2,
		},
		{
			name:           "Given 200h remaining when calculating then score is 2",
			hoursRemaining: 200,
			wantScore:      2,
		},
	}

	component := &TimeSensitivityComponent{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deadline := now.Add(time.Duration(tt.hoursRemaining * float64(time.Hour)))

			input := ScoringInput{
				DisputeDeadline: deadline,
				Now:             now,
			}

			result := component.Calculate(input)

			require.Equal(t, "time_sensitivity", result.Name)
			require.Equal(t, tt.wantScore, result.Score)
			require.NotNil(t, result.Inputs)
			require.Equal(t, deadline, result.Inputs["dispute_deadline"])
		})
	}
}
