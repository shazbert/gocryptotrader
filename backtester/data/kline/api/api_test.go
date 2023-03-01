package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const testExchange = "binanceus"

var cp = currency.NewPair(currency.BTC, currency.USDT)

func TestLoadCandlesAndLoadTrades(t *testing.T) {
	t.Parallel()
	em := engine.SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true}}

	// TODO: When GetHistoricTradesExtended functionality is implemented
	// increase time window to account for ~1k+ trades and matching with candles.
	tt1 := time.Now().Add(-time.Minute * 5).Truncate(gctkline.OneMin.Duration())
	tt2 := time.Now().Truncate(gctkline.OneMin.Duration())

	_, err = LoadData(context.Background(), -1, tt1, tt2, gctkline.OneMin, exch, cp, asset.Spot)
	if !errors.Is(err, common.ErrInvalidDataType) {
		t.Errorf("received: %v, expected: %v", err, common.ErrInvalidDataType)
	}

	dataCandles, err := LoadData(context.Background(), common.DataCandle, tt1, tt2, gctkline.OneMin, exch, cp, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	dataTrades, err := LoadData(context.Background(), common.DataTrade, tt1, tt2, gctkline.OneMin, exch, cp, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if len(dataCandles.Candles) != len(dataTrades.Candles) {
		t.Fatal("expected same candles and data from different data points")
	}

	for x := range dataCandles.Candles {
		// Will skip zero value volume as this is a dummy that was inserted and
		// no trading activity occured.
		if dataCandles.Candles[x].Volume == 0 && dataTrades.Candles[x].Volume == 0 {
			continue
		}

		// Volume has float rounding issues so this can be omitted in comparison.
		if dataCandles.Candles[x].Volume <= 0 || dataTrades.Candles[x].Volume <= 0 {
			t.Fatal("both volume amounts should be set")
		}

		if dataCandles.Candles[x].Time != dataTrades.Candles[x].Time {
			t.Fatalf("received: %v, expected: %v", dataTrades.Candles[x].Time, dataCandles.Candles[x].Time)
		}

		if dataCandles.Candles[x].Open != dataTrades.Candles[x].Open {
			t.Fatalf("received: %v, expected: %v", dataTrades.Candles[x].Open, dataCandles.Candles[x].Open)
		}

		if dataCandles.Candles[x].High != dataTrades.Candles[x].High {
			t.Fatalf("received: %v, expected: %v", dataTrades.Candles[x].High, dataCandles.Candles[x].High)
		}

		if dataCandles.Candles[x].Low != dataTrades.Candles[x].Low {
			t.Fatalf("received: %v, expected: %v", dataTrades.Candles[x].Low, dataCandles.Candles[x].Low)
		}

		if dataCandles.Candles[x].Close != dataTrades.Candles[x].Close {
			t.Fatalf("received: %v, expected: %v", dataTrades.Candles[x].Close, dataCandles.Candles[x].Close)
		}
	}
}
