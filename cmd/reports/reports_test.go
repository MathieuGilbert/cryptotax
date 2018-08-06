package reports

import (
	"testing"
	"time"

	"github.com/mathieugilbert/cryptotax/models"
	"github.com/shopspring/decimal"
)

var trades = []*models.Trade{
	&models.Trade{
		Date:         time.Now().AddDate(0, 0, -10),
		Action:       "BUY",
		Amount:       decimal.NewFromFloat(200),
		Currency:     "AAA",
		BaseAmount:   decimal.NewFromFloat(2000),
		BaseCurrency: "CAD",
		FeeAmount:    decimal.NewFromFloat(1),
		FeeCurrency:  "AAA",
	},
	// AAA: bal = 199, cost = (2000 + 1 * 2) * 199/200 = 1991.99
	&models.Trade{
		Date:         time.Now().AddDate(0, 0, -9),
		Action:       "BUY",
		Amount:       decimal.NewFromFloat(100),
		Currency:     "BBB",
		BaseAmount:   decimal.NewFromFloat(100),
		BaseCurrency: "AAA",
		FeeAmount:    decimal.NewFromFloat(10),
		FeeCurrency:  "BBB",
	},
	// AAA: bal = 99, cost = 1991.99 * 99/199 = 990.99
	// BBB: bal = 90, cost = (100 * 2 + 10 * 2) * 90/100 = 198.00
	&models.Trade{
		Date:         time.Now().AddDate(0, 0, -8),
		Action:       "SELL",
		Amount:       decimal.NewFromFloat(50),
		Currency:     "BBB",
		BaseAmount:   decimal.NewFromFloat(100),
		BaseCurrency: "AAA",
		FeeAmount:    decimal.NewFromFloat(10),
		FeeCurrency:  "BBB",
	},
	// AAA: bal = 199, cost = 990.99 + 100 * 2 = 1190.99
	// BBB: bal = 30, cost = 198.00 * 40/90 = 88.00, 88.00 * 30/40 = 66.00
	&models.Trade{
		Date:         time.Now().AddDate(0, 0, -8),
		Action:       "SELL",
		Amount:       decimal.NewFromFloat(100),
		Currency:     "AAA",
		BaseAmount:   decimal.NewFromFloat(2000),
		BaseCurrency: "CAD",
		FeeAmount:    decimal.NewFromFloat(10),
		FeeCurrency:  "CAD",
	},
	// AAA: bal = 99, cost = 1190.99 * 99/199 = 592.5025628
	// BBB: bal = 30, cost = 66.00
}

var c = Converter{
	Convert: func(amount decimal.Decimal, from, to string, on time.Time) decimal.Decimal {
		if from == to {
			return amount
		}

		return amount.Mul(decimal.NewFromFloat(2))
	},
}

func TestBuildHoldings(t *testing.T) {
	r := &Holdings{}
	if err := r.Build(trades, c); err == nil {
		t.Errorf("Should require currency set.")
	}

	r.Currency = "CAD"
	if err := r.Build(trades, c); err != nil {
		t.Errorf("Should build correctly.")
	}

	if len(r.Items) != 2 {
		t.Errorf("Should have 2 items (1 per currency), not %v.", len(r.Items))
	}

	exp1 := &HoldingItem{
		Asset:  "AAA",
		Amount: decimal.NewFromFloat(99),
		ACB:    decimal.NewFromFloat(592.502563),
		Value:  decimal.NewFromFloat(0),
		Gain:   decimal.NewFromFloat(0),
	}
	exp2 := &HoldingItem{
		Asset:  "BBB",
		Amount: decimal.NewFromFloat(30),
		ACB:    decimal.NewFromFloat(66.00),
		Value:  decimal.NewFromFloat(0),
		Gain:   decimal.NewFromFloat(0),
	}

	item := &HoldingItem{}
	for _, i := range r.Items {
		if i.Asset == "AAA" {
			item = i
			break
		}
	}
	if item.Asset != exp1.Asset {
		t.Errorf("Asset didn't match. Wanted: %v, got: %v.", exp1.Asset, item.Asset)
	}
	if !theSame(item.Amount, exp1.Amount) {
		t.Errorf("Amount didn't match. Wanted: %v, got: %v.", exp1.Amount, item.Amount)
	}
	if !theSame(item.ACB, exp1.ACB) {
		t.Errorf("ACB didn't match. Wanted: %v, got: %v.", exp1.ACB, item.ACB)
	}
	if !theSame(item.Value, exp1.Value) {
		t.Errorf("Value didn't match. Wanted: %v, got: %v.", exp1.Value, item.Value)
	}
	if !theSame(item.Gain, exp1.Gain) {
		t.Errorf("Gain didn't match. Wanted: %v, got: %v.", exp1.Gain, item.Gain)
	}

	for _, i := range r.Items {
		if i.Asset == "BBB" {
			item = i
			break
		}
	}
	if item.Asset != exp2.Asset {
		t.Errorf("Asset didn't match. Wanted: %v, got: %v.", exp2.Asset, item.Asset)
	}
	if !theSame(item.Amount, exp2.Amount) {
		t.Errorf("Amount didn't match. Wanted: %v, got: %v.", exp2.Amount, item.Amount)
	}
	if !theSame(item.ACB, exp2.ACB) {
		t.Errorf("ACB didn't match. Wanted: %v, got: %v.", exp2.ACB, item.ACB)
	}
	if !theSame(item.Value, exp2.Value) {
		t.Errorf("Value didn't match. Wanted: %v, got: %v.", exp2.Value, item.Value)
	}
	if !theSame(item.Gain, exp2.Gain) {
		t.Errorf("Gain didn't match. Wanted: %v, got: %v.", exp2.Gain, item.Gain)
	}

}

func TestBuildACB(t *testing.T) {
	time := time.Now()
	r := &ACB{Currency: "CAD", AsOf: time}
	c, d := r.Build()
	if c != "CAD" {
		t.Errorf("Build did not set currency. Got: %v, want: %v", c, "CAD")
	}
	if d != time {
		t.Errorf("Build did not set currency. Got: %v, want: %v", d, time)
	}
}

func theSame(x, y decimal.Decimal) bool {
	th := decimal.NewFromFloat(0.000001)
	return x.Sub(y).Abs().LessThan(th)
}
