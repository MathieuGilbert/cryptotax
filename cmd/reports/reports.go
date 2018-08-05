package reports

import (
	"bytes"
	"sort"
	"time"

	"github.com/mathieugilbert/cryptotax/models"
	"github.com/shopspring/decimal"
)

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

// Trades are sortable by date
type byDate []*models.Trade

func (t byDate) Len() int {
	return len(t)
}
func (t byDate) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
func (t byDate) Less(i, j int) bool {
	if t[i].Date == t[j].Date {
		return t[i].Amount.GreaterThan(t[j].Amount)
	}
	return t[i].Date.Before(t[j].Date)
}

// Converter currency conversion function
type Converter struct {
	Convert func(amount decimal.Decimal, from, to string, on time.Time) decimal.Decimal
}

// RateRequest has a list of currencies to get a quote for at the timestamp
type RateRequest struct {
	Timestamp int64   `json:"timestamp"`
	Rates     []*Rate `json:"rates"`
}

// Rate pairs a currency and its rate
type Rate struct {
	Currency string `json:"currency"`
	Rate     string `json:"rate"`
}

// add extra trades so all are against base currency
func expandAgainstBase(ts []*models.Trade, base string, convert Convert) (rts []*models.Trade, err error) {
	sort.Sort(byDate(ts))

	for _, t := range ts {
		if t.BaseCurrency != base {
			d := t.Date.Unix()
			if !includes(rmap[d], t.BaseCurrency) {
				rmap[d] = append(rmap[d], &Rate{Currency: t.BaseCurrency})
			}
		}
		if t.FeeCurrency != base {
			d := t.Date.Unix()
			if !includes(rmap[d], t.FeeCurrency) {
				rmap[d] = append(rmap[d], &Rate{Currency: t.FeeCurrency})
			}
		}
	}

	for ts, rs := range rmap {
		rrs = append(rrs, &RateRequest{
			Timestamp: ts,
			Rates:     rs,
		})
	}

	return
}

func includes(rs []*Rate, c string) bool {
	for _, r := range rs {
		if r.Currency == c {
			return true
		}
	}
	return false
}

// add extra trades so all are against base currency
func expandAgainstBase(ts []*models.Trade, base string, c Converter) (rts []*models.Trade, err error) {
	for _, t := range ts {
		fa := c.Convert(t.FeeAmount, t.FeeCurrency, base, t.Date)
		ba := c.Convert(t.BaseAmount, t.BaseCurrency, base, t.Date)

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

// tally the total cost and balance for each currency
func tally(ts []*models.Trade) (map[string]decimal.Decimal, map[string]decimal.Decimal, error) {
	sort.Sort(byDate(ts))

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
