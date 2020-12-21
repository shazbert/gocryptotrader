package orderbook

import (
	"errors"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// const values for orderbook package
const (
	bidLoadBookFailure = "cannot load book for exchange %s pair %s asset %s for Bids: %w"
	askLoadBookFailure = "cannot load book for exchange %s pair %s asset %s for Asks: %w"
	bookLengthIssue    = "Potential book issue for exchange %s pair %s asset %s length Bids %d length Asks %d"
)

// Vars for the orderbook package
var (
	service *Service

	errExchangeNameUnset = errors.New("orderbook exchange name not set")
	errPairNotSet        = errors.New("orderbook currency pair not set")
	errAssetTypeNotSet   = errors.New("orderbook asset type not set")
	errNoOrderbook       = errors.New("orderbook bids and asks are empty")
	errPriceNotSet       = errors.New("price cannot be zero")
	errAmountInvalid     = errors.New("amount cannot be less or equal to zero")
	errOutOfOrder        = errors.New("pricing out of order")
	errDuplication       = errors.New("price duplication")
	errIDDuplication     = errors.New("id duplication")
	errPeriodUnset       = errors.New("funding rate period is unset")
)

func init() {
	service = new(Service)
	service.Books = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*Depth)
}

// Service holds orderbook information for each individual exchange
type Service struct {
	Books map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*Depth
	sync.Mutex
}

// Item stores the amount and price values
type Item struct {
	Amount float64
	Price  float64
	ID     int64

	// Funding rate field
	Period int64

	// Contract variables
	LiquidationOrders int64
	OrderCount        int64
}

// Items defines list of bid or ask levels
type Items []Item

// // Base holds the fields for the orderbook base
// type Base struct {
// 	Pair         currency.Pair `json:"pair"`
// 	Depth        Depth         `json:"depth"`
// 	LastUpdated  time.Time     `json:"lastUpdated"`
// 	LastUpdateID int64         `json:"lastUpdateId"`
// 	AssetType    asset.Item    `json:"assetType"`
// 	ExchangeName string        `json:"exchangeName"`
// 	// NotAggregated defines whether an orderbook can contain duplicate prices
// 	// in a payload
// 	NotAggregated bool `json:"-"`
// 	IsFundingRate bool `json:"fundingRate"`
// }

type byOBPrice []Item

func (a byOBPrice) Len() int           { return len(a) }
func (a byOBPrice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byOBPrice) Less(i, j int) bool { return a[i].Price < a[j].Price }

// Identifier defines fields that are required to match depth instance
type Identifier struct {
	Exchange string
	Pair     currency.Pair
	Asset    asset.Item
}

// Options define params for a depth instance
type Options struct {
	LastUpdated  time.Time
	LastUpdateID int64
	// NotAggregated defines whether an orderbook can contain duplicate prices
	// in a payload
	NotAggregated bool
	IsFundingRate bool
}
