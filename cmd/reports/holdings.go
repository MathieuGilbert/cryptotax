package reports

import (
	"errors"
	"time"

	"github.com/mathieugilbert/cryptotax/models"
	"github.com/shopspring/decimal"
)

type Holdings struct {
	Currency string
	Items    []*HoldingItem
}

type HoldingItem struct {
	Asset  string
	Amount decimal.Decimal
	ACB    decimal.Decimal
	Value  decimal.Decimal
	Gain   decimal.Decimal
}

// Build the report
func (r *Holdings) Build(ts []*models.Trade, c Converter) error {
	if r.Currency == "" {
		return errors.New("Invalid currency")
	}

	trades, err := expandAgainstBase(ts, r.Currency, c)
	if err != nil {
		return err
	}

	cost, bal, err := tally(trades)
	if err != nil {
		return err
	}

	for curr := range cost {
		val := c.Convert(bal[curr], curr, r.Currency, time.Now())
		gain := val.Div(cost[curr]).Sub(decimal.NewFromFloat(1))

		r.Items = append(r.Items, &HoldingItem{
			Asset:  curr,
			Amount: bal[curr],
			ACB:    cost[curr],
			Value:  val,
			Gain:   gain,
		})
	}

	return nil
}
