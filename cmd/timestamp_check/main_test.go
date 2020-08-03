package main

import (
	"strings"
	"testing"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

func TestExchangeImpl(t *testing.T) {
loop:
	for x := range exchange.Exchanges {
		for y := range SupportedExchanges {
			SupportedExchanges[y].SetDefaults()
			if strings.EqualFold(exchange.Exchanges[x], SupportedExchanges[y].GetName()) {
				continue loop
			}
		}
		t.Errorf("%s not yet implemented, please add", exchange.Exchanges[x])
	}
}
