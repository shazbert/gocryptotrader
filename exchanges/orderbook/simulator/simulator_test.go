package simulator

import (
	"sync"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitstamp"
)

func TestSimulate(t *testing.T) {
	b := bitstamp.Bitstamp{}
	b.SetDefaults()
	var wg sync.WaitGroup
	err := b.Start(&wg)
	if err != nil {
		t.Fatal(err)
	}
	b.Verbose = false
	pair := currency.NewPair(currency.BTC, currency.USD)
	err = b.CurrencyPairs.EnablePair(asset.Spot, pair)
	if err != nil {
		t.Fatal(err)
	}
	o, err := b.FetchOrderbook(pair, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.SimulateOrder(10000000, true)
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.SimulateOrder(2171, false)
	if err != nil {
		t.Fatal(err)
	}
}
