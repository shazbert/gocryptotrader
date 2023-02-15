package csv

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const exch = "binance"

var pair = currency.NewPair(currency.BTC, currency.USDT)

func TestLoadDataCandles(t *testing.T) {
	data, err := LoadData(common.DataCandle,
		filepath.Join("..", "..", "..", "..", "testdata", "binance_BTCUSDT_24h_2019_01_01_2020_01_01.csv"),
		exch,
		gctkline.OneDay,
		pair,
		asset.Spot,
		false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	if len(data.Item.Candles) != 365 {
		t.Fatalf("received: '%v' but expected: '%v'", len(data.Item.Candles), 365)
	}

	stream, err := data.GetStream()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(stream) != 365 {
		t.Fatalf("received: '%v' but expected: '%v'", len(stream), 365)
	}
}

func TestLoadDataTrades(t *testing.T) {
	data, err := LoadData(common.DataTrade,
		filepath.Join("..", "..", "..", "..", "testdata", "binance_BTCUSDT_24h-trades_2020_11_16.csv"),
		exch,
		gctkline.OneMin,
		pair,
		asset.Spot,
		false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}

	if len(data.Item.Candles) != 4 {
		t.Fatalf("received: '%v' but expected: '%v'", len(data.Item.Candles), 4)
	}

	stream, err := data.GetStream()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if len(stream) != 4 {
		t.Fatalf("received: '%v' but expected: '%v'", len(stream), 4)
	}
}

func TestLoadDataInvalid(t *testing.T) {
	_, err := LoadData(
		-1,
		filepath.Join("..", "..", "..", "..", "testdata", "binance_BTCUSDT_24h-trades_2020_11_16.csv"),
		exch,
		gctkline.FifteenMin,
		pair,
		asset.Spot,
		false)
	if !errors.Is(err, common.ErrInvalidDataType) {
		t.Errorf("received: %v, expected: %v", err, common.ErrInvalidDataType)
	}

	_, err = LoadData(
		-1,
		filepath.Join("..", "..", "..", "..", "testdata", "binance_BTCUSDT_24h-trades_2020_11_16.csv"),
		exch,
		gctkline.FifteenMin,
		pair,
		asset.Spot,
		true)
	if !errors.Is(err, errNoUSDData) {
		t.Errorf("received: %v, expected: %v", err, errNoUSDData)
	}
}
