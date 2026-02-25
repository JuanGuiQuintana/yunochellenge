//go:build ignore

package scoring

import (
	"fmt"
	"time"
)

func main() {
	now := time.Now()
	engine := NewScoringEngine()

	input := ScoringInput{
		DisputeDeadline:     now.Add(20 * time.Hour),
		AmountUSD:           800,
		ReasonCodeRiskLevel: 4,
		MerchantRatio:       0.011,
		Now:                 now,
	}

	total, breakdown := engine.Calculate(input)
	fmt.Printf("time_score    = %d\n", breakdown.TimeScore.Score)
	fmt.Printf("amount_score  = %d\n", breakdown.AmountScore.Score)
	fmt.Printf("reason_code   = %d\n", breakdown.ReasonCodeScore.Score)
	fmt.Printf("ratio_score   = %d\n", breakdown.RatioScore.Score)
	fmt.Printf("total         = %d\n", total)
	fmt.Printf("expected 87   = match: %v\n", total == 87)
}
