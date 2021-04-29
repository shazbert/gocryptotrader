package engine

import (
	"errors"
	"sync"
	"time"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/log"
)

type AccountManager struct {
	engine *Engine
	// Create a map so we can dynamically load and unload exchanges, timer used
	// so in the event of an exchange limit we can still set the default sync
	// time event to after it has just been updated. Having a rolling start line
	// is sub-optimal.
	accounts map[exchange.IBotExchange]chan struct{}
	syncTime time.Duration
	shutdown chan struct{}
	wg       sync.WaitGroup
	m        sync.Mutex
}

func NewAccountManager(e *Engine) (*AccountManager, error) {
	if e == nil {
		return nil, errors.New("engine is nil")
	}
	return &AccountManager{engine: e, accounts: make(map[exchange.IBotExchange]chan struct{})}, nil
}

func (a *AccountManager) Shutdown() error {
	a.m.Lock()
	defer a.m.Unlock()
	if a.shutdown == nil {
		return errors.New("account updater not started")
	}
	close(a.shutdown)
	a.wg.Wait()
	return nil
}

func (a *AccountManager) RunUpdater(syncTime time.Duration) error {
	a.m.Lock()
	defer a.m.Unlock()
	if a.shutdown != nil {
		return errors.New("account updater already started")
	}
	log.Debugln(log.Global, "Account balance manager started")
	a.syncTime = syncTime
	a.shutdown = make(chan struct{})
	a.wg.Add(1)
	go a.accountUpdater()
	return nil
}

func (a *AccountManager) accountUpdater() {
	tt := time.NewTimer(0)
	defer a.wg.Done()
	for {
		select {
		case <-tt.C:
			a.m.Lock()
			// Add exchange
			for _, exch := range a.engine.GetExchanges() {
				_, ok := a.accounts[exch]
				if ok {
					continue
				}
				log.Debugf(log.Global,
					"Account balance manager: %s monitor started.",
					exch.GetName())
				a.wg.Add(1)
				ch := make(chan struct{})
				a.accounts[exch] = ch
				go a.updateAccountForExchange(exch, ch, a.syncTime)
			}
			// Remove exchange
		accounts:
			for exch, shutdown := range a.accounts {
				for _, enabled := range a.engine.GetExchanges() {
					if exch == enabled {
						continue accounts
					}
				}
				log.Debugf(log.Global,
					"Account balance manager: %s monitor finished.",
					exch.GetName())
				close(shutdown)
				delete(a.accounts, exch)
			}
			a.m.Unlock()
		case <-a.shutdown:
			a.m.Lock()
			for _, ch := range a.accounts {
				close(ch)
			}
			a.m.Unlock()
			return
		}
		tt.Reset(time.Second * 10)
	}
}

func (a *AccountManager) updateAccountForExchange(exch exchange.IBotExchange, shutdown chan struct{}, syncTime time.Duration) {
	defer a.wg.Done()
	tt := time.NewTimer(syncTime)
	for {
		select {
		case <-tt.C:
			base := exch.GetBase()
			if !base.Config.API.AuthenticatedSupport {
				break
			}
			if base.Config.API.AuthenticatedWebsocketSupport {
				log.Debugln(log.Global, "Updating account balance via REST skipped; websocket enabled")
				// Account balance is handled by websocket connection
				// TODO: Check distinct capability
				break
			}
			accounts, err := base.GetAccounts()
			if err != nil {
				log.Errorf(log.Global, "%s failed to get accounts", exch.GetName())
			}

			log.Debugln(log.Global, "Updating account balance via REST")

			assets := exch.GetAssetTypes()
			for x := range accounts {
				for y := range assets {
					_, err = exch.UpdateAccountInfo(accounts[x], assets[y])
					if err != nil {
						log.Errorf(log.Global,
							"%s failed to update account holdings for account: %s asset: %s",
							exch.GetName(),
							accounts[x],
							assets[y])
						break
					}
					// TODO: Update portfolio positioning, would need to tie
					// into websocket as well.
				}
			}
		case <-shutdown:
			return
		case <-a.shutdown:
			return
		}
		tt.Reset(syncTime)
	}
}

func (a *AccountManager) IsRunning() bool {
	a.m.Lock()
	defer a.m.Unlock()
	return a.accounts != nil
}
