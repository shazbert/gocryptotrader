package consolidation

import (
	"errors"
	"math"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	strategy "github.com/thrasher-corp/gocryptotrader/exchanges/strategy/common"
)

var errSignalRequiresUTCAlignment = errors.New("strategy requires utc time alignment")
var errTimeNotSet = errors.New("time not set")
var errIntervalMisalignment = errors.New("interval misalignment")

type Configuration struct {
	Exchange exchange.IBotExchange
	Pair     currency.Pair
	Asset    asset.Item
	Lookback int
	Simulate bool
	Interval kline.Interval
	// Funds define the strategy allocated funds.
	Funds float64
	// RequiredReturn defines at what point the strategy will close the position
	// at a profit.
	RequiredReturn float64
	// MaxLoss defines at what point the strategy will close the position.
	MaxLoss float64
}

type Strategy struct {
	Config *Configuration

	Scheduler  strategy.Scheduler
	TimeSeries kline.Item

	// ConsolidationTracking tracks periods of consolidation of price to
	// determine median time period for paramater adjustment.
	ConsolidationTracking []int
	ConsolidationPeriod   int

	// BreakoutUpTracking tracks periods of break out of price to
	// determine median time period for paramater adjustment.
	BreakoutUpTracking []int
	BreakoutUpPeriod   int

	// BreakoutDownTracking tracks periods of break out of price to
	// determine median time period for paramater adjustment.
	BreakoutDownTracking []int
	BreakoutDownPeriod   int

	Open   *Position
	Closed []Position
}

var portfolioRisk = 0.05 // Max strategy risk value

// Deployment defines an amount and its corresponding currency code.
type Deployment struct {
	Amount   float64       `json:"amount"`
	Currency currency.Code `json:"currency"`
}

// positionSizeAllocator calculates the position size based on the available
// capital, risk tolerance, and risk/reward ratio.
// It takes in the following parameters:
// - capital: the available capital for the trade or investment
// - risk: the percentage of capital that you are willing to risk on the trade
// or investment
//   - reward: the potential gain from the trade or investment, as a percentage of
//     the capital at risk
//   - volatility: the volatility of the asset, as a percentage
//
// It returns the position size as a float64.
func positionSizeAllocator(capital, risk, reward, volatility float64) float64 {
	// Calculate the risk/reward ratio
	r := reward / risk

	// Calculate the position size using the formula:
	// position size = (capital * risk) / (volatility * r)
	positionSize := (capital * risk) / (volatility * r)

	return positionSize
}

// volatility calculates the volatility of an asset based on its historical returns.
// It takes in the following parameters:
// - returns: a slice of floats representing the asset's historical returns
// It returns the volatility as a float64.
func volatility(returns []float64) float64 {
	// Calculate the mean of the returns
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	// Calculate the variance of the returns
	variance := 0.0
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns) - 1)

	// Calculate the standard deviation of the returns
	stddev := math.Sqrt(variance)

	// Return the volatility as a percentage
	return stddev * 100
}

// GetReturnsFromKlineData gets the returns from the lookback period
func GetReturnsFromKlineData(k kline.Item, lookbackPeriod int) []float64 {
	var prices []float64

	if len(k.Candles) <= lookbackPeriod {
		for x := range k.Candles {
			prices = append(prices, k.Candles[x].Close)
		}
	} else {
		for x := len(k.Candles) - lookbackPeriod; x < len(k.Candles); x++ {
			prices = append(prices, k.Candles[x].Close)
		}
	}

	return getReturns(prices)
}

func getReturns(prices []float64) []float64 {
	// Create a slice to store the returns
	var returns []float64

	// Iterate through the slice of prices, starting from the second element
	for i := 1; i < len(prices); i++ {
		// Calculate the return for the current price
		ret := prices[i]/prices[i-1] - 1

		// Append the return to the slice of returns
		returns = append(returns, ret)
	}
	return returns
}

func getZScore(actualValue float64, values []float64) float64 {
	// Calculate the mean of the values
	var mean float64
	for _, value := range values {
		mean += value
	}
	mean /= float64(len(values))

	// Calculate the standard deviation of the values
	var stdDev float64
	for _, value := range values {
		stdDev += math.Pow(value-mean, 2)
	}
	stdDev = math.Sqrt(stdDev / float64(len(values)))

	// Calculate the z-score of the value
	zScore := (actualValue - mean) / stdDev
	return zScore
}

// Define a struct to represent a position
type Position struct {
	EntryPrice float64   // Price at which the position was entered
	ExitPrice  float64   // Price at which the position was exited (0 if still open)
	PNL        float64   // Profit or loss for the position
	Quantity   float64   // Number of units in the position
	EntryTime  time.Time // Time at which the position was entered
	ExitTime   time.Time // Time at which the position was exited (zero value if still open)
	IsLong     bool      // Flag to indicate if the position is a long (buy) or short (sell)
}

// // Create a position
// position := Position{
// 	EntryPrice:   100.0,
// 	ExitPrice:    0.0,
// 	PNL:          0.0,
// 	Quantity:     1.0,
// 	EntryTime:    time.Now(),
// 	ExitTime:     time.Time{},
// 	IsLong:       true,
// }

// // Update the position when it is closed
// position.ExitPrice = 105.0
// position.ExitTime = time.Now()
// position.PNL = (position.ExitPrice - position.EntryPrice

type BacktestKline kline.Item

// GetRange returns the candles between a start and end time, inclusive refers
// to the end time and that allows that to be included in the candle set. e.g.
// 10:00 UTC 10:30 UTC for minute klines will return 30 candles with the last
// candle being 10:29 UTC open. Whereas inclusive true will return 31 candles
// with the last candle being 10:30 UTC open.
func (b *BacktestKline) GetRange(start, end time.Time, inclusive bool) (kline.Item, error) {
	if start.IsZero() || end.IsZero() {
		return kline.Item{}, errTimeNotSet
	}

	window := end.Sub(start)
	length := int(window / b.Interval.Duration())
	if inclusive {
		length++
	}

	for x := range b.Candles {
		if b.Candles[x].Time.After(start) {
			break
		}

		if !b.Candles[x].Time.Equal(start) {
			continue
		}

		if len(b.Candles[x:]) < length {
			return kline.Item{}, errors.New("length bro exceed stored candle data")
		}

		outbound := *b
		outbound.Candles = b.Candles[x : x+length]
		return kline.Item(outbound), nil
	}

	return kline.Item{}, errors.New("candle start not found")
}
