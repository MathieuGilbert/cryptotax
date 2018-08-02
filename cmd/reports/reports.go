package reports

import (
	"bytes"
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

type ACB struct {
	Currency string
	AsOf     time.Time
	Items    []*ACBItem
}

type ACBItem struct {
	Asset     string
	Acquired  int
	Amount    decimal.Decimal
	Proceeds  decimal.Decimal
	ACB       decimal.Decimal
	Expenses  decimal.Decimal
	NetIncome decimal.Decimal
	Gain      decimal.Decimal
}

type Convert func(amount decimal.Decimal, from, to string, on time.Time) (decimal.Decimal, error)

// add extra trades so all are against base currency
func expandAgainstBase(ts []*models.Trade, base string, convert Convert) (rts []*models.Trade, err error) {
	for _, t := range ts {
		fa, err := convert(t.FeeAmount, t.FeeCurrency, base, t.Date)
		if err != nil {
			return nil, err
		}
		ba, err := convert(t.BaseAmount, t.BaseCurrency, base, t.Date)
		if err != nil {
			return nil, err
		}

		if t.Action == "BUY" {
			rts = append(rts, &models.Trade{
				Date:         t.Date,
				Action:       "BUY",
				Amount:       t.Amount,
				Currency:     t.Currency,
				BaseAmount:   ba,
				BaseCurrency: base,
				FeeAmount:    fa,
				FeeCurrency:  base,
			})

			if t.BaseCurrency != base {
				// cross pair, need an extra sell
				rts = append(rts, &models.Trade{
					Date:         t.Date,
					Action:       "SELL",
					Amount:       t.BaseAmount,
					Currency:     t.BaseCurrency,
					BaseAmount:   ba,
					BaseCurrency: base,
					FeeAmount:    decimal.NewFromFloat(0),
					FeeCurrency:  base,
				})
			}
			if t.FeeCurrency != base {
				// cross pair, need an extra sell
				rts = append(rts, &models.Trade{
					Date:         t.Date,
					Action:       "SELL",
					Amount:       t.FeeAmount,
					Currency:     t.FeeCurrency,
					BaseAmount:   fa,
					BaseCurrency: base,
					FeeAmount:    decimal.NewFromFloat(0),
					FeeCurrency:  base,
				})
			}
		} else if t.Action == "SELL" {
			rts = append(rts, &models.Trade{
				Date:         t.Date,
				Action:       "SELL",
				Amount:       t.Amount,
				Currency:     t.Currency,
				BaseAmount:   ba,
				BaseCurrency: base,
				FeeAmount:    fa,
				FeeCurrency:  base,
			})

			if t.BaseCurrency != base {
				// cross pair, need an extra buy
				rts = append(rts, &models.Trade{
					Date:         t.Date,
					Action:       "BUY",
					Amount:       t.BaseAmount,
					Currency:     t.BaseCurrency,
					BaseAmount:   ba,
					BaseCurrency: base,
					FeeAmount:    decimal.NewFromFloat(0),
					FeeCurrency:  base,
				})
			}

			if t.FeeCurrency != base {
				// cross pair, need an extra sell
				rts = append(rts, &models.Trade{
					Date:         t.Date,
					Action:       "SELL",
					Amount:       t.FeeAmount,
					Currency:     t.FeeCurrency,
					BaseAmount:   fa,
					BaseCurrency: base,
					FeeAmount:    decimal.NewFromFloat(0),
					FeeCurrency:  base,
				})
			}
		}
	}

	return
}

func tally(ts []*models.Trade) (map[string]decimal.Decimal, map[string]decimal.Decimal, error) {
	cost := make(map[string]decimal.Decimal)
	bal := make(map[string]decimal.Decimal)
	oversold := make(map[string]decimal.Decimal)

	for _, t := range ts {
		if t.Action == "BUY" {
			cost[t.Currency] = cost[t.Currency].Add(t.BaseAmount).Add(t.FeeAmount)
			bal[t.Currency] = bal[t.Currency].Add(t.Amount)
		}
		if t.Action == "SELL" {
			newb := bal[t.Currency].Sub(t.Amount)
			if newb.IsNegative() {
				oversold[t.Currency] = oversold[t.Currency].Sub(newb)
				continue
			}
			cost[t.Currency] = cost[t.Currency].Div(bal[t.Currency]).Mul(newb)
			bal[t.Currency] = newb
		}
	}

	if len(oversold) > 0 {
		return nil, nil, &Oversold{oversold}
	}
	return cost, bal, nil
}

// Oversold error indicates there were more sold than bought for an asset
type Oversold struct {
	Details map[string]decimal.Decimal
}

func (e *Oversold) Error() string {
	var buf bytes.Buffer
	buf.WriteString("Missing buy trades for:")

	for asset, amount := range e.Details {
		buf.WriteString(" ")
		buf.WriteString(asset)
		buf.WriteString(" (")
		buf.WriteString(amount.StringScaled(-10))
		buf.WriteString("),")
	}
	s := buf.String()
	return s[:len(s)-1]
}

func (r *Holdings) Build(ts []*models.Trade, convert Convert) error {
	if r.Currency == "" {
		return errors.New("Invalid currency")
	}

	trades, err := expandAgainstBase(ts, r.Currency, convert)
	if err != nil {
		return err
	}

	cost, bal, err := tally(trades)
	if err != nil {
		return err
	}

	for curr := range cost {
		val, err := convert(bal[curr], curr, r.Currency, time.Now())
		if err != nil {
			return err
		}
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

func (r ACB) Build() (string, time.Time) {
	return r.Currency, r.AsOf
}
