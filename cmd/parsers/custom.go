package parsers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"html"
	"io"
	"strings"
	"time"

	"github.com/mathieugilbert/cryptotax/models"
	"github.com/shopspring/decimal"
)

// Custom trade struct
type Custom struct{}

// Generate a CSV file from the custom entered trades
func (Custom) Generate(ts []*models.Trade) ([]byte, error) {
	records := [][]string{
		{"date", "action", "amount", "currency", "base_amount", "base_currency", "fee_amount", "fee_currency"},
	}

	for _, t := range ts {
		records = append(records, []string{
			t.Date.Format("2006-01-02"),
			t.Action,
			t.Amount.String(),
			t.Currency,
			t.BaseAmount.String(),
			t.BaseCurrency,
			t.FeeAmount.String(),
			t.FeeCurrency,
		})
	}

	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	w.WriteAll(records)

	if err := w.Error(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// Parse a custom trade file
func (Custom) Parse(r *csv.Reader) (trades []Trade, parseError error) {
	// regexp match fields
	header := []string{"date", "action", "amount", "currency", "base_amount", "base_currency", "fee_amount", "fee_currency"}
	onData := false

	for i := 0; ; i++ {
		row, err := r.Read()
		if err == io.EOF {
			break
		}

		// skip header rows
		if !onData {
			if valuesContain(row, header) {
				onData = true
			}
			continue
		}

		var date time.Time
		if date, err = time.Parse("2006-01-02", row[0]); err != nil {
			parseError = fmt.Errorf("time.Parse failed: %v", row[0])
			break
		}

		action := strings.ToUpper(row[1])
		// skip if not a buy or sell
		if !ValidAction(action) {
			continue
		}

		var amount decimal.Decimal
		if amount, err = decimal.NewFromString(row[2]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[2])
			break
		}

		currency := strings.ToUpper(html.EscapeString(row[3]))

		var baseAmt decimal.Decimal
		if baseAmt, err = decimal.NewFromString(row[4]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[4])
			break
		}

		baseCur := strings.ToUpper(html.EscapeString(row[5]))

		var feeAmt decimal.Decimal
		if feeAmt, err = decimal.NewFromString(row[6]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[6])
			break
		}

		feeCur := strings.ToUpper(html.EscapeString(row[7]))

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
