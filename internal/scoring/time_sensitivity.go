package scoring

import "math"

// TimeSensitivityComponent scores urgency based on hours remaining until the dispute deadline.
type TimeSensitivityComponent struct{}

func (c *TimeSensitivityComponent) Calculate(input ScoringInput) ComponentResult {
	hoursRemaining := input.DisputeDeadline.Sub(input.Now).Hours()

	var score int
	switch {
	case hoursRemaining <= 24:
		score = 25
	case hoursRemaining <= 48:
		score = 22
	case hoursRemaining <= 72:
		score = 18
	case hoursRemaining <= 96:
		score = 14
	case hoursRemaining <= 120:
		score = 10
	case hoursRemaining <= 144:
		score = 6
	default:
		score = 2
	}

	return ComponentResult{
		Name:  "time_sensitivity",
		Score: score,
		Inputs: map[string]any{
			"dispute_deadline": input.DisputeDeadline,
			"hours_remaining":  math.Round(hoursRemaining*10) / 10,
		},
	}
}
