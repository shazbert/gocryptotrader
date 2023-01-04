package consolidation

import (
	"errors"
	"fmt"
	"math"
	"time"

	gctmath "github.com/thrasher-corp/gocryptotrader/common/math"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	strategy "github.com/thrasher-corp/gocryptotrader/exchanges/strategy/common"
	"gonum.org/v1/gonum/stat"
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
	// PortfolioAtRisk defines in what percentage is allowed from the portfolio
	// a strategy position is allowed to utilize.
	PortfolioAtRisk float64

	// MaxLoss defines at what point the strategy will close the position.
	MaxLoss float64
	// Period defines the SMA period for the bollinger band indicator
	Period int64
	// StandardDeviation defines the upper and lower standard deviation
	StandardDeviation float64
}

type Strategy struct {
	Config *Configuration

	Scheduler  strategy.Scheduler
	TimeSeries kline.Item

	// ConsolidationTracking tracks periods of consolidation of price to
	// determine median time period for paramater adjustment.
	ConsolidationTracking []int
	ConsolidationPeriod   int

	BreakoutUpper Breakout
	BreakoutLower Breakout

	// Returns tracks the full returns off candle closes across the time series
	Returns []float64

	// Volatility tracks the volatility across time series data
	Volatility []float64

	// SignalCounter how many signals have been received.
	SignalCounter int64

	Open   *Position
	Closed []Position
}

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
	// mean := stat.Mean(returns, nil)
	mean := stat.Mean(returns, nil)

	result := 0.0
	for x := range returns {
		deviation := returns[x] - mean
		result += deviation * deviation
	}

	fmt.Println(result)
	// // Calculate the variance of the returns
	// variance := 0.0
	// for _, r := range returns {
	// 	variance += (r - mean) * (r - mean)
	// }
	// variance /= float64(len(returns) - 1)

	// // Calculate the standard deviation of the returns
	// stddev := math.Sqrt(variance)

	// Return the volatility as a percentage
	return math.Sqrt(result / float64(len(returns)))
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
	returns := make([]float64, len(prices)-1)

	// Iterate through the slice of prices, starting from the second element
	for i := 1; i < len(prices); i++ {
		// Calculate the return for the current price
		ret := gctmath.CalculatePercentageGainOrLoss(prices[i], prices[i-1])
		// Append the return to the slice of returns
		returns[i-1] = ret
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

// CheckOpenPosition checks current open position
func (s *Strategy) CheckOpenPosition(latestPrice float64) error {
	if s == nil {
		return strategy.ErrIsNil
	}

	if s.Open == nil {
		return nil
	}

	if s.Open.ExitPrice != 0 {
		return errors.New("position already closed")
	}

	gain := gctmath.CalculatePercentageGainOrLoss(latestPrice, s.Open.EntryPrice)
	if !s.Open.IsLong {
		gain *= -1
	}

	switch {
	case gain >= s.Config.RequiredReturn:
		fmt.Printf("ClOSING POSITION LONG:%v OPEN CURRENT PNL:%v IN PROFIT\n", s.Open.IsLong, gain)
	case gain <= s.Config.MaxLoss:
		fmt.Printf("ClOSING POSITION LONG:%v OPEN CURRENT PNL:%v IN LOSS\n", s.Open.IsLong, gain)
	default:
		fmt.Printf("POSITION LONG:%v OPEN CURRENT PNL:%v\n", s.Open.IsLong, gain)
		return nil
	}

	s.Open.ExitPrice = latestPrice
	s.Open.PNL = gain
	s.Open.ExitTime = time.Now()
	s.Closed = append(s.Closed, *s.Open)
	s.Open = nil

	return nil
}

func (s *Strategy) CheckHistoricPositions() error {
	if s == nil {
		return strategy.ErrIsNil
	}

	pnl := 0.0
	for x := range s.Closed {
		pnl += s.Closed[x].PNL
	}

	fmt.Printf("CURRENT STRATEGY PNL [%v] ORDER COUNT [%v]\n", pnl, len(s.Closed))
	return nil
}

// CheckConsolidation determines if the price is consolidating which then allows
// a future breakout to signal an order.
func (s *Strategy) CheckConsolidation(latestPrice, upperBand, lowerBand float64) error {
	if s == nil {
		return strategy.ErrIsNil
	}

	if latestPrice > lowerBand && latestPrice < upperBand {
		s.ConsolidationPeriod++
		fmt.Printf("PRICE CONSOLIDATION FOR %v PERIOD(s) HAS CONSOLIDATED %v TIME(s).\n",
			s.ConsolidationPeriod,
			len(s.ConsolidationTracking))
		return nil
	}

	if s.ConsolidationPeriod == 0 {
		fmt.Printf("PRICE NOT CONSOLIDATING, HAS CONSOLIDATED %v TIMES.\n",
			len(s.ConsolidationTracking))
		return nil
	}

	s.ConsolidationTracking = append(s.ConsolidationTracking, s.ConsolidationPeriod)
	s.ConsolidationPeriod = 0

	if s.Open != nil {
		fmt.Println("BREAKOUT OCCURRED POSITION ALREADY OPENED")
		return nil
	}

	s.Open = &Position{
		EntryPrice: latestPrice,
		Quantity:   positionSizeAllocator(s.Config.Funds, s.Config.PortfolioAtRisk, s.Config.RequiredReturn, s.Volatility[len(s.Volatility)-1]),
		EntryTime:  time.Now(),
		IsLong:     latestPrice > upperBand,
	}

	fmt.Printf("OPENING POSITION EXCHANGE:[%v] PAIR:[%v] ASSET:[%v] PRICE:[%v] AMOUNT:[%v] IS-LONG:[%v]\n",
		s.Config.Exchange.GetName(),
		s.Config.Pair,
		s.Config.Asset,
		s.Open.EntryPrice,
		s.Open.Quantity,
		s.Open.IsLong)
	return nil
}

// CheckBreakoutUpper checks and tracks price above upper band signal
func (s *Strategy) CheckBreakoutUpper(latestPrice, upperBand float64) error {
	if s == nil {
		return strategy.ErrIsNil
	}

	if s.Open == nil {
		return nil
	}

	if latestPrice < upperBand {
		ok, err := s.BreakoutUpper.Commit(latestPrice)
		if err != nil {
			return err
		}
		if ok {
			fmt.Printf("STRATEGY SIGNAL UPPER BAND BREAKOUT COMMITED RUN. POTENTIAL MAX PROFIT:[%v] @ [%d] MAX LOSS: [%v] @ [%d] MEDIAN PROFITS [%v] LOSS [%v].\n",
				s.BreakoutUpper.BranchProfit[len(s.BreakoutUpper.BranchProfit)-1],
				s.BreakoutUpper.BranchProfitPeriod[len(s.BreakoutUpper.BranchProfitPeriod)-1],
				s.BreakoutUpper.BranchLoss[len(s.BreakoutUpper.BranchLoss)-1],
				s.BreakoutUpper.BranchLossPeriod[len(s.BreakoutUpper.BranchLossPeriod)-1],
				s.BreakoutUpper.MedianProfit[len(s.BreakoutUpper.MedianProfit)-1],
				s.BreakoutUpper.MedianLoss[len(s.BreakoutUpper.MedianLoss)-1])
		}
		return nil
	}
	s.BreakoutUpper.Add(latestPrice)
	fmt.Printf("STRATEGY SIGNAL UPPER BAND BREAKOUT FOR %v PERIOD(s) HAS SIGNALLED %v TIMES.\n",
		len(s.BreakoutUpper.IndividualPrices),
		len(s.BreakoutUpper.AllPrices))
	return nil
}

// CheckBreakoutLower checks and tracks price below lower band signal
func (s *Strategy) CheckBreakoutLower(latestPrice, lowerBand float64) error {
	if s == nil {
		return strategy.ErrIsNil
	}

	if s.Open == nil {
		return nil
	}

	if latestPrice > lowerBand {
		ok, err := s.BreakoutLower.Commit(latestPrice)
		if err != nil {
			return err
		}
		if ok {
			fmt.Printf("STRATEGY SIGNAL LOWER BAND BREAKOUT COMMITED RUN. POTENTIAL MAX PROFIT:[%v] @ [%d] MAX LOSS: [%v] @ [%d] MEDIAN PROFITS [%v] LOSS [%v].\n",
				s.BreakoutLower.BranchProfit[len(s.BreakoutLower.BranchProfit)-1],
				s.BreakoutLower.BranchProfitPeriod[len(s.BreakoutLower.BranchProfitPeriod)-1],
				s.BreakoutLower.BranchLoss[len(s.BreakoutLower.BranchLoss)-1],
				s.BreakoutLower.BranchLossPeriod[len(s.BreakoutLower.BranchLossPeriod)-1],
				s.BreakoutLower.MedianProfit[len(s.BreakoutLower.MedianProfit)-1],
				s.BreakoutLower.MedianLoss[len(s.BreakoutLower.MedianLoss)-1])
		}
		return nil
	}

	s.BreakoutLower.Add(latestPrice)
	fmt.Printf("STRATEGY SIGNAL LOWER BAND BREAKOUT FOR %v PERIOD(s) HAS SIGNALLED %v TIMES.\n",
		len(s.BreakoutLower.IndividualPrices),
		len(s.BreakoutLower.AllPrices))
	return nil
}

// Breakout tracks breakout price for parameter optimization
type Breakout struct {
	// MedianProfit defines what is the current median profit
	MedianProfit []float64
	// BranchProfits determines from breakout the highest high in that set
	BranchProfit []float64
	// BranchProfitPeriod determines how long until highest high
	BranchProfitPeriod []int

	// MedianLoss defines what is the current median loss
	MedianLoss []float64
	// BranchLosses determines from breakout the lowest low in that set
	BranchLoss []float64
	// BranchLossPeriod determines how long until lowest low
	BranchLossPeriod []int

	// AllPrices define all the prices for the breakout periods
	AllPrices [][]float64
	// IndividualPrices define the current prices for this current breakout
	// period.
	IndividualPrices []float64
}

// Commit commits the indivual prices to all prices if there are any.
func (b *Breakout) Add(price float64) {
	b.IndividualPrices = append(b.IndividualPrices, price)
}

// Commit commits the indivual prices to all prices if there are any.
func (b *Breakout) Commit(price float64) (bool, error) { // TODO: WOW
	if len(b.IndividualPrices) == 0 {
		return false, nil
	}

	b.IndividualPrices = append(b.IndividualPrices, price)

	high, low := b.IndividualPrices[0], b.IndividualPrices[0]
	highTarget, lowTarget := 0, 0
	for x := range b.IndividualPrices {
		if high < b.IndividualPrices[x] {
			high = b.IndividualPrices[x]
			highTarget = x
		}

		if low > b.IndividualPrices[x] {
			low = b.IndividualPrices[x]
			lowTarget = x
		}
	}

	highGain := gctmath.CalculatePercentageGainOrLoss(high, b.IndividualPrices[0])
	b.BranchProfit = append(b.BranchProfit, highGain)
	b.BranchProfitPeriod = append(b.BranchProfitPeriod, highTarget)

	var filteredProfit []float64
	for x := range b.BranchProfit {
		if b.BranchProfit[x] != 0 {
			filteredProfit = append(filteredProfit, b.BranchProfit[x])
		}
	}
	highMean, _ := gctmath.ArithmeticMean(filteredProfit)
	b.MedianProfit = append(b.MedianProfit, highMean)

	lowGain := gctmath.CalculatePercentageGainOrLoss(low, b.IndividualPrices[0])
	b.BranchLoss = append(b.BranchLoss, lowGain)
	b.BranchLossPeriod = append(b.BranchLossPeriod, lowTarget)

	var filteredLoss []float64
	for x := range b.BranchLoss {
		if b.BranchLoss[x] != 0 {
			filteredLoss = append(filteredLoss, b.BranchLoss[x])
		}
	}
	LowMean, _ := gctmath.ArithmeticMean(filteredLoss)
	b.MedianLoss = append(b.MedianLoss, LowMean)

	b.AllPrices = append(b.AllPrices, b.IndividualPrices)
	b.IndividualPrices = nil
	return true, nil
}
