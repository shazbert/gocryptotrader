package consolidation

import (
	"context"
	"fmt"
	"time"

	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/common"
	gctmath "github.com/thrasher-corp/gocryptotrader/common/math"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	strategy "github.com/thrasher-corp/gocryptotrader/exchanges/strategy/common"
)

// OnSignal processing signals that have been generated by the strategy.
func (s *Strategy) OnSignal(ctx context.Context, sig interface{}) (_continue bool, err error) {
	if s == nil {
		return false, strategy.ErrIsNil
	}

	if sig == nil {
		return false, strategy.ErrNilSignal
	}

	if s.Config == nil {
		return false, strategy.ErrConfigIsNil
	}

	tn, ok := sig.(time.Time)
	if !ok {
		return false, fmt.Errorf("%w of type: %T", strategy.ErrUnhandledSignal, sig)
	}

	if tn.IsZero() {
		return false, errTimeNotSet
	}

	if tn.Location() != time.UTC {
		return false, errSignalRequiresUTCAlignment
	}

	end := tn.Truncate(s.Config.Interval.Duration())
	if !end.Equal(tn) {
		return false, errIntervalMisalignment
	}

	s.SignalCounter++

	if s.TimeSeries.Candles == nil {
		fmt.Println("INITIAL SIGNAL:", tn)
		start := end.Add(-s.Config.Interval.Duration() * time.Duration(s.Config.Lookback))
		s.TimeSeries, err = s.Config.Exchange.GetHistoricCandles(ctx, s.Config.Pair, s.Config.Asset, start, end, s.Config.Interval)
		if err != nil {
			return false, err
		}
		if len(s.TimeSeries.Candles) > s.Config.Lookback {
			fmt.Println("Inclusive end date for historical data:", len(s.TimeSeries.Candles))
			s.TimeSeries.Candles = s.TimeSeries.Candles[len(s.TimeSeries.Candles)-s.Config.Lookback:]
		}
	} else {
		fmt.Println("SIGNAL:", tn, "COUNT:", s.SignalCounter)
		start := s.TimeSeries.Candles[len(s.TimeSeries.Candles)-1].Time.Add(s.Config.Interval.Duration()).UTC()
		var newCandles kline.Item
		newCandles, err = s.Config.Exchange.GetHistoricCandles(ctx, s.Config.Pair, s.Config.Asset, start, end, s.Config.Interval)
		if err != nil {
			return false, err
		}

		if len(newCandles.Candles) > 1 {
			s.TimeSeries.Candles = append(s.TimeSeries.Candles, newCandles.Candles[:1]...)
		} else {
			s.TimeSeries.Candles = append(s.TimeSeries.Candles, newCandles.Candles...)
		}
	}

	latestClose := s.TimeSeries.Candles[len(s.TimeSeries.Candles)-1].Close
	fmt.Println("LAST CLOSE:", latestClose)

	if len(s.Returns) == 0 {
		s.Returns = GetReturnsFromKlineData(s.TimeSeries, s.Config.Lookback)
	} else {
		previousClose := s.TimeSeries.Candles[len(s.TimeSeries.Candles)-2].Close
		gain := gctmath.CalculatePercentageGainOrLoss(latestClose, previousClose)
		s.Returns = append(s.Returns, gain)
	}

	fmt.Println("RETURN:", s.Returns[len(s.Returns)-1])

	s.Volatility = append(s.Volatility, volatility(s.Returns))
	fmt.Println("VOLATILITY:", s.Volatility[len(s.Volatility)-1])

	err = s.CheckOpenPosition(latestClose)
	if err != nil {
		return false, err
	}

	err = s.CheckHistoricPositions()
	if err != nil {
		return false, err
	}

	// restrict to lookback?
	bol, err := s.TimeSeries.GetBollingerBands(s.Config.Period,
		s.Config.StandardDeviation,
		s.Config.StandardDeviation,
		indicators.Sma)
	if err != nil {
		return false, err
	}

	upperBand := bol.Upper[len(bol.Upper)-1]
	lowerBand := bol.Lower[len(bol.Lower)-1]

	fmt.Printf("BB SIGNAL - UPPERBAND: [%v] MA: [%v] LOWERBAND:[%v] PERIOD: [%v] STDDEV: [%v]\n",
		upperBand,
		bol.Middle[len(bol.Middle)-1],
		lowerBand,
		s.Config.Period,
		s.Config.StandardDeviation)

	err = s.CheckConsolidation(latestClose, upperBand, lowerBand)
	if err != nil {
		return false, err
	}

	err = s.CheckBreakoutUpper(latestClose, upperBand)
	if err != nil {
		return false, err
	}

	err = s.CheckBreakoutLower(latestClose, lowerBand)
	if err != nil {
		return false, err
	}

	fmt.Println()
	return false, nil
}

// GetDescription returns the strategy description
func (s *Strategy) GetDescription() strategy.Descriptor {
	if s == nil {
		return nil
	}

	sched := s.Scheduler.GetSchedule()
	untilStart := "immediately"
	if until := time.Until(sched.Next); until > 0 {
		untilStart = until.String()
	}

	sinceStart := "not yet started"
	if since := time.Since(sched.Start); since > 0 {
		sinceStart = since.String()
	}

	return &Description{
		Exchange:           s.Config.Exchange.GetName(),
		Pair:               s.Config.Pair,
		Asset:              s.Config.Asset,
		Start:              sched.Start.UTC().Format(common.SimpleTimeFormat),
		End:                sched.End.UTC().Format(common.SimpleTimeFormat),
		UntilStart:         untilStart,
		SinceStart:         sinceStart,
		DeploymentInterval: s.Config.Interval,
		OperatingWindow:    sched.Window.String(),
		Simulation:         s.Config.Simulate,
	}
}

// Description defines the full operating description of the strategy with its
// configuration parameters.
type Description struct {
	Exchange           string         `json:"exchange"`
	Pair               currency.Pair  `json:"pair"`
	Asset              asset.Item     `json:"asset"`
	Start              string         `json:"start"`
	End                string         `json:"end"`
	UntilStart         string         `json:"untilStart"`
	SinceStart         string         `json:"sinceStart"`
	Aligned            bool           `json:"aligned"`
	DeploymentInterval kline.Interval `json:"deploymentInterval"`
	OperatingWindow    string         `json:"operatingWindow"`
	Overtrade          bool           `json:"overtrade"`
	Simulation         bool           `json:"simulation"`
}

// String implements stringer interface for a short description
func (d *Description) String() string {
	if d == nil {
		return ""
	}

	sim := "[STRATEGY IS LIVE]"
	if d.Simulation {
		sim = "[STRATEGY IS IN SIMULATION]"
	}

	return sim
}