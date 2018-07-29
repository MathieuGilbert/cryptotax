package parsers

import (
	"encoding/csv"
	"fmt"
	"html"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Coinbase struct
type Coinbase struct{}

// Parse a Coinbase file
/*
Timestamp: date
Transaction Type: action
Asset: currency
Quantity Transacted: amount
CAD Spot Price at Transaction: unit price
CAD Amount Transacted (Inclusive of Coinbase Fees): baseAmt + feeAmt
Address: withdrawal/deposit address
Notes: description
*/
func (Coinbase) Parse(r *csv.Reader) (trades []Trade, parseError error) {
	for i := 0; ; i++ {
		row, err := r.Read()
		if err == io.EOF {
			break
		}

		// skip header rows
		if i < 3 {
			continue
		}

		// ensure basic structure
		if len(row) != 8 {
			parseError = fmt.Errorf("Wrong number of columns: %v", len(row))
			break
		}

		var date time.Time
		if date, err = time.Parse("01/02/2006", row[0]); err != nil {
			parseError = fmt.Errorf("time.Parse failed: %v", row[0])
			break
		}

		action := strings.ToUpper(row[1])
		// skip if not a buy or sell
		if !ValidAction(action) {
			continue
		}

		currency := strings.ToUpper(html.EscapeString(row[2]))

		var amount decimal.Decimal
		if amount, err = decimal.NewFromString(row[3]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[3])
			break
		}

		var unitPrice decimal.Decimal
		if unitPrice, err = decimal.NewFromString(row[4]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[4])
			break
		}

		var totalCost decimal.Decimal
		if totalCost, err = decimal.NewFromString(row[5]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[5])
			break
		}

		baseAmt := amount.Mul(unitPrice)
		feeAmt := totalCost.Sub(baseAmt)

		// base currency is hidden in
		re := regexp.MustCompile("for\\s.+\\s(\\w+)")
		m := re.FindStringSubmatch(row[7])
		if m == nil {
			parseError = fmt.Errorf("Couldn't determine currency")
			break
		}
		baseCur := strings.ToUpper(m[1])

		trades = append(trades, Trade{
			Date:         date,
			Action:       action,
			Amount:       amount,
			Currency:     currency,
			BaseAmount:   baseAmt,
			BaseCurrency: baseCur,
			FeeAmount:    feeAmt,
			FeeCurrency:  baseCur,
		})
	}
	return
}
