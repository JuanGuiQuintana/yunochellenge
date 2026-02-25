package scoring

// AmountComponent scores risk based on the transaction amount in USD.
type AmountComponent struct{}

func (c *AmountComponent) Calculate(input ScoringInput) ComponentResult {
	var score int
	switch {
	case input.AmountUSD >= 1000:
		score = 25
	case input.AmountUSD >= 500:
		score = 20
	case input.AmountUSD >= 200:
		score = 15
	case input.AmountUSD >= 100:
		score = 10
	case input.AmountUSD >= 50:
		score = 6
	default:
		score = 3
	}

	return ComponentResult{
		Name:  "amount",
		Score: score,
		Inputs: map[string]any{
			"amount_usd": input.AmountUSD,
		},
	}
}
