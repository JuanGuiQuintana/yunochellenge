package domain

import "encoding/json"

// ScoreBreakdown holds per-component scores and their inputs for transparency.
// Stored as JSONB in PostgreSQL.
type ScoreBreakdown struct {
	TimeScore       ComponentScore `json:"time_score"`
	AmountScore     ComponentScore `json:"amount_score"`
	ReasonCodeScore ComponentScore `json:"reason_code_score"`
	RatioScore      ComponentScore `json:"ratio_score"`
	Total           int            `json:"total"`
	FXPending       bool           `json:"fx_pending,omitempty"`
}

// ComponentScore holds the score and its contributing inputs for one scoring dimension.
type ComponentScore struct {
	Score  int            `json:"score"`
	Inputs map[string]any `json:"inputs"`
}

// MarshalJSON serializes ScoreBreakdown to JSON bytes for JSONB storage.
func (s ScoreBreakdown) MarshalJSON() ([]byte, error) {
	type Alias ScoreBreakdown
	return json.Marshal(Alias(s))
}

// UnmarshalJSON deserializes from JSONB bytes.
func (s *ScoreBreakdown) UnmarshalJSON(data []byte) error {
	type Alias ScoreBreakdown
	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*s = ScoreBreakdown(alias)
	return nil
}

// ToBytes serializes to []byte for pgx JSONB storage.
func (s ScoreBreakdown) ToBytes() ([]byte, error) {
	return json.Marshal(s)
}

// ScoreBreakdownFromBytes deserializes from pgx JSONB bytes.
func ScoreBreakdownFromBytes(data []byte) (ScoreBreakdown, error) {
	var sb ScoreBreakdown
	if err := json.Unmarshal(data, &sb); err != nil {
		return sb, err
	}
	return sb, nil
}
