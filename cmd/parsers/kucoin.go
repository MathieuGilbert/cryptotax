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
/*
Time: date
Coins: trading pair
Sell/Buy: action
Filled Price: unit base price
Coin: unit base currency
Amount: amount
Coin: currency
Volume: base amount
Coin: base currency
Fee: fee amount
Coin: fee currency
*/
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

		action := strings.ToUpper(row[2])
		// skip if not a buy or sell
		if !ValidAction(action) {
			continue
		}

		var amount decimal.Decimal
		if amount, err = decimal.NewFromString(row[5]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[5])
			break
		}
		currency := strings.ToUpper(html.EscapeString(row[6]))

		var baseAmt decimal.Decimal
		if baseAmt, err = decimal.NewFromString(row[7]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[7])
			break
		}
		baseCur := strings.ToUpper(html.EscapeString(row[8]))

		var feeAmt decimal.Decimal
		if feeAmt, err = decimal.NewFromString(row[9]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[9])
			break
		}
		feeCur := strings.ToUpper(html.EscapeString(row[10]))

		trades = append(trades, Trade{
			Date:         date,
			Action:       action,
			Amount:       amount,
			Currency:     currency,
			BaseAmount:   baseAmt,
			BaseCurrency: baseCur,
			FeeAmount:    feeAmt,
			FeeCurrency:  feeCur,
		})
	}
	return
}
