package scoring

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAmountComponent_Calculate(t *testing.T) {
	tests := []struct {
		name      string
		amountUSD float64
		wantScore int
	}{
		{
			name:      "Given amount exactly 1000 when calculating then score is 25",
			amountUSD: 1000,
			wantScore: 25,
		},
		{
			name:      "Given amount above 1000 when calculating then score is 25",
			amountUSD: 2500,
			wantScore: 25,
		},
		{
			name:      "Given amount exactly 500 when calculating then score is 20",
			amountUSD: 500,
			wantScore: 20,
		},
		{
			name:      "Given amount 999 when calculating then score is 20",
			amountUSD: 999,
			wantScore: 20,
		},
		{
			name:      "Given amount exactly 200 when calculating then score is 15",
			amountUSD: 200,
			wantScore: 15,
		},
		{
			name:      "Given amount 499 when calculating then score is 15",
			amountUSD: 499,
			wantScore: 15,
		},
		{
			name:      "Given amount exactly 100 when calculating then score is 10",
			amountUSD: 100,
			wantScore: 10,
		},
		{
			name:      "Given amount 199 when calculating then score is 10",
			amountUSD: 199,
			wantScore: 10,
		},
		{
			name:      "Given amount exactly 50 when calculating then score is 6",
			amountUSD: 50,
			wantScore: 6,
		},
		{
			name:      "Given amount 99 when calculating then score is 6",
			amountUSD: 99,
			wantScore: 6,
		},
		{
			name:      "Given amount 49 when calculating then score is 3",
			amountUSD: 49,
			wantScore: 3,
		},
		{
			name:      "Given amount 10 when calculating then score is 3",
			amountUSD: 10,
			wantScore: 3,
		},
		{
			name:      "Given amount 0 when calculating then score is 3",
			amountUSD: 0,
			wantScore: 3,
		},
	}

	component := &AmountComponent{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := ScoringInput{AmountUSD: tt.amountUSD}

			result := component.Calculate(input)

			require.Equal(t, "amount", result.Name)
			require.Equal(t, tt.wantScore, result.Score)
			require.NotNil(t, result.Inputs)
			require.Equal(t, tt.amountUSD, result.Inputs["amount_usd"])
		})
	}
}
