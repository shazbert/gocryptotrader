package engine

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// syncBase stores information
type syncBase struct {
	IsUsingWebsocket bool
	IsUsingREST      bool
	IsProcessing     bool
	LastUpdated      time.Time
	HaveData         bool
	NumErrors        int
}

// syncAgent stores the sync agent info
type syncAgent struct {
	Created   time.Time
	Exchange  string
	AssetType asset.Item
	Pair      currency.Pair
	Ticker    syncBase
	Orderbook syncBase
	Trade     syncBase
}

// Config stores the currency pair config
type Config struct {
	SyncTicker           bool
	SyncOrderbook        bool
	SyncTrades           bool
	SyncContinuously     bool
	SyncTimeoutREST      time.Duration
	SyncTimeoutWebsocket time.Duration
	NumWorkers           int
	Verbose              bool
}

// syncManager stores the exchange currency pair syncer object
type syncManager struct {
	initSyncCompleted              int32
	initSyncStarted                int32
	started                        int32
	delimiter                      string
	uppercase                      bool
	initSyncStartTime              time.Time
	fiatDisplayCurrency            currency.Code
	websocketRoutineManagerEnabled bool
	mtx                            sync.Mutex
	initSyncWG                     sync.WaitGroup
	inService                      sync.WaitGroup

	syncAgents               map[string]map[currency.Pair]map[asset.Item]*syncAgent
	tickerBatchLastRequested map[string]time.Time

	remoteConfig    *config.RemoteControlConfig
	config          Config
	exchangeManager iExchangeManager

	timer    time.Timer
	jobs     chan job
	route    map[string]chan job // We can buffer this
	shutdown chan struct{}
}

type job struct {
	Exchange string
}

func (s *syncManager) router() {
	for {
		select {
		case job := <-s.jobs:
			pipe, ok := s.route[job.Exchange]
			if !ok {
				log.Errorln(log.SyncMgr, "cannot process job:", job.Exchange)
				break
			}
			pipe <- job
		case <-s.shutdown:
			return
		}
	}
}
