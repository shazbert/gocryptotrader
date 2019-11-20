package engine

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

// Priority constants determines API function prirority. This allows
// reorganisation to occur on job queue heap e.g. priority1 for all order
// operations
const (
	// Cancel is lowest priority so we execute higher priority jobs under heavy
	// work loads and cleanup remnants when workloads drop
	cancel Priority = iota
	low
	medium
	high
	extreme

	defaultWorkerCount = int32(10)
)

var (
	errWorkManagerStopped = errors.New("work manager has stopped")
	errWorkManagerStarted = errors.New("work manager already started")
	errExchangNotFound    = errors.New("exchange not found")
	errJobCancelled       = errors.New("job cancelled")
)

// WorkManager defines a prioritised job queue manager for generalised API calls
// that will also act as a security layer i.e. general exchange rate limits and
// client call permission sets
type WorkManager struct {
	Jobs    PriorityJobQueue
	jobsMtx sync.Mutex

	workAvailable atomic.Value

	shutdown chan struct{}
	p        *sync.Pool

	wg          sync.WaitGroup
	workerCount int32
	started     int32
	running     int32
	verbose     bool
}

// Priority defines an explicit priority level
type Priority int

// Exchange couples a calling systems intended trading API functionality with
// an exchange
type Exchange struct {
	e  exchange.IBotExchange
	wm *WorkManager
}

// Command wraps execute functionality for our priority work queue
type Command interface {
	Execute()
}

// FetchTicker defines a coupler to an exchange REST request
type FetchTicker struct {
	exchange.IBotExchange
	Pair  currency.Pair
	Asset asset.Item
	Price ticker.Price
	Error error
}

// UpdateTicker defines a coupler to an exchange REST request
type UpdateTicker struct {
	exchange.IBotExchange
	Pair  currency.Pair
	Asset asset.Item
	Price ticker.Price
	Error error
}

// FetchOrderbook defines a coupler to an exchange REST request
type FetchOrderbook struct {
	exchange.IBotExchange
	Pair      currency.Pair
	Asset     asset.Item
	Orderbook orderbook.Base
	Error     error
}

// UpdateOrderbook defines a coupler to an exchange REST request
type UpdateOrderbook struct {
	exchange.IBotExchange
	Pair      currency.Pair
	Asset     asset.Item
	Orderbook orderbook.Base
	Error     error
}

// GetAccountInfo defines a coupler to an exchange REST request
type GetAccountInfo struct {
	exchange.IBotExchange
	AccountInfo exchange.AccountInfo
	Error       error
}

// GetExchangeHistory defines a coupler to an exchange REST request
type GetExchangeHistory struct {
	exchange.IBotExchange
	Request  *exchange.TradeHistoryRequest
	Response []exchange.TradeHistory
	Error    error
}

// GetFeeByType defines a coupler to an exchange REST request
type GetFeeByType struct {
	exchange.IBotExchange
	Request  *exchange.FeeBuilder
	Response float64
	Error    error
}

// GetFundingHistory defines a coupler to an exchange REST request
type GetFundingHistory struct {
	exchange.IBotExchange
	Response []exchange.FundHistory
	Error    error
}

// SubmitOrder defines a coupler to an exchange REST request
type SubmitOrder struct {
	exchange.IBotExchange
	Request  *order.Submit
	Response order.SubmitResponse
	Error    error
}

// ModifyOrder defines a coupler to an exchange REST request
type ModifyOrder struct {
	exchange.IBotExchange
	Request  *order.Modify
	Response string
	Error    error
}

// CancelOrder defines a coupler to an exchange REST request
type CancelOrder struct {
	exchange.IBotExchange
	Request *order.Cancel
	Error   error
}

// CancelAllOrders defines a coupler to an exchange REST request
type CancelAllOrders struct {
	exchange.IBotExchange
	Request  *order.Cancel
	Response order.CancelAllResponse
	Error    error
}

// GetOrderInfo defines a coupler to an exchange REST request
type GetOrderInfo struct {
	exchange.IBotExchange
	Request  string
	Response order.Detail
	Error    error
}

// GetDepositAddress defines a coupler to an exchange REST request
type GetDepositAddress struct {
	exchange.IBotExchange
	Crypto    currency.Code
	AccountID string
	Response  string
	Error     error
}

// GetOrderHistory defines a coupler to an exchange REST request
type GetOrderHistory struct {
	exchange.IBotExchange
	Request  *order.GetOrdersRequest
	Response []order.Detail
	Error    error
}

// GetActiveOrders defines a coupler to an exchange REST request
type GetActiveOrders struct {
	exchange.IBotExchange
	Request  *order.GetOrdersRequest
	Response []order.Detail
	Error    error
}

// WithdrawCryptocurrencyFunds defines a coupler to an exchange REST request
type WithdrawCryptocurrencyFunds struct {
	exchange.IBotExchange
	Request  *exchange.CryptoWithdrawRequest
	Response string
	Error    error
}

// WithdrawFiatFunds defines a coupler to an exchange REST request
type WithdrawFiatFunds struct {
	exchange.IBotExchange
	Request  *exchange.FiatWithdrawRequest
	Response string
	Error    error
}

// WithdrawFiatFundsToInternationalBank defines a coupler to an exchange REST
// request
type WithdrawFiatFundsToInternationalBank struct {
	exchange.IBotExchange
	Request  *exchange.FiatWithdrawRequest
	Response string
	Error    error
}
