// Package parsers reads trade data CSV of particular format
// and inserts into database.
package parsers

import (
	"time"

	"github.com/shopspring/decimal"
)

// Trade is the common structure for an exchange action
type Trade struct {
	Date         time.Time       // Date of trade
	Action       string          // Buy/Sell
	Amount       decimal.Decimal // Number of tokens traded
	Currency     string          // Token traded
	BaseAmount   decimal.Decimal // Amount paid in base currency
	BaseCurrency string          // Currency traded against token
	FeeAmount    decimal.Decimal // Trade fee
	FeeCurrency  string          // Trade fee currency
}

// ValidAction is either a buy or sell
func ValidAction(a string) bool {
	return a == "buy" || a == "sell"
}
