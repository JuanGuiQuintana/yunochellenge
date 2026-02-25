package scoring

// MerchantRatioComponent scores risk based on the merchant's chargeback ratio.
// The ratio is expressed as a decimal (e.g. 0.011 = 1.1%).
type MerchantRatioComponent struct{}

func (c *MerchantRatioComponent) Calculate(input ScoringInput) ComponentResult {
	var score int
	switch {
	case input.MerchantRatio >= 0.015:
		score = 25
	case input.MerchantRatio >= 0.010:
		score = 22
	case input.MerchantRatio >= 0.009:
		score = 18
	case input.MerchantRatio >= 0.007:
		score = 13
	case input.MerchantRatio >= 0.005:
		score = 8
	case input.MerchantRatio >= 0.003:
		score = 4
	default:
		score = 1
	}

	return ComponentResult{
		Name:  "merchant_ratio",
		Score: score,
		Inputs: map[string]any{
			"ratio":     input.MerchantRatio,
			"ratio_pct": input.MerchantRatio * 100,
		},
	}
}
