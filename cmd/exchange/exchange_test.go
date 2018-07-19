package exchange

import (
	"testing"
	"time"

	"github.com/mathieugilbert/cryptotax/cmd/redis"
	"github.com/shopspring/decimal"
)

func TestConvert1(t *testing.T) {
	// clear cache
	redis.New().Delete("ETHBTC1489536000")

	type TestCase struct {
		C *Conversion
		X *Conversion
	}
	tcs := []*TestCase{
		// sets rate to 1 from same from/to asset
		&TestCase{
			C: &Conversion{
				From: &Asset{Amount: decimal.NewFromFloat(2), Currency: "BTC"},
				To:   &Asset{Currency: "BTC"},
				Date: time.Date(2017, time.March, 15, 0, 0, 0, 0, time.UTC),
			},
			X: &Conversion{
				From: &Asset{Amount: decimal.NewFromFloat(2), Currency: "BTC"},
				To:   &Asset{Amount: decimal.NewFromFloat(2), Currency: "BTC"},
				Date: time.Date(2017, time.March, 15, 0, 0, 0, 0, time.UTC),
				Rate: decimal.NewFromFloat(1),
			},
		},
		// does a conversion, hitting the actual API
		&TestCase{
			C: &Conversion{
				From: &Asset{Amount: decimal.NewFromFloat(10), Currency: "ETH"},
				To:   &Asset{Currency: "BTC"},
				Date: time.Date(2017, time.March, 15, 0, 0, 0, 0, time.UTC),
			},
			X: &Conversion{
				From: &Asset{Amount: decimal.NewFromFloat(10), Currency: "ETH"},
				To:   &Asset{Amount: decimal.NewFromFloat(0.2533), Currency: "BTC"},
				Date: time.Date(2017, time.March, 15, 0, 0, 0, 0, time.UTC),
				Rate: decimal.NewFromFloat(0.02533),
			},
		},
		// same as previous, should hit the cache
		&TestCase{
			C: &Conversion{
				From: &Asset{Amount: decimal.NewFromFloat(10), Currency: "ETH"},
				To:   &Asset{Currency: "BTC"},
				Date: time.Date(2017, time.March, 15, 0, 0, 0, 0, time.UTC),
			},
			X: &Conversion{
				From: &Asset{Amount: decimal.NewFromFloat(10), Currency: "ETH"},
				To:   &Asset{Amount: decimal.NewFromFloat(0.2533), Currency: "BTC"},
				Date: time.Date(2017, time.March, 15, 0, 0, 0, 0, time.UTC),
				Rate: decimal.NewFromFloat(0.02533),
			},
		},
	}

	th := decimal.NewFromFloat(0.000001)

	for _, tc := range tcs {
		if err := tc.C.Convert(); err != nil {
			t.Error("Should not have an error")
		}
		if tc.C.To.Amount.Sub(tc.X.To.Amount).GreaterThan(th) {
			t.Errorf("Wrong amount. Expected: %v | Got: %v", tc.X.To.Amount, tc.C.To.Amount)
		}
		if tc.C.Rate.Sub(tc.X.Rate).GreaterThan(th) {
			t.Errorf("Wrong rate. Expected: %v | Got: %v", tc.X.Rate, tc.C.Rate)
		}
	}
}

func TestInvalidSymbol(t *testing.T) {
	c := &Conversion{
		From: &Asset{Amount: decimal.NewFromFloat(2), Currency: "AAAXXX"},
		To:   &Asset{Currency: "BTC"},
		Date: time.Date(2017, time.March, 15, 0, 0, 0, 0, time.UTC),
	}

	err := c.Convert()

	if err == nil {
		t.Error("There should be an error")
	}

	if err.Error() != "Couldn't find AAAXXX" {
		t.Errorf("Wrong kind of error: %v", err.Error())
	}
}
