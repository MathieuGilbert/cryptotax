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

		var cost decimal.Decimal
		if cost, err = decimal.NewFromString(row[5]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[5])
			break
		}

		var quantity decimal.Decimal
		if quantity, err = decimal.NewFromString(row[3]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[3])
			break
		}

		var unitPrice decimal.Decimal
		if unitPrice, err = decimal.NewFromString(row[4]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[4])
			break
		}
		fee := cost.Sub(quantity.Mul(unitPrice))
		cost = cost.Sub(fee)

		action := strings.ToLower(row[1])
		// skip if not a buy or sell
		if !ValidAction(action) {
			continue
		}

		asset := html.EscapeString(row[2])

		// base currency is hidden in
		re := regexp.MustCompile("for\\s.+\\s(\\w+)")
		m := re.FindStringSubmatch(row[7])
		if m == nil {
			parseError = fmt.Errorf("Couldn't determine currency")
			break
		}
		base := m[1]

		trades = append(trades, Trade{
			Date:         date,
			Action:       action,
			Asset:        asset,
			Quantity:     quantity,
			BaseCurrency: base,
			BasePrice:    cost,
			BaseFee:      fee,
		})
	}
	return
}
