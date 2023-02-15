package kline

import (
	"errors"
	"testing"
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

const testExchange = "binance"

var (
	elite = decimal.NewFromInt(1337)
	pair  = currency.NewPair(currency.BTC, currency.USDT)
)

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

	start := time.Date(2020, 1, 0, 0, 0, 0, 0, time.UTC)
	end := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

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
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	tt1 := time.Date(2020, 1, 0, 0, 0, 0, 0, time.UTC)
	tt2 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	d := DataFromKline{
		Base: &data.Base{},
		Item: &gctkline.Item{
			Exchange: testExchange,
			Asset:    a,
			Pair:     p,
			Interval: gctkline.OneDay,
		},
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	item := gctkline.Item{
		Interval: gctkline.OneDay,
		Candles: []gctkline.Candle{
			{
				Time:   tt1,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
			{
				Time:   tt2,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}
	err := d.AppendResults(&item)
	if !errors.Is(err, gctkline.ErrItemNotEqual) {
		t.Errorf("received: %v, expected: %v", err, gctkline.ErrItemNotEqual)
	}

	item.Exchange = testExchange
	item.Pair = p
	item.Asset = a

	err = d.AppendResults(&item)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = d.AppendResults(&item)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	err = d.AppendResults(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}
}

func TestStreamOpen(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{
		Base: &data.Base{},
	}
	bad, err := d.StreamOpen()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(bad) > 0 {
		t.Error("expected no stream")
	}
	err = d.SetStream([]data.Event{
		&kline.Kline{
			Base: &event.Base{
				Exchange:     exch,
				Time:         time.Now(),
				Interval:     gctkline.OneDay,
				CurrencyPair: p,
				AssetType:    a,
			},
			Open:   elite,
			High:   elite,
			Low:    elite,
			Close:  elite,
			Volume: elite,
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	_, err = d.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	open, err := d.StreamOpen()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(open) == 0 {
		t.Error("expected open")
	}
}

func TestStreamVolume(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{
		Base: &data.Base{},
	}
	bad, err := d.StreamVol()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(bad) > 0 {
		t.Error("expected no stream")
	}
	err = d.SetStream([]data.Event{
		&kline.Kline{
			Base: &event.Base{
				Exchange:     exch,
				Time:         time.Now(),
				Interval:     gctkline.OneDay,
				CurrencyPair: p,
				AssetType:    a,
			},
			Open:   elite,
			High:   elite,
			Low:    elite,
			Close:  elite,
			Volume: elite,
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	_, err = d.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	vol, err := d.StreamVol()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(vol) == 0 {
		t.Error("expected volume")
	}
}

func TestStreamClose(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{
		Base: &data.Base{},
	}
	bad, err := d.StreamClose()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(bad) > 0 {
		t.Error("expected no stream")
	}

	err = d.SetStream([]data.Event{
		&kline.Kline{
			Base: &event.Base{
				Exchange:     exch,
				Time:         time.Now(),
				Interval:     gctkline.OneDay,
				CurrencyPair: p,
				AssetType:    a,
			},
			Open:   elite,
			High:   elite,
			Low:    elite,
			Close:  elite,
			Volume: elite,
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	_, err = d.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	cl, err := d.StreamClose()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(cl) == 0 {
		t.Error("expected close")
	}
}

func TestStreamHigh(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{
		Base: &data.Base{},
	}
	bad, err := d.StreamHigh()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(bad) > 0 {
		t.Error("expected no stream")
	}

	err = d.SetStream([]data.Event{
		&kline.Kline{
			Base: &event.Base{
				Exchange:     exch,
				Time:         time.Now(),
				Interval:     gctkline.OneDay,
				CurrencyPair: p,
				AssetType:    a,
			},
			Open:   elite,
			High:   elite,
			Low:    elite,
			Close:  elite,
			Volume: elite,
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	_, err = d.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	high, err := d.StreamHigh()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(high) == 0 {
		t.Error("expected high")
	}
}

func TestStreamLow(t *testing.T) {
	t.Parallel()
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := DataFromKline{
		Base:        &data.Base{},
		RangeHolder: &gctkline.IntervalRangeHolder{},
	}
	bad, err := d.StreamLow()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(bad) > 0 {
		t.Error("expected no stream")
	}

	err = d.SetStream([]data.Event{
		&kline.Kline{
			Base: &event.Base{
				Exchange:     exch,
				Time:         time.Now(),
				Interval:     gctkline.OneDay,
				CurrencyPair: p,
				AssetType:    a,
			},
			Open:   elite,
			High:   elite,
			Low:    elite,
			Close:  elite,
			Volume: elite,
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	_, err = d.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	low, err := d.StreamLow()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(low) == 0 {
		t.Error("expected low")
	}
}
