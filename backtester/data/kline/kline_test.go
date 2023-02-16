package kline

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const testExchange = "binance"

var pair = currency.NewPair(currency.BTC, currency.USDT)

func TestNewDataFromKline(t *testing.T) {
	t.Parallel()
	_, err := NewDataFromKline(nil, time.Time{}, time.Time{})
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Fatalf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	dummy := &gctkline.Item{}
	_, err = NewDataFromKline(dummy, time.Time{}, time.Time{})
	if !errors.Is(err, errExchangeNameUnset) {
		t.Fatalf("received: %v, expected: %v", err, errExchangeNameUnset)
	}

	dummy.Exchange = testExchange
	_, err = NewDataFromKline(dummy, time.Time{}, time.Time{})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: %v, expected: %v", err, currency.ErrCurrencyPairEmpty)
	}

	dummy.Pair = pair
	_, err = NewDataFromKline(dummy, time.Time{}, time.Time{})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v, expected: %v", err, asset.ErrNotSupported)
	}

	dummy.Asset = asset.Spot
	_, err = NewDataFromKline(dummy, time.Time{}, time.Time{})
	if !errors.Is(err, gctcommon.ErrDateUnset) {
		t.Fatalf("received: %v, expected: %v", err, gctcommon.ErrDateUnset)
	}

	end := time.Now().Truncate(gctkline.OneDay.Duration())
	start := end.Add(-gctkline.OneDay.Duration())

	_, err = NewDataFromKline(dummy, start, end)
	if !errors.Is(err, gctkline.ErrInvalidInterval) {
		t.Fatalf("received: %v, expected: %v", err, gctkline.ErrInvalidInterval)
	}

	dummy.Interval = gctkline.OneDay
	_, err = NewDataFromKline(dummy, start, end)
	if !errors.Is(err, errNoCandleData) {
		t.Fatalf("received: %v, expected: %v", err, errNoCandleData)
	}

	dummy.Candles = []gctkline.Candle{{Time: start.Add(-time.Duration(gctkline.OneDay)).UTC()}}
	_, err = NewDataFromKline(dummy, start, end)
	if !errors.Is(err, gctkline.ErrInvalidTimePeriod) {
		t.Fatalf("received: %v, expected: %v", err, gctkline.ErrInvalidTimePeriod)
	}

	dummy.Candles = []gctkline.Candle{{Time: start.UTC()}}
	data, err := NewDataFromKline(dummy, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}

	stream, err := data.Base.GetStream()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}

	if len(stream) != 1 {
		t.Fatalf("received: %v, expected: %v", len(stream), 1)
	}
}

func TestHasDataAtTime(t *testing.T) {
	t.Parallel()

	var dataKline *DataFromKline
	_, err := dataKline.HasDataAtTime(time.Now())
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	dataKline = &DataFromKline{}
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	dataKline = &DataFromKline{Base: &data.Base{}}
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	dummy := &gctkline.Item{
		Exchange: testExchange,
		Pair:     pair,
		Asset:    asset.Spot,
		Interval: gctkline.OneDay,
		Candles:  []gctkline.Candle{{Time: start, Open: 1337, High: 1337, Low: 1337, Close: 1337, Volume: 1337}},
	}

	dataKline, err = NewDataFromKline(dummy, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}

	has, err := dataKline.HasDataAtTime(time.Now())
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if has {
		t.Error("expected false")
	}

	has, err = dataKline.HasDataAtTime(start)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if !has {
		t.Error("expected true")
	}

	err = dataKline.SetLive(true)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	has, err = dataKline.HasDataAtTime(time.Time{})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if has {
		t.Error("expected false")
	}
	has, err = dataKline.HasDataAtTime(start)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if !has {
		t.Error("expected true")
	}
}

func TestAppend(t *testing.T) {
	t.Parallel()

	var dataKline *DataFromKline
	err := dataKline.AppendResults(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2020, 1, 4, 0, 0, 0, 0, time.UTC)
	dummy := &gctkline.Item{
		Exchange: testExchange,
		Pair:     pair,
		Asset:    asset.Spot,
		Interval: gctkline.OneDay,
		Candles: []gctkline.Candle{
			{
				Time:   start,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
			{
				Time:   start.Add(gctkline.OneDay.Duration()),
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}

	dataKline, err = NewDataFromKline(dummy, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}

	err = dataKline.AppendResults(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Fatalf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	err = dataKline.AppendResults(&gctkline.Item{})
	if !errors.Is(err, gctkline.ErrItemNotEqual) {
		t.Fatalf("received: %v, expected: %v", err, gctkline.ErrItemNotEqual)
	}

	err = dataKline.AppendResults(&gctkline.Item{Exchange: testExchange, Pair: pair, Asset: asset.Spot})
	if !errors.Is(err, errNoCandleData) {
		t.Fatalf("received: %v, expected: %v", err, errNoCandleData)
	}

	err = dataKline.AppendResults(dummy)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}

	err = dataKline.AppendResults(&gctkline.Item{
		Exchange: testExchange,
		Pair:     pair,
		Asset:    asset.Spot,
		Interval: gctkline.OneDay,
		Candles: []gctkline.Candle{
			{
				Time:   time.Date(2020, 1, 3, 0, 0, 0, 0, time.UTC),
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
			{
				Time:   time.Date(2020, 1, 4, 0, 0, 0, 0, time.UTC),
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}

	stream, err := dataKline.GetStream()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}

	if len(stream) != 4 {
		t.Fatalf("received: %v, expected: %v", len(stream), 4)
	}
}

func TestStream_OpenLowHighCloseVolume(t *testing.T) {
	t.Parallel()
	var dataKline *DataFromKline
	_, err := dataKline.StreamOpen()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	_, err = dataKline.StreamHigh()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	_, err = dataKline.StreamLow()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	_, err = dataKline.StreamClose()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	_, err = dataKline.StreamVol()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2020, 1, 4, 0, 0, 0, 0, time.UTC)
	dummy := &gctkline.Item{
		Exchange: testExchange,
		Pair:     pair,
		Asset:    asset.Spot,
		Interval: gctkline.OneDay,
		Candles: []gctkline.Candle{
			{
				Time:   start,
				Open:   1,
				High:   2,
				Low:    3,
				Close:  4,
				Volume: 5,
			},
			{
				Time:   start.Add(gctkline.OneDay.Duration()),
				Open:   1,
				High:   2,
				Low:    3,
				Close:  4,
				Volume: 5,
			},
		},
	}

	dataKline, err = NewDataFromKline(dummy, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v, expected: %v", err, nil)
	}
	event, err := dataKline.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if !event.GetTime().Equal(start) {
		t.Errorf("received: %v, expected: %v", event.GetTime(), start)
	}
	open, err := dataKline.StreamOpen()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(open) != 1 {
		t.Fatalf("received: %v, expected: %v", len(open), 1)
	}
	if !open[0].Equal(decimal.NewFromInt(1)) {
		t.Errorf("received: %v, expected: %v", open[0], 1)
	}

	high, err := dataKline.StreamHigh()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(high) != 1 {
		t.Fatalf("received: %v, expected: %v", len(high), 1)
	}
	if !high[0].Equal(decimal.NewFromInt(2)) {
		t.Errorf("received: %v, expected: %v", high[0], 2)
	}

	low, err := dataKline.StreamLow()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(low) != 1 {
		t.Fatalf("received: %v, expected: %v", len(low), 1)
	}
	if !low[0].Equal(decimal.NewFromInt(3)) {
		t.Errorf("received: %v, expected: %v", low[0], 3)
	}

	close, err := dataKline.StreamClose()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(close) != 1 {
		t.Fatalf("received: %v, expected: %v", len(close), 1)
	}
	if !close[0].Equal(decimal.NewFromInt(4)) {
		t.Errorf("received: %v, expected: %v", close[0], 4)
	}

	volume, err := dataKline.StreamVol()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(volume) != 1 {
		t.Fatalf("received: %v, expected: %v", len(volume), 1)
	}
	if !volume[0].Equal(decimal.NewFromInt(5)) {
		t.Errorf("received: %v, expected: %v", open[0], 5)
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	var dataKline *DataFromKline
	err := dataKline.validate()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}
	dataKline = &DataFromKline{}
	err = dataKline.validate()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	dataKline.Base = &data.Base{}
	err = dataKline.validate()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	dataKline.Item = &gctkline.Item{}
	err = dataKline.validate()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	dataKline.RangeHolder = &gctkline.IntervalRangeHolder{}
	err = dataKline.validate()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}
