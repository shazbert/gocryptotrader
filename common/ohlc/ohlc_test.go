package ohlc

import (
	"math/rand"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestPlatformHistory(t *testing.T) {
	var p []exchange.TradeHistory

	err := ValidatData(&p)
	if err == nil {
		t.Error("Test Failed - PlatformHistory ValidateData() error cannot be nil")
	}

	tn := time.Now()

	p = []exchange.TradeHistory{
		{Timestamp: tn.Add(2 * time.Minute), TID: "2"},
		{Timestamp: tn.Add(time.Minute), TID: "1"},
		{Timestamp: tn.Add(3 * time.Minute), TID: "3"},
	}

	err = ValidatData(&p)
	if err != nil {
		t.Error("Test Failed - PlatformHistory Sort() error cannot be nil", err)
	}

	p = []exchange.TradeHistory{
		{Timestamp: tn.Add(2 * time.Minute), TID: "2", Amount: 1, Price: 0},
	}

	err = ValidatData(&p)
	if err == nil {
		t.Error("Test Failed - PlatformHistory ValidateData() error cannot be nil")
	}

	p = []exchange.TradeHistory{
		{TID: "2", Amount: 1, Price: 0},
	}

	err = ValidatData(&p)
	if err == nil {
		t.Error("Test Failed - PlatformHistory ValidateData() error cannot be nil")
	}

	p = []exchange.TradeHistory{
		{Timestamp: tn.Add(2 * time.Minute), TID: "2", Amount: 1, Price: 1000},
		{Timestamp: tn.Add(time.Minute), TID: "1", Amount: 1, Price: 1001},
		{Timestamp: tn.Add(3 * time.Minute), TID: "3", Amount: 1, Price: 1001.5},
	}

	err = ValidatData(&p)
	if err != nil {
		t.Error("Test Failed - PlatformHistory ValidateData() error", err)
	}
}

func TestOHLC(t *testing.T) {
	var p []exchange.TradeHistory
	rand.Seed(time.Now().Unix())
	for i := 0; i < 24000; i++ {
		p = append(p, exchange.TradeHistory{
			Timestamp: time.Now().Add((time.Duration(rand.Intn(10)) * time.Minute) + (time.Duration(rand.Intn(10)) * time.Second)),
			TID:       crypto.HexEncodeToString([]byte(string(i))),
			Amount:    float64(rand.Intn(20)) + 1,
			Price:     1000 + float64(rand.Intn(1000)),
		})
	}

	c, err := CreateOHLC(p,
		5*time.Minute,
		currency.NewPair(currency.BTC, currency.USD),
		asset.Spot,
		"Binance")
	if err != nil {
		t.Error("Test Failed - CreateOHLC error", err)
	}

	t.Error(c)
}
