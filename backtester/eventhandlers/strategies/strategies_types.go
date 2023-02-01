package strategies

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
)

// ErrStrategyAlreadyExists returned when a strategy matches the same name
var ErrStrategyAlreadyExists = errors.New("strategy already exists")

// StrategyHolder holds strategies
type StrategyHolder []Handler

// Handler defines all functions required to run strategies against data events
type Handler interface {
	GetName() (string, error)
	GetDescription() (string, error)
	OnSignal(data.IntervalSegregated, funding.IFundingTransferer, portfolio.Handler) (signal.Events, error)
	// NOTE: [][]data.Handler 1st dimension is asset then second dimension is interval segregation
	OnSimultaneousSignals(data.AssetSegregated, funding.IFundingTransferer, portfolio.Handler) (signal.AssetEvents, error)
	UsingSimultaneousProcessing() (bool, error)
	SupportsSimultaneousProcessing() (bool, error)
	SetSimultaneousProcessing(bool) error
	SetCustomSettings(map[string]interface{}) error
	SetDefaults() error
	CloseAllPositions([]holdings.Holding, []data.Event) ([]signal.Event, error)
}
