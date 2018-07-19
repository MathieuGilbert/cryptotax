package acb

import (
	"bytes"
	"sort"

	"github.com/mathieugilbert/cryptotax/cmd/exchange"
	"github.com/mathieugilbert/cryptotax/models"
	"github.com/shopspring/decimal"
)

// ACB stucture to hold calculated Average Cost Basis
type ACB struct {
	Asset               string
	Action              string
	Quantity            decimal.Decimal
	YearAcquired        int
	Proceeds            decimal.Decimal
	CostBase            decimal.Decimal
	DispositionExpenses decimal.Decimal
	NetIncome           decimal.Decimal
	CoinBalance         decimal.Decimal
}

// Oversold error indicates there were more sold than bought for an asset
type Oversold struct {
	Details map[string]decimal.Decimal
}

func (e *Oversold) Error() string {
	var buf bytes.Buffer
	buf.WriteString("Missing previous buys for:")

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

// Calculate the average cost basis for the stream of trades
func Calculate(trades []*models.Trade, currency string) ([]*ACB, error) {
	// convert trades to base of specified currency
	trades, err := ToBaseCurrency(trades, currency)
	if err != nil {
		return nil, err
	}
	// sort by Asset, Date
	SortAssetDate(trades)

	cost := make(map[string]decimal.Decimal)
	bal := make(map[string]decimal.Decimal)
	oversold := make(map[string]decimal.Decimal)
	acb := []*ACB{}

	for _, t := range trades {
		if t.Action == "buy" {
			cost[t.Asset] = cost[t.Asset].Add(t.BasePrice).Add(t.BaseFee)
			bal[t.Asset] = bal[t.Asset].Add(t.Quantity)

			a := &ACB{
				Asset:        t.Asset,
				Action:       t.Action,
				Quantity:     t.Quantity,
				YearAcquired: t.Date.Year(),
				CostBase:     cost[t.Asset],
				CoinBalance:  bal[t.Asset],
			}
			acb = append(acb, a)
		} else {
			nb := bal[t.Asset].Sub(t.Quantity)
			if nb.IsNegative() {
				oversold[t.Asset] = oversold[t.Asset].Sub(nb)
				continue
			}

			cost[t.Asset] = cost[t.Asset].Div(bal[t.Asset]).Mul(nb)
			bal[t.Asset] = nb

			a := &ACB{
				Asset:               t.Asset,
				Action:              t.Action,
				Quantity:            t.Quantity,
				YearAcquired:        t.Date.Year(),
				Proceeds:            t.BasePrice,
				CostBase:            cost[t.Asset],
				DispositionExpenses: t.BaseFee,
				CoinBalance:         bal[t.Asset],
				NetIncome:           t.BasePrice.Sub(cost[t.Asset]).Sub(t.BaseFee),
			}
			acb = append(acb, a)
		}
	}

	if len(oversold) > 0 {
		return nil, &Oversold{oversold}
	}

	return acb, nil
}

// SortAssetDate sorts the trades by Asset then by Date
func SortAssetDate(ts []*models.Trade) error {
	sort.Slice(ts, func(i, j int) bool {
		if a := ts[i].Asset < ts[j].Asset; a {
			return true
		}
		if a := ts[i].Asset > ts[j].Asset; a {
			return false
		}

		return ts[i].Date.Before(ts[j].Date)
	})

	return nil
}

// ToBaseCurrency converts trades to be valued in base currency
// C2C trades need corresponding sell/buy order also in base
func ToBaseCurrency(ts []*models.Trade, base string) ([]*models.Trade, error) {
	extras := []*models.Trade{}

	for _, t := range ts {
		if t.BaseCurrency != base {
			// get amounts in base currency
			val := t.BasePrice
			fee := t.BaseFee
			r, err := exchange.FetchRate(t.BaseCurrency, base, t.Date)
			if err != nil {
				return nil, err
			}
			val = val.Mul(r)
			fee = fee.Mul(r)

			var act string
			if t.Action == "buy" {
				act = "sell"
			} else {
				act = "buy"
			}

			// build second trade, fee applied to first
			extras = append(extras, &models.Trade{
				Date:         t.Date,
				Asset:        t.BaseCurrency,
				Action:       act,
				Quantity:     t.BasePrice,
				BaseCurrency: base,
				BasePrice:    val,
			})

			// update first trade values
			t.BasePrice = val
			t.BaseFee = fee
			t.BaseCurrency = base
		}
	}

	return append(ts, extras...), nil
}

// SellOnly filters for only sell actions
func SellOnly(all []*ACB) (sells []*ACB, err error) {
	for _, a := range all {
		if a.Action == "sell" {
			sells = append(sells, a)
		}
	}

	return
}
