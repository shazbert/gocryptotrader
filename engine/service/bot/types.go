package bot

import (
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Config defines a bot configuration
type Config struct {
	Funding            []Fund
	ExchangeCurrencies map[string]currency.Pairs
	Trading            Trader
}

// Trader interface defines functionality for trading decision software which
// acts on new data
type Trader interface {
	Setup(cfg interface{}) error
	Start() error
	CheckTicker(t ticker.Price) (*Decision, error)
	CheckOrderbook(t orderbook.Base) (*Decision, error)
	CheckRawTrade(tradeData string, interval int) (*Decision, error)
}

// Decision defines a trading algorithm decision on data feeds
type Decision struct {
	Price    float64
	Amount   float64
	Pair     currency.Pair
	Exchange string
}

// Fund defines an allowable amount associated with an account
type Fund struct {
	currency.Code
	Amount float64
}
