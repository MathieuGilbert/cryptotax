// Package exchange aids in getting historical fiat prices
// BTC/FIAT: Powered by <a href="https://www.coindesk.com/price/">CoinDesk</a>
// ALL: https://min-api.cryptocompare.com/data/dayAvg?fsym=ETH&tsym=CAD&toTs=1489536000&extraParams="cryptotax"
package exchange

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-redis/cache"
	"github.com/mathieugilbert/cryptotax/cmd/redis"
	"github.com/shopspring/decimal"
)

// Asset holds amount and currency units
type Asset struct {
	Amount   decimal.Decimal
	Currency string
}

// Conversion wraps a currency conversion
type Conversion struct {
	From *Asset
	To   *Asset
	Date time.Time
	Rate decimal.Decimal
}

// Convert the amount from one currency to the other
func (c *Conversion) Convert() error {
	if err := c.SetRate(); err != nil {
		return err
	}
	c.To.Amount = c.From.Amount.Mul(c.Rate)
	return nil
}

// SetRate on the conversion if not set
func (c *Conversion) SetRate() error {
	// no conversion needed
	if c.From.Currency == c.To.Currency {
		c.Rate = decimal.NewFromFloat(1)
	}

	if c.Rate.IsZero() {
		rate, err := FetchRate(c.From.Currency, c.To.Currency, c.Date)
		if err != nil {
			return err
		}
		c.Rate = rate
	}

	return nil
}

// FetchRate gets the exchange rate using Cryptocompare API, if not in cache
// Rate limits: 15/s, 300/min, 8000/hr
func FetchRate(from, to string, date time.Time) (rate decimal.Decimal, err error) {
	codec := redis.New()
	key := redis.Key(from, to, date)

	// look for rate in cache
	var rs string
	if err = codec.Get(key, &rs); err == nil {
		// found the key
		rate, err = decimal.NewFromString(rs)
		if err != nil {
			return
		}
		return
	}

	url := fmt.Sprintf(
		"https://min-api.cryptocompare.com/data/dayAvg?fsym=%v&tsym=%v&toTs=%v&extraParams=cryptotax",
		from,
		to,
		date.Unix(),
	)

	fmt.Printf("Calling API: %v\n", key)

	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	data := make(map[string]interface{})
	err = json.Unmarshal([]byte(body), &data)
	if err != nil {
		return
	}
	if data[to] == nil {
		err = fmt.Errorf("Couldn't find %v", from)
		return
	}
	rate = decimal.NewFromFloat(data[to].(float64))

	// cache the rate
	codec.Set(&cache.Item{
		Key:        key,
		Object:     rate.String(),
		Expiration: time.Hour,
	})

	return
}
