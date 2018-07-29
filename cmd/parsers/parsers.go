// Package parsers reads trade data CSV of particular format
// and inserts into database.
package parsers

import (
	"errors"
	"regexp"
	"strings"
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

// NewParser returns the matching type of parser
func NewParser(name string) (p interface{}, err error) {
	switch name {
	case "Coinbase":
		p = &Coinbase{}
	case "Kucoin":
		p = &Kucoin{}
	case "Cryptotax":
		p = &Custom{}
	default:
		err = errors.New("Invalid exchange")
	}
	return p, err
}

// ValidAction is either a buy or sell
func ValidAction(a string) bool {
	a = strings.ToUpper(a)
	return a == "BUY" || a == "SELL"
}

// match: "Transacted CAD" with "Transacted"
func valuesContain(full, test []string) bool {
	if len(full) != len(test) {
		return false
	}
	for i, s := range full {
		m, _ := regexp.MatchString(test[i], s)

		if !m {
			return false
		}
	}
	return true
}
