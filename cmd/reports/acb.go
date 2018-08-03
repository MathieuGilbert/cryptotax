package reports

import (
	"time"

	"github.com/shopspring/decimal"
)

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

func (r *ACB) Build() (string, time.Time) {
	return r.Currency, r.AsOf
}
