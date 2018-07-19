package parsers

import (
	"encoding/csv"
	"fmt"
	"html"
	"io"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Kucoin struct
type Kucoin struct{}

// Parse a Kucoin file
func (Kucoin) Parse(r *csv.Reader) (trades []Trade, parseError error) {
	for i := 0; ; i++ {
		row, err := r.Read()
		if err == io.EOF {
			break
		}

		// skip header rows
		if i < 1 {
			continue
		}

		// ensure basic structure
		if len(row) != 11 {
			parseError = fmt.Errorf("Wrong number of columns: %v", len(row))
			break
		}

		var date time.Time
		if date, err = time.Parse("2006-01-02 15:04:05", row[0]); err != nil {
			parseError = fmt.Errorf("time.Parse failed: %v", row[0])
			break
		}

		var quantity decimal.Decimal
		if quantity, err = decimal.NewFromString(row[5]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[5])
			break
		}

		// in BTC
		var cost decimal.Decimal
		if cost, err = decimal.NewFromString(row[7]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[7])
			break
		}

		// convert fee to base currency
		var unitPrice decimal.Decimal
		if unitPrice, err = decimal.NewFromString(row[3]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[3])
			break
		}
		var fee decimal.Decimal
		if fee, err = decimal.NewFromString(row[9]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[9])
			break
		}
		fee = fee.Mul(unitPrice)

		action := strings.ToLower(row[2])
		// skip if not a buy or sell
		if !ValidAction(action) {
			continue
		}

		asset := html.EscapeString(row[6])
		base := html.EscapeString(row[4])

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
