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
	Asset        string          // Token traded
	Quantity     decimal.Decimal // Number of tokens traded
	BaseCurrency string          // Currency traded against token
	BasePrice    decimal.Decimal // Amount paid in base currency
	BaseFee      decimal.Decimal // Trade fee in base currency
}

// ValidAction is either a buy or sell
func ValidAction(a string) bool {
	return a == "buy" || a == "sell"
}
