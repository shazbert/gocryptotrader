package engine

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio"
)

// vars for the fund manager package
var (
	PortfolioSleepDelay = time.Minute
)

type portfolioManager struct {
	started    int32
	processing int32
	shutdown   chan struct{}
	*portfolio.Base
}

func (p *portfolioManager) Started() bool {
	return atomic.LoadInt32(&p.started) == 1
}

func (p *portfolioManager) Start() error {
	if atomic.AddInt32(&p.started, 1) != 1 {
		return errors.New("portfolio manager already started")
	}

	log.Debugln(log.PortfolioMgr, "Portfolio manager starting...")
	Bot.Portfolio = &portfolio.Portfolio
	Bot.Portfolio.Seed(Bot.Config.Portfolio)
	p.shutdown = make(chan struct{})
	portfolio.Verbose = Bot.Settings.Verbose

	go p.run()
	return nil
}
func (p *portfolioManager) Stop() error {
	if atomic.LoadInt32(&p.started) == 0 {
		return fmt.Errorf("portfolio manager %w", subsystem.ErrSubSystemNotStarted)
	}
	defer func() {
		atomic.CompareAndSwapInt32(&p.started, 1, 0)
	}()

	log.Debugln(log.PortfolioMgr, "Portfolio manager shutting down...")
	close(p.shutdown)
	return nil
}

func (p *portfolioManager) run() {
	log.Debugln(log.PortfolioMgr, "Portfolio manager started.")
	Bot.ServicesWG.Add(1)

	tick := time.NewTicker(Bot.Settings.PortfolioManagerDelay)
	defer func() {
		tick.Stop()
		Bot.ServicesWG.Done()
		log.Debugf(log.PortfolioMgr, "Portfolio manager shutdown.")
	}()

	p.Base = portfolio.GetPortfolio()
	go p.processPortfolio()
	go p.SyncAddresses(Bot)
	go portfolio.StartPortfolioWatcher()
	for {
		select {
		case <-p.shutdown:
			return
		case <-tick.C:
			go p.processPortfolio()
		}
	}
}

func (p *portfolioManager) processPortfolio() {
	if !atomic.CompareAndSwapInt32(&p.processing, 0, 1) {
		return
	}

	data := p.GetPortfolioGroupedCoin()
	for key, value := range data {
		err := p.UpdatePortfolio(value, key)
		if err != nil {
			log.Errorf(log.PortfolioMgr,
				"PortfolioWatcher error %s for currency %s\n",
				err,
				key)
			continue
		}

		log.Debugf(log.PortfolioMgr,
			"Portfolio manager: Successfully updated address balance for %s address(es) %s\n",
			key,
			value)
	}

	enabledExchangeAccounts := Bot.GetAllEnabledExchangeAccountInfo()
	p.SeedExchangeAccountInfo(enabledExchangeAccounts.Data)
	atomic.CompareAndSwapInt32(&p.processing, 1, 0)
}

// SeedExchangeAccountInfo seeds account info
func (p *portfolioManager) SeedExchangeAccountInfo(accounts map[string]account.FullSnapshot) error {
	if len(accounts) == 0 {
		return errors.New("cannot seed portfolio, no account data")
	}

	for exchName, m1 := range accounts {
		for acc, m2 := range m1 {
			for ai, m3 := range m2 {
				for c, balance := range m3 {
					err := p.UpdateInsertExchangeBalance(exchName,
						&portfolio.Holdings{
							Account: acc,
							Holding: portfolio.Holding{
								Currency: c.String(),
								Asset:    ai.String(),
								Balance:  balance.Total,
							},
						})
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// Sync synchronises all deposit addresses
func (d *portfolioManager) SyncAddresses(bot *Engine) error {
	e := bot.GetExchanges()
	for x := range e {
		batched, err := e[x].GetDepositAddresses(account.Default)
		if err != nil {
			log.Errorf(log.PortfolioMgr, "GetDepositAddresses ERRORS OCCURED: %v", err)
		}

		if batched != nil {
			// On a successful batch call we can integrate all deposit addresses
			// associaated with the account.
			for y := range batched {
				err = d.LoadDepositAddress(e[x].GetName(),
					account.Default,
					batched[y].Address,
					batched[y].TagMemo,
					batched[y].Currency.String())
				if err != nil {
					return err
				}
			}
			continue
		}

		log.Warnf(log.PortfolioMgr,
			"Batching is not enabled for retreiving deposit address for exchange: %s, PR's are welcome.",
			e[x].GetName())

		// This constructs the currency codes needed for request and building
		// deposit address.
		m := make(map[currency.Code]struct{})
		ai := e[x].GetAssetTypes()
		for y := range ai {
			pairs, err := e[x].GetEnabledPairs(ai[y])
			if err != nil {
				return err
			}

			for z := range pairs {
				if pairs[z].Base.IsCryptocurrency() {
					m[pairs[z].Base] = struct{}{}
				}

				if pairs[z].Quote.IsCryptocurrency() {
					m[pairs[z].Quote] = struct{}{}
				}
			}
		}

		for code := range m {
			addr, err := e[x].GetDepositAddress(code, account.Default)
			if err != nil {
				return err
			}

			err = d.LoadDepositAddress(e[x].GetName(),
				account.Default,
				addr.Address,
				addr.TagMemo,
				code.String())
			if err != nil {
				return err
			}
		}
	}
	return nil
}
