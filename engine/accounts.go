package engine

import (
	"errors"
	"sync"
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// AccountManager defines account management
type AccountManager struct {
	engine                  *Engine
	accounts                map[exchange.IBotExchange]int // synchronisation
	synchronizationInterval time.Duration
	shutdown                chan struct{}
	wg                      sync.WaitGroup
	m                       sync.Mutex
	verbose                 bool
}

var (
	errEngineIsNil                  = errors.New("engine is nil")
	errAccountManagerNotStarted     = errors.New("account manager not started")
	errAccountManagerAlreadyStarted = errors.New("account manager already started")
	errUnrealisticUpdateInterval    = errors.New("unrealistic update interval should be equal or greater than 10 seconds")
)

// NewAccountManager returns a pointer of a new instance of an account manager
func NewAccountManager(e *Engine, verbose bool) (*AccountManager, error) {
	if e == nil {
		return nil, errEngineIsNil
	}
	return &AccountManager{
		engine:   e,
		accounts: make(map[exchange.IBotExchange]int),
		verbose:  verbose,
	}, nil
}

// Shutdown shuts down account management instance
func (a *AccountManager) Shutdown() error {
	a.m.Lock()
	defer a.m.Unlock()
	if a.shutdown == nil {
		return errAccountManagerNotStarted
	}
	close(a.shutdown)
	a.wg.Wait()
	return nil
}

// RunUpdater takes in a sync duration and spawns an update routine.
func (a *AccountManager) RunUpdater(interval time.Duration) error {
	if interval < time.Second*10 {
		return errUnrealisticUpdateInterval
	}
	a.m.Lock()
	defer a.m.Unlock()
	if a.shutdown != nil {
		return errAccountManagerAlreadyStarted
	}
	if a.verbose {
		log.Debugln(log.Global, "Account Manager started...")
	}
	a.synchronizationInterval = interval
	a.shutdown = make(chan struct{})
	a.wg.Add(1)
	go a.accountUpdater()
	return nil
}

func (a *AccountManager) accountUpdater() {
	tt := time.NewTimer(a.synchronizationInterval) // Immediately set up exchanges
	defer a.wg.Done()
	for {
		select {
		case <-tt.C:
			exchs := a.engine.GetExchanges()
			for x := range exchs {
				go a.updateAccountForExchange(exchs[x])
			}
		case <-a.shutdown:
			return
		}
	}
}

func (a *AccountManager) updateAccountForExchange(exch exchange.IBotExchange) {
	base := exch.GetBase()
	if !base.Config.API.AuthenticatedSupport {
		return
	}
	if base.Config.API.AuthenticatedWebsocketSupport {
		// This extends the request out to 6 x the synchronisation duration
		a.m.Lock()
		count, ok := a.accounts[exch]
		if !ok {
			a.accounts[exch] = 1
			count = 1
		}
		if count%6 != 0 {
			a.accounts[exch]++
			a.m.Unlock()
			return
		}
		a.accounts[exch] = 1
		a.m.Unlock()
	}

	// at := exch.GetAssetTypes()
	// for x := range at {
	_, err := exch.UpdateAccountInfo()
	if err != nil {
		log.Errorf(log.Global,
			"%s failed to update account holdings for account: %v",
			exch.GetName(),
			err)
	}
	// }

	// TODO: Update portfolio positioning, would need to tie
	// into websocket as well.
}

func (a *AccountManager) IsRunning() bool {
	a.m.Lock()
	defer a.m.Unlock()
	return a.shutdown != nil
}
