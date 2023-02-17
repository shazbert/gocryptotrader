package top2bottom2

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	// Name is the strategy name
	Name         = "top2bottom2"
	mfiPeriodKey = "mfi-period"
	mfiLowKey    = "mfi-low"
	mfiHighKey   = "mfi-high"
	description  = `This is an example strategy to highlight more complex strategy design. All signals are processed and then ranked. Only the top 2 and bottom 2 proceed further`
)

var (
	errStrategyOnlySupportsSimultaneousProcessing = errors.New("strategy only supports simultaneous processing")
	errStrategyCurrencyRequirements               = errors.New("top2bottom2 strategy requires at least 4 currencies")
)

// Strategy is an implementation of the Handler interface
type Strategy struct {
	base.Strategy
	mfiPeriod decimal.Decimal
	mfiLow    decimal.Decimal
	mfiHigh   decimal.Decimal
}

// Name returns the name of the strategy
func (s *Strategy) Name() string {
	return Name
}

// Description provides a nice overview of the strategy
// be it definition of terms or to highlight its purpose
func (s *Strategy) Description() string {
	return description
}

// OnSignal handles a data event and returns what action the strategy believes should occur
// however,this complex strategy cannot function on an individual basis
func (s *Strategy) OnSignal(_ data.Handler, _ funding.IFundingTransferer, _ portfolio.Handler) (signal.Event, error) {
	return nil, errStrategyOnlySupportsSimultaneousProcessing
}

// SupportsSimultaneousProcessing highlights whether the strategy can handle multiple currency calculation
// There is nothing actually stopping this strategy from considering multiple currencies at once
// but for demonstration purposes, this strategy does not
func (s *Strategy) SupportsSimultaneousProcessing() bool {
	return true
}

type mfiFundEvent struct {
	event signal.Event
	mfi   decimal.Decimal
	funds funding.IFundReader
}

// OnSimultaneousSignals analyses multiple data points simultaneously, allowing flexibility
// in allowing a strategy to only place an order for X currency if Y currency's price is Z
func (s *Strategy) OnSimultaneousSignals(d []data.Handler, f funding.IFundingTransferer, _ portfolio.Handler) ([]signal.Event, error) {
	if len(d) < 4 {
		return nil, errStrategyCurrencyRequirements
	}
	mfiFundEvents := make([]mfiFundEvent, 0, len(d))
	var resp []signal.Event
	for i := range d {
		if d == nil {
			return nil, common.ErrNilEvent
		}
		sig, err := s.NewSignal(d[i])
		if err != nil {
			return nil, err
		}
		latest, err := d[i].Latest()
		if err != nil {
			return nil, err
		}
		sig.SetPrice(latest.GetClosePrice())
		offset := latest.GetOffset()

		if offset <= s.mfiPeriod.IntPart() {
			sig.AppendReason("Not enough data for signal generation")
			sig.SetDirection(order.DoNothing)
			resp = append(resp, sig)
			continue
		}

		history, err := d[i].History()
		if err != nil {
			return nil, err
		}
		var (
			closeData  = make([]decimal.Decimal, len(history))
			volumeData = make([]decimal.Decimal, len(history))
			highData   = make([]decimal.Decimal, len(history))
			lowData    = make([]decimal.Decimal, len(history))
		)
		for i := range history {
			closeData[i] = history[i].GetClosePrice()
			volumeData[i] = history[i].GetVolume()
			highData[i] = history[i].GetHighPrice()
			lowData[i] = history[i].GetLowPrice()
		}
		var massagedCloseData, massagedVolumeData, massagedHighData, massagedLowData []float64
		massagedCloseData, err = s.massageMissingData(closeData, sig.GetTime())
		if err != nil {
			return nil, err
		}
		massagedVolumeData, err = s.massageMissingData(volumeData, sig.GetTime())
		if err != nil {
			return nil, err
		}
		massagedHighData, err = s.massageMissingData(highData, sig.GetTime())
		if err != nil {
			return nil, err
		}
		massagedLowData, err = s.massageMissingData(lowData, sig.GetTime())
		if err != nil {
			return nil, err
		}
		mfi := indicators.MFI(massagedHighData, massagedLowData, massagedCloseData, massagedVolumeData, int(s.mfiPeriod.IntPart()))
		latestMFI := decimal.NewFromFloat(mfi[len(mfi)-1])
		hasDataAtTime, err := d[i].HasDataAtTime(latest.GetTime())
		if err != nil {
			return nil, err
		}
		if !hasDataAtTime {
			sig.SetDirection(order.MissingData)
			sig.AppendReasonf("missing data at %v, cannot perform any actions. MFI %v", latest.GetTime(), latestMFI)
			resp = append(resp, sig)
			continue
		}

		sig.SetDirection(order.DoNothing)
		sig.AppendReasonf("MFI at %v", latestMFI)

		funds, err := f.GetFundingForEvent(sig)
		if err != nil {
			return nil, err
		}
		mfiFundEvents = append(mfiFundEvents, mfiFundEvent{
			event: sig,
			mfi:   latestMFI,
			funds: funds.FundReader(),
		})
	}

	return s.selectTopAndBottomPerformers(mfiFundEvents, resp)
}

func (s *Strategy) selectTopAndBottomPerformers(events []mfiFundEvent, resp []signal.Event) ([]signal.Event, error) {
	if len(events) == 0 {
		return resp, nil
	}
	sort.Slice(events, func(i int, j int) bool { return events[i].mfi.LessThan(events[j].mfi) })
	buyingOrSelling := false
	for i := range events {
		if i < 2 && events[i].mfi.GreaterThanOrEqual(s.mfiHigh) {
			events[i].event.SetDirection(order.Sell)
			buyingOrSelling = true
		} else if i >= 2 {
			break
		}
	}
	sort.Slice(events, func(i int, j int) bool { return events[i].mfi.GreaterThan(events[j].mfi) })
	for i := range events {
		if i < 2 && events[i].mfi.LessThanOrEqual(s.mfiLow) {
			events[i].event.SetDirection(order.Buy)
			buyingOrSelling = true
		} else if i >= 2 {
			break
		}
	}
	for i := range events {
		if buyingOrSelling && events[i].event.GetDirection() == order.DoNothing {
			events[i].event.AppendReason("MFI was not in the top or bottom two ranks")
		}
		resp = append(resp, events[i].event)
	}
	return resp, nil
}

// SetCustomSettings allows a user to modify the MFI limits in their config
func (s *Strategy) SetCustomSettings(customSettings map[string]interface{}) error {
	for k, v := range customSettings {
		switch k {
		case mfiHighKey:
			mfiHigh, ok := v.(float64)
			if !ok || mfiHigh <= 0 {
				return fmt.Errorf("%w provided mfi-high value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.mfiHigh = decimal.NewFromFloat(mfiHigh)
		case mfiLowKey:
			mfiLow, ok := v.(float64)
			if !ok || mfiLow <= 0 {
				return fmt.Errorf("%w provided mfi-low value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.mfiLow = decimal.NewFromFloat(mfiLow)
		case mfiPeriodKey:
			mfiPeriod, ok := v.(float64)
			if !ok || mfiPeriod <= 0 {
				return fmt.Errorf("%w provided mfi-period value could not be parsed: %v", base.ErrInvalidCustomSettings, v)
			}
			s.mfiPeriod = decimal.NewFromFloat(mfiPeriod)
		default:
			return fmt.Errorf("%w unrecognised custom setting key %v with value %v. Cannot apply", base.ErrInvalidCustomSettings, k, v)
		}
	}

	return nil
}

// SetDefaults sets the custom settings to their default values
func (s *Strategy) SetDefaults() {
	s.mfiHigh = decimal.NewFromInt(70)
	s.mfiLow = decimal.NewFromInt(30)
	s.mfiPeriod = decimal.NewFromInt(14)
}

// massageMissingData will replace missing data with the previous candle's data
// this will ensure that mfi can be calculated correctly
// the decision to handle missing data occurs at the strategy level, not all strategies
// may wish to modify data
func (s *Strategy) massageMissingData(data []decimal.Decimal, t time.Time) ([]float64, error) {
	resp := make([]float64, len(data))
	var missingDataStreak int64
	for i := range data {
		if data[i].IsZero() && i > int(s.mfiPeriod.IntPart()) {
			data[i] = data[i-1]
			missingDataStreak++
		} else {
			missingDataStreak = 0
		}
		if missingDataStreak >= s.mfiPeriod.IntPart() {
			return nil, fmt.Errorf("missing data exceeds mfi period length of %v at %s and will distort results. %w",
				s.mfiPeriod,
				t.Format(gctcommon.SimpleTimeFormat),
				base.ErrTooMuchBadData)
		}
		resp[i] = data[i].InexactFloat64()
	}
	return resp, nil
}
