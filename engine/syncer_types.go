package engine

// Retrieve rate limit for exchange systems
// Derive max protocol through put via exchange rate limit
// Retrieve exchange functionality
// Create sync agent for individual item with interval for update
// Allocate sync agent to low to high priority pools
// allocate a high low priority job buffer
// workers wait on channel
// Rate limit atomic counter
//

// Initial sync group == trades, account info including orders/fees, orderbook
// background items == ticker, ohlc etc
// priority items == trades, orders, fees, orderbook

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
)

// CurrencyPairSyncerConfig stores the currency pair config
type CurrencyPairSyncerConfig struct {
	SyncTicker       bool
	SyncOrderbook    bool
	SyncTrades       bool
	SyncContinuously bool
	SyncTimeout      time.Duration
	NumWorkers       int
	Verbose          bool
}

// ExchangeSyncerConfig stores the exchange syncer config
type ExchangeSyncerConfig struct {
	SyncDepositAddresses bool
	SyncOrders           bool
}

// SyncManager stores the exchange currency pair syncer object
type SyncManager struct {
	Config                   SyncConfig
	SyncAgents               []*SyncAgent
	tickerBatchLastRequested map[string]time.Time

	initSyncStartTime time.Time
	shutdown          int32
	initialChan       chan struct{}
	initSync          sync.WaitGroup
	sync.RWMutex
}

// SyncBase stores information
type SyncBase struct {
	IsProcessing int32
	LastUpdated  time.Time
	HaveData     bool
	NumErrors    int
}

// SyncAgent stores the sync agent info
type SyncAgent struct {
	Created       time.Time
	Exchange      string
	AssetType     asset.Item
	Pair          currency.Pair
	Features      *protocol.Features
	Ticker        *SyncBase
	Orderbook     *SyncBase
	Trade         *SyncBase
	HistoricTrade *SyncBase
}
