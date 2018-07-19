package acb

import (
	"testing"
	"time"

	"github.com/go-redis/cache"
	"github.com/mathieugilbert/cryptotax/cmd/redis"
	"github.com/mathieugilbert/cryptotax/models"
	"github.com/shopspring/decimal"
)

func TestSortAssetDate(t *testing.T) {
	ts := []*models.Trade{
		&models.Trade{ID: 1, Asset: "BBB", Date: time.Now().AddDate(0, -5, 0)},
		&models.Trade{ID: 2, Asset: "AAA", Date: time.Now().AddDate(0, -5, 0)},
		&models.Trade{ID: 3, Asset: "BBB", Date: time.Now().AddDate(0, -3, 0)},
		&models.Trade{ID: 4, Asset: "AAA", Date: time.Now().AddDate(0, -3, 0)},
		&models.Trade{ID: 5, Asset: "CCC", Date: time.Now().AddDate(0, -3, 0)},
		&models.Trade{ID: 6, Asset: "BBB", Date: time.Now().AddDate(0, -7, 0)},
		&models.Trade{ID: 7, Asset: "AAA", Date: time.Now().AddDate(0, -7, 0)},
	}

	SortAssetDate(ts)

	expect := [7]uint{7, 2, 4, 6, 1, 3, 5}
	actual := [7]uint{}
	for i, t := range ts {
		actual[i] = t.ID
	}

	if expect != actual {
		t.Errorf("Sorted order is wrong. Got: %v, want: %v", actual, expect)
	}
}

func TestCalculate(t *testing.T) {
	ts := []*models.Trade{
		&models.Trade{
			Asset:        "AAA",
			Action:       "buy",
			Quantity:     decimal.NewFromFloat(10),
			BasePrice:    decimal.NewFromFloat(2000),
			BaseFee:      decimal.NewFromFloat(20),
			BaseCurrency: "CAD",
			Date:         time.Now().AddDate(0, 0, 1),
		},
		&models.Trade{
			Asset:        "AAA",
			Action:       "buy",
			Quantity:     decimal.NewFromFloat(40),
			BasePrice:    decimal.NewFromFloat(3000),
			BaseFee:      decimal.NewFromFloat(5),
			BaseCurrency: "CAD",
			Date:         time.Now().AddDate(0, 0, 2),
		},
		&models.Trade{
			Asset:        "AAA",
			Action:       "sell",
			Quantity:     decimal.NewFromFloat(20),
			BasePrice:    decimal.NewFromFloat(5000),
			BaseFee:      decimal.NewFromFloat(10),
			BaseCurrency: "CAD",
			Date:         time.Now().AddDate(0, 0, 3),
		},
		&models.Trade{
			Asset:        "AAA",
			Action:       "buy",
			Quantity:     decimal.NewFromFloat(30),
			BasePrice:    decimal.NewFromFloat(6000),
			BaseFee:      decimal.NewFromFloat(20),
			BaseCurrency: "CAD",
			Date:         time.Now().AddDate(0, 0, 4),
		},
	}

	c, err := Calculate(ts, "CAD")

	if err != nil {
		t.Error("There should not be an error")
	}

	// trade 1
	i := 0
	if exp, act := decimal.NewFromFloat(2020), c[i].CostBase; !act.Equal(exp) {
		t.Errorf("CostBase[%v] is wrong. Got: %v, want: %v", i, c[i].CostBase, exp)
	}
	if exp, act := decimal.NewFromFloat(10), c[i].CoinBalance; !act.Equal(exp) {
		t.Errorf("CoinBalance[%v] is wrong. Got: %v, want: %v", i, act, exp)
	}
	// trade 2
	i = 1
	if exp, act := decimal.NewFromFloat(5025), c[i].CostBase; !act.Equal(exp) {
		t.Errorf("CostBase[%v] is wrong. Got: %v, want: %v", i, act, exp)
	}
	if exp, act := decimal.NewFromFloat(50), c[i].CoinBalance; !act.Equal(exp) {
		t.Errorf("CoinBalance[%v] is wrong. Got: %v, want: %v", i, act, exp)
	}
	// trade 3
	i = 2
	if exp, act := decimal.NewFromFloat(3015), c[i].CostBase; !act.Equal(exp) {
		t.Errorf("CostBase[%v] is wrong. Got: %v, want: %v", i, act, exp)
	}
	if exp, act := decimal.NewFromFloat(30), c[i].CoinBalance; !act.Equal(exp) {
		t.Errorf("CoinBalance[%v] is wrong. Got: %v, want: %v", i, act, exp)
	}
	if exp, act := decimal.NewFromFloat(5000), c[i].Proceeds; !act.Equal(exp) {
		t.Errorf("Proceeds[%v] is wrong. Got: %v, want: %v", i, act, exp)
	}
	if exp, act := decimal.NewFromFloat(10), c[i].DispositionExpenses; !act.Equal(exp) {
		t.Errorf("DispositionExpenses[%v] is wrong. Got: %v, want: %v", i, act, exp)
	}
	if exp, act := decimal.NewFromFloat(4990), c[i].NetIncome; !act.Equal(exp) {
		t.Errorf("NetIncome[%v] is wrong. Got: %v, want: %v", i, act, exp)
	}
	// trade 4
	i = 3
	if exp, act := decimal.NewFromFloat(9035), c[i].CostBase; !act.Equal(exp) {
		t.Errorf("CostBase[%v] is wrong. Got: %v, want: %v", i, c[i].CostBase, exp)
	}
	if exp, act := decimal.NewFromFloat(60), c[i].CoinBalance; !act.Equal(exp) {
		t.Errorf("CoinBalance[%v] is wrong. Got: %v, want: %v", i, act, exp)
	}
}

func TestToBaseCurrency(t *testing.T) {
	ts := []*models.Trade{
		&models.Trade{
			Asset:        "ETH",
			Action:       "buy",
			Quantity:     decimal.NewFromFloat(10),
			BasePrice:    decimal.NewFromFloat(2000),
			BaseFee:      decimal.NewFromFloat(20),
			BaseCurrency: "CAD",
			Date:         time.Now().AddDate(0, 0, 1),
		},
		&models.Trade{
			Asset:        "BTC",
			Action:       "buy",
			Quantity:     decimal.NewFromFloat(20),
			BasePrice:    decimal.NewFromFloat(1000),
			BaseFee:      decimal.NewFromFloat(5),
			BaseCurrency: "CAD",
			Date:         time.Now().AddDate(0, 0, 2),
		},
		&models.Trade{
			Asset:        "ETH",
			Action:       "sell",
			Quantity:     decimal.NewFromFloat(5),
			BasePrice:    decimal.NewFromFloat(8),
			BaseFee:      decimal.NewFromFloat(0.1),
			BaseCurrency: "BTC",
			Date:         time.Now().AddDate(0, 0, 3),
		},
		&models.Trade{
			Asset:        "ETH",
			Action:       "buy",
			Quantity:     decimal.NewFromFloat(5),
			BasePrice:    decimal.NewFromFloat(8),
			BaseFee:      decimal.NewFromFloat(0.1),
			BaseCurrency: "BTC",
			Date:         time.Now().AddDate(0, 0, 4),
		},
	}

	// write to redis cache to prevent API call
	codec := redis.New()
	for _, t := range ts {
		if t.BaseCurrency != "CAD" {
			codec.Set(&cache.Item{
				Key:        redis.Key(t.BaseCurrency, "CAD", t.Date),
				Object:     "1.0",
				Expiration: time.Hour,
			})
		}
	}
	ts, err := ToBaseCurrency(ts, "CAD")

	if err != nil {
		t.Error("There should not be an error")
	}

	if len(ts) != 6 {
		t.Errorf("Cross trades should add extra entries. Expected: %v, Got: %v", 6, len(ts))
	}

	pass := true
	for _, t := range ts {
		pass = pass && (t.BaseCurrency == "CAD")
	}
	if !pass {
		t.Error("All trades should have same base currency")
	}
}

func TestOversoldSingle(t *testing.T) {
	ts := []*models.Trade{
		&models.Trade{
			Asset:        "AAA",
			Action:       "Sell",
			Quantity:     decimal.NewFromFloat(20),
			BasePrice:    decimal.NewFromFloat(5000),
			BaseFee:      decimal.NewFromFloat(10),
			BaseCurrency: "CAD",
		},
	}
	_, err := Calculate(ts, "CAD")

	if err == nil {
		t.Errorf("Should return an error")
	}

	switch e := err.(type) {
	case *Oversold:
		expect := "Missing previous buys for: AAA (20)"
		if e.Error() != expect {
			t.Errorf("Error message is wrong. Got: %v, Want: %v", e.Error(), expect)
		}
	default:
		t.Errorf("Should return Oversold error")
	}
}

func TestOversoldMultiple(t *testing.T) {
	ts := []*models.Trade{
		&models.Trade{
			Asset:        "AAA",
			Action:       "Sell",
			Quantity:     decimal.NewFromFloat(20),
			BasePrice:    decimal.NewFromFloat(5000),
			BaseFee:      decimal.NewFromFloat(10),
			BaseCurrency: "CAD",
		},
		&models.Trade{
			Asset:        "BBB",
			Action:       "Sell",
			Quantity:     decimal.NewFromFloat(12.345),
			BasePrice:    decimal.NewFromFloat(600),
			BaseFee:      decimal.NewFromFloat(4),
			BaseCurrency: "CAD",
		},
	}
	_, err := Calculate(ts, "CAD")

	if err == nil {
		t.Errorf("Should return an error")
	}

	switch e := err.(type) {
	case *Oversold:
		expect := "Missing previous buys for: AAA (20), BBB (12.345)"
		if e.Error() != expect {
			t.Errorf("Error message is wrong. Got: %v, Want: %v", e.Error(), expect)
		}
	default:
		t.Errorf("Should return Oversold error")
	}
}
