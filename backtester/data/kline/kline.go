package kline

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var (
	// TODO: Shift to backtester common
	errExchangeNameUnset = errors.New("exchange name unset")
	errInvalidInterval   = errors.New("invalid interval")
	errNoCandleData      = errors.New("no candle data")
)

// DataFromKline is a struct which implements the data.Streamer interface
// It holds candle data for a specified range with helper functions
type DataFromKline struct {
	*data.Base
	Item        *gctkline.Item
	RangeHolder *gctkline.IntervalRangeHolder
}

// NewDataFromKline in time series and sets up the range holder and base events
// defined by that data.
func NewDataFromKline(timeSeries *gctkline.Item, start, end time.Time) (*DataFromKline, error) {
	if timeSeries == nil {
		return nil, fmt.Errorf("%w for %T", gctcommon.ErrNilPointer, timeSeries)
	}

	if timeSeries.Exchange == "" {
		return nil, fmt.Errorf("%w for %T", errExchangeNameUnset, timeSeries)
	}

	if timeSeries.Pair.IsEmpty() {
		return nil, fmt.Errorf("%w for %T", currency.ErrCurrencyPairEmpty, timeSeries)
	}

	if !timeSeries.Asset.IsValid() {
		return nil, fmt.Errorf("%w for %T", asset.ErrNotSupported, timeSeries)
	}

	if timeSeries.Interval <= 0 {
		return nil, fmt.Errorf("%w for %T", errInvalidInterval, timeSeries)
	}

	err := gctcommon.StartEndTimeCheck(start, end)
	if err != nil {
		return nil, err
	}

	if len(timeSeries.Candles) == 0 {
		return nil, fmt.Errorf("%w for %T", errNoCandleData, timeSeries)
	}

	rangeHolder, err := gctkline.CalculateCandleDateRanges(start, end, timeSeries.Interval, 0)
	if err != nil {
		return nil, err
	}

	// TODO: rangeholder to data check.

	events, err := getEventsFromKlines(timeSeries)
	if err != nil {
		return nil, err
	}

	dataBase := &data.Base{}
	err = dataBase.SetStream(events)
	if err != nil {
		return nil, err
	}

	return &DataFromKline{
		Base:        dataBase,
		Item:        timeSeries,
		RangeHolder: rangeHolder,
	}, nil
}

// HasDataAtTime verifies checks the underlying range data
// To determine whether there is any candle data present at the time provided
func (d *DataFromKline) HasDataAtTime(t time.Time) (bool, error) {
	if d == nil {
		return false, fmt.Errorf("%w for %T", gctcommon.ErrNilPointer, d)
	}

	isLive, err := d.Base.IsLive()
	if err != nil {
		return false, err
	}
	if isLive {
		var s []data.Event
		s, err = d.GetStream()
		if err != nil {
			return false, err
		}
		for i := range s {
			if s[i].GetTime().Equal(t) {
				return true, nil
			}
		}
		return false, nil
	}
	if d.RangeHolder == nil {
		return false, fmt.Errorf("%w RangeHolder", gctcommon.ErrNilPointer)
	}
	return d.RangeHolder.HasDataAtDate(t), nil
}

// getEventsFromKlines
func getEventsFromKlines(k *gctkline.Item) ([]data.Event, error) {
	if k == nil {
		return nil, fmt.Errorf("%w for %T", gctcommon.ErrNilPointer, k)
	}

	if len(k.Candles) == 0 {
		return nil, errNoCandleData
	}

	events := make([]data.Event, len(k.Candles))
	for i := range k.Candles {
		baseEvent, err := event.NewBaseFromKline(k, k.Candles[i].Time, int64(i+1))
		if err != nil {
			return nil, err
		}
		events[i] = &kline.Kline{
			Base:             baseEvent,
			Open:             decimal.NewFromFloat(k.Candles[i].Open),
			High:             decimal.NewFromFloat(k.Candles[i].High),
			Low:              decimal.NewFromFloat(k.Candles[i].Low),
			Close:            decimal.NewFromFloat(k.Candles[i].Close),
			Volume:           decimal.NewFromFloat(k.Candles[i].Volume),
			ValidationIssues: k.Candles[i].ValidationIssues,
		}
	}
	return events, nil
}

// AppendResults adds a candle item to the data stream and sorts it to ensure it is all in order
func (d *DataFromKline) AppendResults(ki *gctkline.Item) error {
	if d == nil {
		return fmt.Errorf("%w for %T", gctcommon.ErrNilPointer, d)
	}

	if ki == nil {
		return fmt.Errorf("%w kline item", gctcommon.ErrNilPointer)
	}
	err := d.Item.EqualSource(ki)
	if err != nil {
		return err
	}
	var gctCandles []gctkline.Candle
	stream, err := d.Base.GetStream()
	if err != nil {
		return err
	}

candleLoop:
	for x := range ki.Candles {
		for y := range stream {
			if stream[y].GetTime().Equal(ki.Candles[x].Time) {
				continue candleLoop
			}
		}
		gctCandles = append(gctCandles, ki.Candles[x])
	}
	if len(gctCandles) == 0 {
		return nil
	}

	cpyKi := *ki
	cpyKi.Candles = gctCandles
	events, err := getEventsFromKlines(&cpyKi)
	if err != nil {
		return err
	}

	err = d.AppendStream(events...)
	if err != nil {
		return err
	}

	// TODO: Should not have duplicates.
	d.Item.RemoveDuplicates()
	// TODO: Should already be aligned
	d.Item.SortCandlesByTimestamp(false)

	if d.RangeHolder != nil {
		// TODO: d.Item.Candles[0].Time
		d.RangeHolder, err = gctkline.CalculateCandleDateRanges(d.Item.Candles[0].Time,
			d.Item.Candles[len(d.Item.Candles)-1].Time.Add(d.Item.Interval.Duration()),
			d.Item.Interval,
			uint32(d.RangeHolder.Limit))
		if err != nil {
			return err
		}
		// offline data check when there is a known range
		// live data does not need this
		return d.RangeHolder.SetHasDataFromCandles(d.Item.Candles)
	}
	return nil
}

// StreamOpen returns all Open prices from the beginning until the current iteration
// TODO: Stream infers *all* data might change name to HistoryOpen etc?
func (d *DataFromKline) StreamOpen() ([]decimal.Decimal, error) {
	if d == nil {
		return nil, fmt.Errorf("%w for %T", gctcommon.ErrNilPointer, d)
	}

	s, err := d.History()
	if err != nil {
		return nil, err
	}

	ret := make([]decimal.Decimal, len(s))
	for x := range s {
		ret[x] = s[x].GetOpenPrice()
	}
	return ret, nil
}

// StreamHigh returns all High prices from the beginning until the current iteration
func (d *DataFromKline) StreamHigh() ([]decimal.Decimal, error) {
	if d == nil {
		return nil, fmt.Errorf("%w for %T", gctcommon.ErrNilPointer, d)
	}

	s, err := d.History()
	if err != nil {
		return nil, err
	}

	ret := make([]decimal.Decimal, len(s))
	for x := range s {
		ret[x] = s[x].GetHighPrice()
	}
	return ret, nil
}

// StreamLow returns all Low prices from the beginning until the current iteration
func (d *DataFromKline) StreamLow() ([]decimal.Decimal, error) {
	if d == nil {
		return nil, fmt.Errorf("%w for %T", gctcommon.ErrNilPointer, d)
	}

	s, err := d.History()
	if err != nil {
		return nil, err
	}

	ret := make([]decimal.Decimal, len(s))
	for x := range s {
		ret[x] = s[x].GetLowPrice()
	}
	return ret, nil
}

// StreamClose returns all Close prices from the beginning until the current iteration
func (d *DataFromKline) StreamClose() ([]decimal.Decimal, error) {
	if d == nil {
		return nil, fmt.Errorf("%w for %T", gctcommon.ErrNilPointer, d)
	}

	s, err := d.History()
	if err != nil {
		return nil, err
	}

	ret := make([]decimal.Decimal, len(s))
	for x := range s {
		ret[x] = s[x].GetClosePrice()
	}
	return ret, nil
}

// StreamVol returns all Volume prices from the beginning until the current iteration
func (d *DataFromKline) StreamVol() ([]decimal.Decimal, error) {
	if d == nil {
		return nil, fmt.Errorf("%w for %T", gctcommon.ErrNilPointer, d)
	}
	s, err := d.History()
	if err != nil {
		return nil, err
	}

	ret := make([]decimal.Decimal, len(s))
	for x := range s {
		ret[x] = s[x].GetVolume()
	}
	return ret, nil
}
