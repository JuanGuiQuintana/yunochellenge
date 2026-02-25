package scoring

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMerchantRatioComponent_Calculate(t *testing.T) {
	tests := []struct {
		name          string
		merchantRatio float64
		wantScore     int
	}{
		{
			// >= 1.5% → 25
			name:          "Given ratio exactly 1.5pct when calculating then score is 25",
			merchantRatio: 0.015,
			wantScore:     25,
		},
		{
			name:          "Given ratio above 1.5pct when calculating then score is 25",
			merchantRatio: 0.02,
			wantScore:     25,
		},
		{
			// 1.0-1.49% → 22
			name:          "Given ratio exactly 1.0pct when calculating then score is 22",
			merchantRatio: 0.010,
			wantScore:     22,
		},
		{
			name:          "Given ratio 1.1pct when calculating then score is 22",
			merchantRatio: 0.011,
			wantScore:     22,
		},
		{
			name:          "Given ratio 1.49pct when calculating then score is 22",
			merchantRatio: 0.0149,
			wantScore:     22,
		},
		{
			// 0.9-0.99% → 18
			name:          "Given ratio exactly 0.9pct when calculating then score is 18",
			merchantRatio: 0.009,
			wantScore:     18,
		},
		{
			name:          "Given ratio 0.95pct when calculating then score is 18",
			merchantRatio: 0.0095,
			wantScore:     18,
		},
		{
			// 0.7-0.89% → 13
			name:          "Given ratio exactly 0.7pct when calculating then score is 13",
			merchantRatio: 0.007,
			wantScore:     13,
		},
		{
			name:          "Given ratio 0.8pct when calculating then score is 13",
			merchantRatio: 0.008,
			wantScore:     13,
		},
		{
			// 0.5-0.69% → 8
			name:          "Given ratio exactly 0.5pct when calculating then score is 8",
			merchantRatio: 0.005,
			wantScore:     8,
		},
		{
			name:          "Given ratio 0.6pct when calculating then score is 8",
			merchantRatio: 0.006,
			wantScore:     8,
		},
		{
			// 0.3-0.49% → 4
			name:          "Given ratio exactly 0.3pct when calculating then score is 4",
			merchantRatio: 0.003,
			wantScore:     4,
		},
		{
			name:          "Given ratio 0.4pct when calculating then score is 4",
			merchantRatio: 0.004,
			wantScore:     4,
		},
		{
			// < 0.3% → 1
			name:          "Given ratio 0.29pct when calculating then score is 1",
			merchantRatio: 0.0029,
			wantScore:     1,
		},
		{
			name:          "Given ratio 0 when calculating then score is 1",
			merchantRatio: 0.0,
			wantScore:     1,
		},
	}

	component := &MerchantRatioComponent{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := ScoringInput{MerchantRatio: tt.merchantRatio}

			result := component.Calculate(input)

			require.Equal(t, "merchant_ratio", result.Name)
			require.Equal(t, tt.wantScore, result.Score)
			require.NotNil(t, result.Inputs)
			require.Equal(t, tt.merchantRatio, result.Inputs["ratio"])
			require.InDelta(t, tt.merchantRatio*100, result.Inputs["ratio_pct"], 1e-9)
		})
	}
}
