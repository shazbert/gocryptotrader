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
	errNoCandleData      = errors.New("no candle data")
)

// DataFromKline is a struct which implements the data.Streamer interface
// It holds candle data for a specified range with helper functions.
type DataFromKline struct {
	*data.Base
	Item        *gctkline.Item
	RangeHolder *gctkline.IntervalRangeHolder
}

// NewDataFromKline takes in time series and sets up the range holder and base
// events defined by that data.
func NewDataFromKline(ki *gctkline.Item, start, end time.Time) (*DataFromKline, error) {
	if ki == nil {
		return nil, fmt.Errorf("%w for %T", gctcommon.ErrNilPointer, ki)
	}

	if ki.Exchange == "" {
		return nil, fmt.Errorf("%w for %T", errExchangeNameUnset, ki)
	}

	if ki.Pair.IsEmpty() {
		return nil, fmt.Errorf("%w for %T", currency.ErrCurrencyPairEmpty, ki)
	}

	if !ki.Asset.IsValid() {
		return nil, fmt.Errorf("%w for %T", asset.ErrNotSupported, ki)
	}

	rangeHolder, err := gctkline.CalculateCandleDateRanges(start, end, ki.Interval, 0)
	if err != nil {
		return nil, err
	}

	if len(ki.Candles) == 0 {
		return nil, fmt.Errorf("%w for %T", errNoCandleData, ki)
	}

	err = rangeHolder.SetHasDataFromCandles(ki.Candles)
	if err != nil {
		return nil, err
	}

	events, err := getEventsFromKlines(ki)
	if err != nil {
		return nil, err
	}

	// TODO: NewDataBase function and only append stream.
	dataBase := &data.Base{}
	err = dataBase.SetStream(events)
	if err != nil {
		return nil, err
	}

	return &DataFromKline{
		Base:        dataBase,
		Item:        ki,
		RangeHolder: rangeHolder,
	}, nil
}

// HasDataAtTime verifies checks the underlying range data
// To determine whether there is any candle data present at the time provided
func (d *DataFromKline) HasDataAtTime(t time.Time) (bool, error) {
	err := d.validate()
	if err != nil {
		return false, err
	}

	isLive, err := d.Base.IsLive()
	if err != nil {
		return false, err
	}

	if !isLive {
		return d.RangeHolder.HasDataAtDate(t)
	}

	stream, err := d.GetStream()
	if err != nil {
		return false, err
	}

	for i := range stream {
		if stream[i].GetTime().Equal(t) {
			return true, nil
		}
	}
	return false, nil
}

// getEventsFromKlines returns data events from gct candles
func getEventsFromKlines(k *gctkline.Item) ([]data.Event, error) {
	if k == nil {
		return nil, fmt.Errorf("cannot get data events from gct klines: %w for %T", gctcommon.ErrNilPointer, k)
	}

	if len(k.Candles) == 0 {
		return nil, fmt.Errorf("cannot get data events from gct klines: %w", errNoCandleData)
	}

	events := make([]data.Event, len(k.Candles))
	for i := range k.Candles {
		baseEvent, err := event.NewBaseFromKline(k, k.Candles[i].Time, int64(i+1))
		if err != nil {
			return nil, fmt.Errorf("cannot get data events from gct klines: %w", err)
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

// validate checks to see if the DataFromKline struct is setup correctly
func (d *DataFromKline) validate() error {
	if d == nil {
		return fmt.Errorf("%w for %T", gctcommon.ErrNilPointer, d)
	}

	if d.Base == nil {
		return fmt.Errorf("%w for %T", gctcommon.ErrNilPointer, d.Base)
	}

	if d.Item == nil {
		return fmt.Errorf("%w for %T", gctcommon.ErrNilPointer, d.Item)
	}

	if d.RangeHolder == nil {
		return fmt.Errorf("%w for %T", gctcommon.ErrNilPointer, d.RangeHolder)
	}

	return nil
}

// AppendResults adds a candle item to the data stream and sorts it to ensure it is all in order
func (d *DataFromKline) AppendResults(ki *gctkline.Item) error {
	err := d.validate()
	if err != nil {
		return err
	}

	if ki == nil {
		return fmt.Errorf("%w for %T", gctcommon.ErrNilPointer, ki)
	}

	err = d.Item.EqualSource(ki)
	if err != nil {
		return err
	}

	if len(ki.Candles) == 0 {
		return errNoCandleData
	}

	stream, err := d.Base.GetStream()
	if err != nil {
		return err
	}

	candles := make([]gctkline.Candle, len(ki.Candles))
	copy(candles, ki.Candles)
	if len(stream) > 0 {
		// NOTE: At this point both candles stream/incoming **should** be
		// correctly aligned as ascending.
		boundary := stream[len(stream)-1].GetTime().Add(ki.Interval.Duration())
		target := 0
		for x := range ki.Candles {
			if !ki.Candles[x].Time.Before(boundary) {
				break
			}
			// TODO: Reject if candle is found in original list. We should
			// only be fetching data based of last + interval.Duration().
			target++
		}
		candles = candles[target:]
	}

	if len(candles) == 0 {
		return nil
	}

	cpyKi := *ki
	cpyKi.Candles = candles

	events, err := getEventsFromKlines(&cpyKi)
	if err != nil {
		return err
	}

	// NOTE: This will check for duplicates.
	err = d.AppendStream(events...)
	if err != nil {
		return err
	}

	// TODO: Reject on invalid time alignment, probably in append stream.
	d.Item.SortCandlesByTimestamp(false)

	start := d.Item.Candles[0].Time
	end := d.Item.Candles[len(d.Item.Candles)-1].Time.Add(d.Item.Interval.Duration())
	d.RangeHolder, err = gctkline.CalculateCandleDateRanges(start, end, d.Item.Interval, uint32(d.RangeHolder.Limit))
	if err != nil {
		return err
	}

	// Offline data check when there is a known range live data does not need
	// this.
	return d.RangeHolder.SetHasDataFromCandles(d.Item.Candles)
}

// StreamOpen returns all Open prices from the beginning until the current
// iteration.
func (d *DataFromKline) StreamOpen() ([]decimal.Decimal, error) {
	err := d.validate()
	if err != nil {
		return nil, err
	}

	historicEvents, err := d.History()
	if err != nil {
		return nil, err
	}

	ret := make([]decimal.Decimal, len(historicEvents))
	for x := range historicEvents {
		ret[x] = historicEvents[x].GetOpenPrice()
	}
	return ret, nil
}

// StreamHigh returns all High prices from the beginning until the current
// iteration.
func (d *DataFromKline) StreamHigh() ([]decimal.Decimal, error) {
	err := d.validate()
	if err != nil {
		return nil, err
	}

	historicEvents, err := d.History()
	if err != nil {
		return nil, err
	}

	ret := make([]decimal.Decimal, len(historicEvents))
	for x := range historicEvents {
		ret[x] = historicEvents[x].GetHighPrice()
	}
	return ret, nil
}

// StreamLow returns all Low prices from the beginning until the current
// iteration.
func (d *DataFromKline) StreamLow() ([]decimal.Decimal, error) {
	err := d.validate()
	if err != nil {
		return nil, err
	}

	historicEvents, err := d.History()
	if err != nil {
		return nil, err
	}

	ret := make([]decimal.Decimal, len(historicEvents))
	for x := range historicEvents {
		ret[x] = historicEvents[x].GetLowPrice()
	}
	return ret, nil
}

// StreamClose returns all Close prices from the beginning until the current
// iteration.
func (d *DataFromKline) StreamClose() ([]decimal.Decimal, error) {
	err := d.validate()
	if err != nil {
		return nil, err
	}

	historicEvents, err := d.History()
	if err != nil {
		return nil, err
	}

	ret := make([]decimal.Decimal, len(historicEvents))
	for x := range historicEvents {
		ret[x] = historicEvents[x].GetClosePrice()
	}
	return ret, nil
}

// StreamVol returns all Volume prices from the beginning until the current
// iteration.
func (d *DataFromKline) StreamVol() ([]decimal.Decimal, error) {
	err := d.validate()
	if err != nil {
		return nil, err
	}

	historicEvents, err := d.History()
	if err != nil {
		return nil, err
	}

	ret := make([]decimal.Decimal, len(historicEvents))
	for x := range historicEvents {
		ret[x] = historicEvents[x].GetVolume()
	}
	return ret, nil
}
