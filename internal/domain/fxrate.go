package domain

import "time"

// FxRate holds the exchange rate for a currency to USD on a specific date.
type FxRate struct {
	Currency  string
	Date      time.Time
	RateToUSD float64
	Source    string
	CreatedAt time.Time
}
