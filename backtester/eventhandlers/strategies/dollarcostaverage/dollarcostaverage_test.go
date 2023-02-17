package dollarcostaverage

import (
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestName(t *testing.T) {
	d := Strategy{}
	if n := d.Name(); n != Name {
		t.Errorf("expected %v", Name)
	}
}

func TestSupportsSimultaneousProcessing(t *testing.T) {
	s := Strategy{}
	if !s.SupportsSimultaneousProcessing() {
		t.Error("expected true")
	}
}

func TestSetCustomSettings(t *testing.T) {
	s := Strategy{}
	err := s.SetCustomSettings(nil)
	if !errors.Is(err, base.ErrCustomSettingsUnsupported) {
		t.Errorf("received: %v, expected: %v", err, base.ErrCustomSettingsUnsupported)
	}
}

func TestOnSignal(t *testing.T) {
	s := Strategy{}
	_, err := s.OnSignal(nil, nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
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

	da, err := kline.NewDataFromKline(dummy, dStart, dEnd)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := s.OnSignal(da, nil, nil)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if resp.GetDirection() != gctorder.MissingData {
		t.Error("expected missing data")
	}

	_, err = da.Next()
	if err != nil {
		t.Fatal(err)
	}

	resp, err = s.OnSignal(da, nil, nil)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if resp.GetDirection() != gctorder.Buy {
		t.Errorf("expected buy, received %v", resp.GetDirection())
	}
}

func TestOnSignals(t *testing.T) {
	s := Strategy{}
	_, err := s.OnSignal(nil, nil, nil)
	if !errors.Is(err, common.ErrNilEvent) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNilEvent)
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
	da, err := kline.NewDataFromKline(dummy, dStart, dEnd)
	if err != nil {
		t.Fatal(err)
	}

	var resp []signal.Event
	resp, err = s.OnSimultaneousSignals([]data.Handler{da}, nil, nil)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(resp) != 1 {
		t.Fatal("expected 1 response")
	}
	if resp[0].GetDirection() != gctorder.MissingData {
		t.Error("expected missing data")
	}

	_, err = da.Next()
	if err != nil {
		t.Fatal(err)
	}

	resp, err = s.OnSimultaneousSignals([]data.Handler{da}, nil, nil)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
	if len(resp) != 1 {
		t.Fatal("expected 1 response")
	}
	if resp[0].GetDirection() != gctorder.Buy {
		t.Error("expected buy")
	}
}

func TestSetDefaults(t *testing.T) {
	s := Strategy{}
	s.SetDefaults()
	if s != (Strategy{}) {
		t.Error("expected no changes")
	}
}
