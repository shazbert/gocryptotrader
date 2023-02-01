package data

import (
	"errors"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var (
	// ErrHandlerNotFound returned when a handler is not found for specified exchange, asset, pair
	ErrHandlerNotFound = errors.New("handler not found")
	// ErrInvalidEventSupplied returned when a bad event is supplied
	ErrInvalidEventSupplied = errors.New("invalid event supplied")
	// ErrEmptySlice is returned when the supplied slice is nil or empty
	ErrEmptySlice = errors.New("empty slice")
	// ErrEndOfData is returned when attempting to load the next offset when there is no more
	ErrEndOfData = errors.New("no more data to retrieve")

	errNothingToAdd    = errors.New("cannot append empty event to stream")
	errMisMatchedEvent = errors.New("cannot add event to stream, does not match")
)

// HandlerHolder stores an event handler per exchange asset pair
type HandlerHolder struct {
	data map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]map[gctkline.Interval]Handler
	m    sync.Mutex
}

// Holder interface dictates what a Data holder is expected to do
// TODO: Only one type is utilizing this interface. Just for the sake of keeping
// things simple we can remove this until multiple structs are interfacing.
type Holder interface {
	SetDataForCurrency(string, asset.Item, currency.Pair, gctkline.Interval, Handler) error
	GetAllData() (AssetSegregated, error)
	GetDataForCurrency(exchangeName string, a asset.Item, p currency.Pair) (IntervalSegregated, error)
	Reset() error
}

// Base is the base implementation of some interface functions where further
// specific functions are implemented in DataFromKline
type Base struct {
	latest     Event
	stream     []Event
	offset     int
	isLiveData bool
	m          sync.Mutex
}

// Details defines details of data handler storage.
type Details struct {
	ExchangeName string
	Asset        asset.Item
	Pair         currency.Pair
	Interval     gctkline.Interval
}

// // MultiInterval holds data handlers for each individual interval level
// // for that asset. NOTE: Name changes welcome.
// type MultiInterval struct {
// 	handlers []Handler
// }

// // GetIntervals returns the intervals for the events that are stored.
// func (m *MultiInterval) GetIntervals() ([]gctkline.Interval, error) {
// 	if m == nil {
// 		return nil, errors.New("this is nil bro")
// 	}

// 	var klines []gctkline.Interval
// 	for x := range m.handlers {
// 		d, err := m.handlers[x].GetDetails()
// 		if err != nil {
// 			return nil, err
// 		}
// 		klines = append(klines, d.Interval)
// 	}

// 	// Temp sort
// 	sort.Slice(klines, func(i, j int) bool { return klines[i] < klines[j] })
// 	return klines, nil
// }

// AssetSegregated defines a list of data handlers (these hold currency data)
// for different asset types e.g. BTC-USD SPOT or BTC-USDT SPOT.
type AssetSegregated []IntervalSegregated

// IntervalSegregated defines a list of data handlers (these hold currency data)
// for the same asset type but segragated by time intervals e.g. BTC-USD SPOT
// 1HR or BTC-USD SPOT 3HR etc.
type IntervalSegregated []Handler

// Handler interface for Loading and Streaming Data
type Handler interface {
	Loader
	Streamer
	GetDetails() (Details, error)
	Reset() error
}

// Loader interface for Loading Data into backtest supported format
type Loader interface {
	Load() error
	AppendStream(s ...Event) error
}

// Streamer interface handles loading, parsing, distributing BackTest Data
type Streamer interface {
	Next() (Event, error)
	// NextByTime will push forward event if found for multi interval processing
	// so this will allow e.g 1hr candles to fetch the next 5 hr only when it's
	// aligned correctly if not it will return the last event if available for
	// signal comparison.
	NextByTime(time.Time) (Event, error)
	GetStream() (Events, error)
	History() (Events, error)
	Latest() (Event, error)
	List() (Events, error)
	IsLastEvent() (bool, error)
	Offset() (int64, error)

	StreamOpen() ([]decimal.Decimal, error)
	StreamHigh() ([]decimal.Decimal, error)
	StreamLow() ([]decimal.Decimal, error)
	StreamClose() ([]decimal.Decimal, error)
	StreamVol() ([]decimal.Decimal, error)

	HasDataAtTime(time.Time) (bool, error)
}

// Event interface used for loading and interacting with Data
type Event interface {
	common.Event
	GetUnderlyingPair() currency.Pair
	GetClosePrice() decimal.Decimal
	GetHighPrice() decimal.Decimal
	GetLowPrice() decimal.Decimal
	GetOpenPrice() decimal.Decimal
	GetVolume() decimal.Decimal
}

// Events allows for some common functions on a slice of events
type Events []Event
