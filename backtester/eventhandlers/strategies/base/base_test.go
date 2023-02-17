package base

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	datakline "github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestNewSignal(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	_, err := s.NewSignal(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrNilPointer)
	}

	dStart := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	dummy := &gctkline.Item{
		Exchange: "binance",
		Pair:     currency.NewPair(currency.BTC, currency.USDT),
		Asset:    asset.Spot,
		Interval: gctkline.OneDay,
		Candles: []gctkline.Candle{
			{Time: dStart, Open: 1337, High: 1337, Low: 1337, Close: 1337, Volume: 1337},
		},
	}

	da, err := datakline.NewDataFromKline(dummy, dStart, dEnd)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := s.NewSignal(da)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
	}

	fmt.Println(resp)

	_, err = da.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	_, err = s.NewSignal(da)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestSetSimultaneousProcessing(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	is := s.UsingSimultaneousProcessing()
	if is {
		t.Error("expected false")
	}
	s.SetSimultaneousProcessing(true)
	is = s.UsingSimultaneousProcessing()
	if !is {
		t.Error("expected true")
	}
}

func TestCloseAllPositions(t *testing.T) {
	t.Parallel()
	s := &Strategy{}
	_, err := s.CloseAllPositions(nil, nil)
	if !errors.Is(err, gctcommon.ErrFunctionNotSupported) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrFunctionNotSupported)
	}
}
