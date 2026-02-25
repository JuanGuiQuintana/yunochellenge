package scoring

// ReasonCodeComponent scores risk based on the reason code's risk level (1-5).
type ReasonCodeComponent struct{}

func (c *ReasonCodeComponent) Calculate(input ScoringInput) ComponentResult {
	var score int
	switch input.ReasonCodeRiskLevel {
	case 5:
		score = 25
	case 4:
		score = 20
	case 3:
		score = 14
	case 2:
		score = 8
	case 1:
		score = 4
	default:
		score = 4 // unknown → treat as minimal
	}

	return ComponentResult{
		Name:  "reason_code",
		Score: score,
		Inputs: map[string]any{
			"risk_level": input.ReasonCodeRiskLevel,
		},
	}
}
