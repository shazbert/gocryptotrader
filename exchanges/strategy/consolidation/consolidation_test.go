package consolidation

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	strategy "github.com/thrasher-corp/gocryptotrader/exchanges/strategy/common"
)

var b binance.Binance

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Binance load config error", err)
	}
	b.SetDefaults()
	binanceConfig, err := cfg.GetExchangeConfig("Binance")
	if err != nil {
		log.Fatal("Binance Setup() init error", err)
	}
	err = b.Setup(binanceConfig)
	if err != nil {
		log.Fatal("Binance setup error", err)
	}
	os.Exit(m.Run())
}

// Backtester standard test backtester for overiding request functionality for
// hybrid backtester live functionality.
type Backtester struct {
	exchange.IBotExchange
	history BacktestKline
}

func (b *Backtester) GetHistoricCandles(ctx context.Context, pair currency.Pair, a asset.Item, start, end time.Time, in kline.Interval) (kline.Item, error) {
	return b.history.GetRange(start, end, false)
}

func TestOnsignal(t *testing.T) {
	var s *Strategy

	_, err := s.OnSignal(context.Background(), nil)
	if !errors.Is(err, strategy.ErrIsNil) {
		t.Fatalf("received: '%v' but expected '%v'", err, strategy.ErrIsNil)
	}

	s = &Strategy{}
	_, err = s.OnSignal(context.Background(), nil)
	if !errors.Is(err, strategy.ErrNilSignal) {
		t.Fatalf("received: '%v' but expected '%v'", err, strategy.ErrNilSignal)
	}

	_, err = s.OnSignal(context.Background(), .007)
	if !errors.Is(err, strategy.ErrConfigIsNil) {
		t.Fatalf("received: '%v' but expected '%v'", err, strategy.ErrConfigIsNil)
	}

	backtestStart := time.Date(2022, 11, 30, 0, 0, 0, 0, time.UTC)
	backtestEnd := backtestStart.Add(time.Minute * 400)

	candles, err := b.GetHistoricCandles(context.Background(),
		currency.NewPair(currency.BTC, currency.USDT),
		asset.Spot,
		backtestStart,
		backtestEnd,
		kline.OneMin)
	if err != nil {
		t.Fatal(err)
	}

	backtester := Backtester{IBotExchange: &b, history: BacktestKline(candles)}

	s.Config = &Configuration{
		Exchange: &backtester,
		Pair:     currency.NewPair(currency.BTC, currency.USDT),
		Asset:    asset.Spot,
		Lookback: 200,
		Simulate: true,
		Interval: kline.OneMin,
	}

	_, err = s.OnSignal(context.Background(), .007)
	if !errors.Is(err, strategy.ErrUnhandledSignal) {
		t.Fatalf("received: '%v' but expected '%v'", err, strategy.ErrUnhandledSignal)
	}

	_, err = s.OnSignal(context.Background(), time.Time{})
	if !errors.Is(err, errTimeNotSet) {
		t.Fatalf("received: '%v' but expected '%v'", err, errTimeNotSet)
	}

	_, err = s.OnSignal(context.Background(), time.Now())
	if !errors.Is(err, errSignalRequiresUTCAlignment) {
		t.Fatalf("received: '%v' but expected '%v'", err, errSignalRequiresUTCAlignment)
	}

	endDate := backtestStart.Add(time.Minute * 200) // 200 lookback for initial
	_, err = s.OnSignal(context.Background(), endDate.Add(time.Second))
	if !errors.Is(err, errIntervalMisalignment) {
		t.Fatalf("received: '%v' but expected '%v'", err, errIntervalMisalignment)
	}

	// Roll through signals
	for x := 0; x < 200; x++ {
		_, err = s.OnSignal(context.Background(), endDate.Add(time.Minute*time.Duration(x)))
		if !errors.Is(err, nil) {
			t.Fatalf("received: '%v' but expected '%v'", err, nil)
		}
	}
}

func TestXxx(t *testing.T) {
	fmt.Println(positionSizeAllocator(1000, 5, 6, 5))
}

func TestBro(t *testing.T) {
	bruh := time.Date(2022, 12, 24, 11, 0, 0, 0, time.Local)

	fmt.Println(bruh)

	fmt.Println(bruh.UTC())
}
