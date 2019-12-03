package ohlc

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Candles contains collated data for differing time periods
type Candles struct {
	Pair             currency.Pair
	Asset            asset.Item
	ExchangeName     string
	TimePeriod       time.Duration
	PercentageChange []float64
	Open             []float64
	Close            []float64
	High             []float64
	Low              []float64
	Volume           []float64
	Date             []int64
}

// HeartBeat denotes open and close times on a chart respective to time period
type HeartBeat struct {
	Open  time.Time
	Close time.Time
}

// CreateOHLC creates candles out of trade history data for a set time period
func CreateOHLC(h []exchange.TradeHistory, timePeriod time.Duration, p currency.Pair, a asset.Item, exchangeName string) (*Candles, error) {
	err := ValidatData(&h)
	if err != nil {
		return nil, err
	}

	timeIntervalStart := h[0].Timestamp.Truncate(timePeriod)
	timeIntervalEnd := h[len(h)-1].Timestamp

	// Adds time interval buffer zones
	var timeIntervalCache [][]exchange.TradeHistory
	var OpenClose []HeartBeat

	for t := timeIntervalStart; t.Before(timeIntervalEnd); t = t.Add(timePeriod) {
		timeBufferEnd := t.Add(timePeriod)
		insertionCount := 0

		var zonedTradeHistory []exchange.TradeHistory
		for i := 0; i < len(h); i++ {
			if (h[i].Timestamp.After(t) || h[i].Timestamp.Equal(t)) &&
				(h[i].Timestamp.Before(timeBufferEnd) ||
					h[i].Timestamp.Equal(timeBufferEnd)) {
				zonedTradeHistory = append(zonedTradeHistory, h[i])
				insertionCount++
				continue
			}
			h = h[i:]
			break
		}

		// Insert dummy in time period when there is no price action
		if insertionCount == 0 {
			OpenClose = append(OpenClose, HeartBeat{Open: t, Close: timeBufferEnd})
			timeIntervalCache = append(timeIntervalCache, []exchange.TradeHistory{})
			continue
		}
		OpenClose = append(OpenClose, HeartBeat{Open: t, Close: timeBufferEnd})
		timeIntervalCache = append(timeIntervalCache, zonedTradeHistory)
	}

	c := Candles{
		Pair:         p,
		Asset:        a,
		ExchangeName: exchangeName,
		TimePeriod:   timePeriod,
	}

	var closePriceOfLast float64
	for x := range timeIntervalCache {
		if len(timeIntervalCache[x]) == 0 {
			c.Date = append(c.Date, OpenClose[x].Open.Unix())
			c.High = append(c.High, closePriceOfLast)
			c.Low = append(c.Low, closePriceOfLast)
			c.Close = append(c.Close, closePriceOfLast)
			c.Open = append(c.Open, closePriceOfLast)
			c.Volume = append(c.Volume, 0)
			c.PercentageChange = append(c.PercentageChange, 0)
			continue
		}

		for y := range timeIntervalCache[x] {
			if y == 0 {
				c.Open = append(c.Open, timeIntervalCache[x][y].Price)
				c.Date = append(c.Date, OpenClose[x].Open.Unix())
				c.High = append(c.High, timeIntervalCache[x][y].Price)
				c.Low = append(c.Low, timeIntervalCache[x][y].Price)
				c.Volume = append(c.Volume, timeIntervalCache[x][y].Amount)
			} else {
				c.Volume[x] += timeIntervalCache[x][y].Amount
			}
			if y == len(timeIntervalCache[x])-1 {
				c.Close = append(c.Close, timeIntervalCache[x][y].Price)
				closePriceOfLast = timeIntervalCache[x][y].Price
			}
			if c.High[x] < timeIntervalCache[x][y].Price {
				c.High[x] = timeIntervalCache[x][y].Price
			}
			if c.Low[x] > timeIntervalCache[x][y].Price || c.Low[x] == 0 {
				c.Low[x] = timeIntervalCache[x][y].Price
			}
		}
		percentChange := ((c.Close[x] - c.Open[x]) / c.Open[x]) * 100
		c.PercentageChange = append(c.PercentageChange, percentChange)
	}
	return &c, nil
}

// ValidatData checks for zero values on data
func ValidatData(h *[]exchange.TradeHistory) error {
	wow := make([]exchange.TradeHistory, len(*h))
	copy(wow, *h)

	if len(wow) <= 1 {
		return errors.New("insufficient data to validate")
	}

	for i := range wow {
		if wow[i].Timestamp.IsZero() ||
			wow[i].Timestamp.Unix() == 0 {
			return fmt.Errorf("timestamp not set for element %d", i)
		}

		if wow[i].Amount == 0 {
			return fmt.Errorf("amount not set for element %d", i)
		}

		if wow[i].Price == 0 {
			return fmt.Errorf("price not set for element %d", i)
		}
	}

	sort.Slice(wow, func(i, j int) bool {
		return wow[i].Timestamp.Before(wow[j].Timestamp)
	})

	fmt.Println("MEOW:", wow)

	*h = wow
	return nil
}
