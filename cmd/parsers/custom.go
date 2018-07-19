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
		{"date", "asset", "action", "quantity", "base_price", "base_fee", "base_currency"},
	}

	for _, t := range ts {
		records = append(records, []string{
			t.Date.Format("2006-01-02"),
			t.Asset,
			t.Action,
			t.Quantity.String(),
			t.BasePrice.String(),
			t.BaseFee.String(),
			t.BaseCurrency,
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
		if len(row) != 7 {
			parseError = fmt.Errorf("Wrong number of columns: %v", len(row))
			break
		}

		var date time.Time
		if date, err = time.Parse("2006-01-02", row[0]); err != nil {
			parseError = fmt.Errorf("time.Parse failed: %v", row[0])
			break
		}

		asset := strings.ToUpper(html.EscapeString(row[1]))

		action := strings.ToLower(row[2])
		// skip if not a buy or sell
		if !ValidAction(action) {
			continue
		}

		var quantity decimal.Decimal
		if quantity, err = decimal.NewFromString(row[3]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[3])
			break
		}

		var cost decimal.Decimal
		if cost, err = decimal.NewFromString(row[4]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[4])
			break
		}

		var fee decimal.Decimal
		if fee, err = decimal.NewFromString(row[5]); err != nil {
			parseError = fmt.Errorf("decimal.NewFromString failed: %v", row[5])
			break
		}

		base := strings.ToUpper(html.EscapeString(row[6]))

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
