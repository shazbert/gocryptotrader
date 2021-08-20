package engine

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stats"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// SyncItem is a sync item type
type SyncItem int

// const holds the sync item types
const (
	Ticker SyncItem = iota
	Orderbook
	Trade
	SyncManagerName = "exchange_syncer"
)

var (
	createdCounter = 0
	removedCounter = 0
	// DefaultSyncerWorkers limits the number of sync workers
	DefaultSyncerWorkers = 15
	// DefaultSyncerTimeoutREST the default time to switch from REST to websocket protocols without a response
	DefaultSyncerTimeoutREST = time.Second * 15
	// DefaultSyncerTimeoutWebsocket the default time to switch from websocket to REST protocols without a response
	DefaultSyncerTimeoutWebsocket = time.Minute
	errNoSyncItemsEnabled         = errors.New("no sync items enabled")
	errUnknownSyncItem            = errors.New("unknown sync item")
	errSyncPairNotFound           = errors.New("exchange currency pair syncer not found")
)

// setupSyncManager starts a new CurrencyPairSyncer
func setupSyncManager(c *Config, exchangeManager iExchangeManager, remoteConfig *config.RemoteControlConfig, websocketRoutineManagerEnabled bool) (*syncManager, error) {
	if !c.SyncOrderbook && !c.SyncTicker && !c.SyncTrades {
		return nil, errNoSyncItemsEnabled
	}
	if exchangeManager == nil {
		return nil, errNilExchangeManager
	}
	if remoteConfig == nil {
		return nil, errNilConfig
	}

	if c.NumWorkers <= 0 {
		c.NumWorkers = DefaultSyncerWorkers
	}

	if c.SyncTimeoutREST <= time.Duration(0) {
		c.SyncTimeoutREST = DefaultSyncerTimeoutREST
	}

	if c.SyncTimeoutWebsocket <= time.Duration(0) {
		c.SyncTimeoutWebsocket = DefaultSyncerTimeoutWebsocket
	}

	s := &syncManager{
		config:                         *c,
		remoteConfig:                   remoteConfig,
		exchangeManager:                exchangeManager,
		websocketRoutineManagerEnabled: websocketRoutineManagerEnabled,
		tickerBatchLastRequested:       make(map[string]time.Time),
	}

	log.Debugf(log.SyncMgr,
		"Exchange currency pair syncer config: continuous: %v ticker: %v"+
			" orderbook: %v trades: %v workers: %v verbose: %v timeout REST: %v"+
			" timeout Websocket: %v",
		s.config.SyncContinuously, s.config.SyncTicker, s.config.SyncOrderbook,
		s.config.SyncTrades, s.config.NumWorkers, s.config.Verbose, s.config.SyncTimeoutREST,
		s.config.SyncTimeoutWebsocket)
	s.inService.Add(1)
	return s, nil
}

// IsRunning safely checks whether the subsystem is running
func (m *syncManager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

// Start runs the subsystem
func (m *syncManager) Start() error {
	if m == nil {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return ErrSubSystemAlreadyStarted
	}
	m.initSyncWG.Add(1)
	m.inService.Done()
	log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer started.")

	exchanges := m.exchangeManager.GetExchanges()
	for x := range exchanges {
		err := m.GenerateAgentsByExchange(exchanges[x])
		if err != nil {
			return err
		}
	}

	if atomic.CompareAndSwapInt32(&m.initSyncStarted, 0, 1) {
		log.Debugf(log.SyncMgr,
			"Exchange CurrencyPairSyncer initial sync started. %d items to process.",
			createdCounter)
		m.initSyncStartTime = time.Now()
	}

	go func() {
		m.initSyncWG.Wait()
		if atomic.CompareAndSwapInt32(&m.initSyncCompleted, 0, 1) {
			log.Debugf(log.SyncMgr, "Exchange CurrencyPairSyncer initial sync is complete.")
			completedTime := time.Now()
			log.Debugf(log.SyncMgr, "Exchange CurrencyPairSyncer initial sync took %v [%v sync items].",
				completedTime.Sub(m.initSyncStartTime), createdCounter)

			if !m.config.SyncContinuously {
				log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopping.")
				err := m.Stop()
				if err != nil {
					log.Error(log.SyncMgr, err)
				}
				return
			}
		}
	}()

	if atomic.LoadInt32(&m.initSyncCompleted) == 1 && !m.config.SyncContinuously {
		return nil
	}

	for i := 0; i < m.config.NumWorkers; i++ {
		go m.worker()
	}
	m.initSyncWG.Done()
	return nil
}

// GenerateAgentsByExchange generates new syncronisation agents for an exchange.
func (m *syncManager) GenerateAgentsByExchange(exch exchange.IBotExchange) error {
	if !exch.SupportsREST() && !exch.SupportsWebsocket() {
		log.Warnf(log.SyncMgr,
			"Loaded exchange %s does not support REST or Websocket.",
			exch.GetName())
		return nil
	}

	wsAvail, restAvail := m.protocolAvailability(exch)

	assetTypes := exch.GetAssetTypes(false)
	for y := range assetTypes {
		if !exch.IsAssetEnabled(assetTypes[y]) {
			log.Warnf(log.SyncMgr,
				"%s asset type %s is disabled, fetching enabled pairs is paused",
				exch.GetName(),
				assetTypes[y])
			continue
		}

		wsAssetSupported := exch.IsAssetWebsocketSupported(assetTypes[y])
		if !wsAssetSupported {
			if !restAvail {
				log.Warnf(log.SyncMgr,
					"%s asset type %s websocket functionality is unsupported & REST protocol is not available, skipping.",
					exch.GetName(),
					assetTypes[y])
				continue
			}
			log.Warnf(log.SyncMgr,
				"%s asset type %s websocket functionality is unsupported, REST fetching only.",
				exch.GetName(),
				assetTypes[y])
		}

		enabledPairs, err := exch.GetEnabledPairs(assetTypes[y])
		if err != nil {
			log.Errorf(log.SyncMgr,
				"%s failed to get enabled pairs. Err: %s",
				exch.GetName(),
				err)
			continue
		}
		for i := range enabledPairs {
			c := &syncAgent{
				Exchange:  exch.GetName(),
				AssetType: assetTypes[y],
				Pair:      enabledPairs[i],
				Created:   time.Now(),
			}
			sBase := syncBase{
				IsUsingREST:      restAvail,
				IsUsingWebsocket: wsAvail && wsAssetSupported,
			}
			if m.config.SyncTicker {
				c.Ticker = sBase
			}
			if m.config.SyncOrderbook {
				c.Orderbook = sBase
			}
			if m.config.SyncTrades {
				c.Trade = sBase
			}
			m.addUpdate(c)
		}
	}
	return nil
}

// protocolAvailability determines for each exchange what protocol is available
// for syncrhonisation.
func (m *syncManager) protocolAvailability(exch exchange.IBotExchange) (websocket, rest bool) {
	return m.websocketRoutineManagerEnabled &&
			exch.SupportsWebsocket() &&
			exch.IsWebsocketEnabled(),
		exch.SupportsREST()
}

// Stop shuts down the exchange currency pair syncer
func (m *syncManager) Stop() error {
	if m == nil {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 1, 0) {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", ErrSubSystemNotStarted)
	}
	m.inService.Add(1)
	log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopped.")
	return nil
}

func (m *syncManager) getAgent(exchangeName string, p currency.Pair, a asset.Item) (*syncAgent, error) {
	m1, ok := m.syncAgents[exchangeName]
	if !ok {
		return nil, errors.New("exchange not found in agent list")
		// fmt.Errorf("%v %v %v %w", exchangeName, a, p, errSyncPairNotFound)
	}

	m2, ok := m1[p]
	if !ok {
		return nil, errors.New("pair not found in agent list")
	}

	agent, ok := m2[a]
	if !ok {
		return nil, errors.New("agent not found for asset")
	}

	return agent, nil
}

func (m *syncManager) addUpdate(c *syncAgent) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	m1, ok := m.syncAgents[c.Exchange]
	if !ok {
		m1 = make(map[currency.Pair]map[asset.Item]*syncAgent)
		m.syncAgents[c.Exchange] = m1
	}

	m2, ok := m1[c.Pair]
	if !ok {
		m2 = make(map[asset.Item]*syncAgent)
		m1[c.Pair] = m2
	}

	_, ok = m2[c.AssetType]
	if !ok {
		m2[c.AssetType] = c
	} else {
		// *val = *c // crap update
		return
	}

	if m.config.SyncTicker {
		if m.config.Verbose {
			log.Debugf(log.SyncMgr,
				"%s: Added ticker sync item %v: using websocket: %v using REST: %v",
				c.Exchange, m.FormatCurrency(c.Pair).String(), c.Ticker.IsUsingWebsocket,
				c.Ticker.IsUsingREST)
		}
		m.increaseCounter()
	}

	if m.config.SyncOrderbook {
		if m.config.Verbose {
			log.Debugf(log.SyncMgr,
				"%s: Added orderbook sync item %v: using websocket: %v using REST: %v",
				c.Exchange, m.FormatCurrency(c.Pair).String(), c.Orderbook.IsUsingWebsocket,
				c.Orderbook.IsUsingREST)
		}
		m.increaseCounter()
	}

	if m.config.SyncTrades {
		if m.config.Verbose {
			log.Debugf(log.SyncMgr,
				"%s: Added trade sync item %v: using websocket: %v using REST: %v",
				c.Exchange, m.FormatCurrency(c.Pair).String(), c.Trade.IsUsingWebsocket,
				c.Trade.IsUsingREST)
		}
		m.increaseCounter()
	}
}

// increaseCounter increases the counter when it is still in initial sync.
func (m *syncManager) increaseCounter() {
	if atomic.LoadInt32(&m.initSyncCompleted) != 1 {
		m.initSyncWG.Add(1)
		createdCounter++
	}
}

// isProcessing determines if the sync item is processing.
func (m *syncManager) isProcessing(exchangeName string, p currency.Pair, a asset.Item, sync SyncItem) (bool, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	agent, err := m.getAgent(exchangeName, p, a)
	if err != nil {
		return false, err
	}

	return agent.isProcessing(sync)
}

func (a *syncAgent) isProcessing(sync SyncItem) (bool, error) {
	switch sync {
	case Ticker:
		return a.Ticker.IsProcessing, nil
	case Orderbook:
		return a.Orderbook.IsProcessing, nil
	case Trade:
		return a.Trade.IsProcessing, nil
	default:
		return false, errors.New("sync type not handled")
	}
}

func (m *syncManager) setProcessing(exchangeName string, p currency.Pair, a asset.Item, sync SyncItem, processing bool) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	agent, err := m.getAgent(exchangeName, p, a)
	if err != nil {
		return err
	}

	switch sync {
	case Ticker:
		agent.Ticker.IsProcessing = processing
	case Orderbook:
		agent.Orderbook.IsProcessing = processing
	case Trade:
		agent.Trade.IsProcessing = processing
	default:
		return errors.New("sync type not handled")
	}
	return nil
}

// Update notifies the syncManager to change the last updated time for a
// exchange asset pair
func (m *syncManager) Update(exchangeName string, p currency.Pair, a asset.Item, sync SyncItem, incoming error) error {
	if m == nil {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("exchange CurrencyPairSyncer %w", ErrSubSystemNotStarted)
	}

	if atomic.LoadInt32(&m.initSyncStarted) != 1 {
		return nil
	}

	enabled, err := m.syncItemEnabled(sync)
	if err != nil {
		return err
	}

	if !enabled {
		return nil
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()

	agent, err := m.getAgent(exchangeName, p, a) // TODO: Agent needs mutex protection
	if err != nil {
		return err
	}

	var syncStr string
	switch sync {
	case Ticker:
		syncStr = "ticker"
	case Orderbook:
		syncStr = "orderbook"
	case Trade:
		syncStr = "trade"
	default:
		return errors.New("unhandled sync type")
	}

	return m.update(agent.Exchange,
		syncStr,
		agent.Pair,
		agent.AssetType,
		&agent.Ticker,
		incoming)
}

// update updates the time values of the sync base and sets the intial sync
// status
func (m *syncManager) update(exchange, sync string, pair currency.Pair, a asset.Item, b *syncBase, incoming error) error {
	if m == nil {
		return errors.New("sync manager is nil")
	}

	hadData := b.update(incoming)
	if atomic.LoadInt32(&m.initSyncCompleted) == 1 || hadData {
		return nil
	}

	removedCounter++
	m.initSyncWG.Done()

	log.Debugf(log.SyncMgr, "%s %s sync complete %v %v [%d/%d].",
		exchange,
		sync,
		m.FormatCurrency(pair).String(),
		a,
		removedCounter,
		createdCounter)
	return nil
}

// update updates the sync base time for each individual sync item
func (b *syncBase) update(incoming error) (hadData bool) {
	b.LastUpdated = time.Now()
	if incoming != nil {
		b.NumErrors++
	}
	hadData = b.HaveData
	b.HaveData = true
	b.IsProcessing = false
	return
}

// syncItemEnabled determines by config if a sync item is enabled
func (m *syncManager) syncItemEnabled(sync SyncItem) (bool, error) {
	switch sync {
	case Orderbook:
		return m.config.SyncOrderbook, nil
	case Ticker:
		return m.config.SyncTicker, nil
	case Trade:
		return m.config.SyncTrades, nil
	default:
		return false, fmt.Errorf("%v %w", sync, errUnknownSyncItem)
	}
}

func (m *syncManager) worker() {
	defer func() {
		log.Debugln(log.SyncMgr,
			"Exchange CurrencyPairSyncer worker shutting down.")
	}()

	for atomic.LoadInt32(&m.started) != 0 {
		exchanges := m.exchangeManager.GetExchanges()
		for x := range exchanges {
			exchangeName := exchanges[x].GetName()
			supportsREST := exchanges[x].SupportsREST()
			supportsRESTTickerBatching := exchanges[x].SupportsRESTTickerBatchUpdates()
			var usingREST bool
			var usingWebsocket bool
			var switchedToRest bool
			if exchanges[x].SupportsWebsocket() && exchanges[x].IsWebsocketEnabled() {
				ws, err := exchanges[x].GetWebsocket()
				if err != nil {
					log.Errorf(log.SyncMgr,
						"%s unable to get websocket pointer. Err: %s",
						exchangeName,
						err)
					usingREST = true
				}

				if ws.IsConnected() {
					usingWebsocket = true
				} else {
					usingREST = true
				}
			} else if supportsREST {
				usingREST = true
			}

			assetTypes := exchanges[x].GetAssetTypes(true)
			for y := range assetTypes {
				wsAssetSupported := exchanges[x].IsAssetWebsocketSupported(assetTypes[y])
				enabledPairs, err := exchanges[x].GetEnabledPairs(assetTypes[y])
				if err != nil {
					log.Errorf(log.SyncMgr,
						"%s failed to get enabled pairs. Err: %s",
						exchangeName,
						err)
					continue
				}
				for i := range enabledPairs {
					if atomic.LoadInt32(&m.started) == 0 {
						return
					}

					agent, err := m.getAgent(exchangeName, enabledPairs[i], assetTypes[y])
					if err != nil {
						if err != errSyncPairNotFound {
							log.Error(log.SyncMgr, err)
							continue
						}
						agent = &syncAgent{
							AssetType: assetTypes[y],
							Exchange:  exchangeName,
							Pair:      enabledPairs[i],
							Created:   time.Now(),
						}

						sBase := syncBase{
							IsUsingREST:      usingREST || !wsAssetSupported,
							IsUsingWebsocket: usingWebsocket && wsAssetSupported,
						}

						if m.config.SyncTicker {
							agent.Ticker = sBase
						}

						if m.config.SyncOrderbook {
							agent.Orderbook = sBase
						}

						if m.config.SyncTrades {
							agent.Trade = sBase
						}

						m.addUpdate(agent)
					}

					if switchedToRest && usingWebsocket {
						log.Warnf(log.SyncMgr,
							"%s %s: Websocket re-enabled, switching from rest to websocket",
							agent.Exchange,
							m.FormatCurrency(enabledPairs[i]).String())
						switchedToRest = false
					}

					if m.config.SyncOrderbook {
						err = m.processOrderbook(agent, exchanges[x], usingWebsocket, usingREST, switchedToRest)
						if err != nil {
							log.Errorln(log.SyncMgr, err)
						}
					}

					if m.config.SyncTicker {
						err = m.processTicker(agent, exchanges[x], usingWebsocket, usingREST, switchedToRest, supportsRESTTickerBatching)
						if err != nil {
							log.Errorln(log.SyncMgr, err)
						}
					}

					if m.config.SyncTrades {
						err = m.processTrade(agent)
						if err != nil {
							log.Errorln(log.SyncMgr, err)
						}
					}
				}
			}
		}
	}
}

// processOrderbook processes orderbook things
func (m *syncManager) processOrderbook(agent *syncAgent, exch exchange.IBotExchange, usingWebsocket, usingRest, switchedToRest bool) error {
	proc, err := m.isProcessing(agent.Exchange, agent.Pair, agent.AssetType, Orderbook)
	if err != nil {
		return err
	}

	if proc {
		return nil
	}

	if agent.Orderbook.LastUpdated.IsZero() ||
		(time.Since(agent.Orderbook.LastUpdated) > m.config.SyncTimeoutREST &&
			agent.Orderbook.IsUsingREST) ||
		(time.Since(agent.Orderbook.LastUpdated) > m.config.SyncTimeoutWebsocket &&
			agent.Orderbook.IsUsingWebsocket) {
		if agent.Orderbook.IsUsingWebsocket {
			if time.Since(agent.Created) < m.config.SyncTimeoutWebsocket {
				return nil
			}

			if exch.SupportsREST() {
				err = m.setProcessing(agent.Exchange, agent.Pair, agent.AssetType, Orderbook, true)
				if err != nil {
					return err
				}

				agent.Orderbook.IsUsingWebsocket = false
				agent.Orderbook.IsUsingREST = true
				log.Warnf(log.SyncMgr,
					"%s %s %s: No orderbook update after %s, switching from websocket to rest",
					agent.Exchange,
					m.FormatCurrency(agent.Pair).String(),
					strings.ToUpper(agent.AssetType.String()),
					m.config.SyncTimeoutWebsocket,
				)
				switchedToRest = true
				err = m.setProcessing(agent.Exchange,
					agent.Pair,
					agent.AssetType,
					Orderbook,
					false)
				if err != nil {
					return err
				}
			}
		}

		err = m.setProcessing(agent.Exchange,
			agent.Pair,
			agent.AssetType,
			Orderbook,
			true)
		if err != nil {
			return err
		}

		result, err := exch.UpdateOrderbook(agent.Pair, agent.AssetType)
		m.PrintOrderbookSummary(result, "REST", err)
		if err == nil && m.remoteConfig.WebsocketRPC.Enabled {
			relayWebsocketEvent(result,
				"orderbook_update",
				agent.AssetType.String(),
				agent.Exchange)
		}

		updateErr := m.Update(agent.Exchange,
			agent.Pair,
			agent.AssetType,
			Orderbook,
			err)
		if updateErr != nil {
			log.Error(log.SyncMgr, updateErr)
		}
	} else {
		time.Sleep(time.Millisecond * 50)
	}

	return nil
}

// processTicker processes ticker things
func (m *syncManager) processTicker(agent *syncAgent, exch exchange.IBotExchange, usingWebsocket, usingRest, switchedToRest, tickerBatch bool) error {
	proc, err := m.isProcessing(agent.Exchange, agent.Pair, agent.AssetType, Ticker)
	if err != nil {
		return err
	}

	if proc {
		return nil
	}

	if agent.Ticker.LastUpdated.IsZero() ||
		(time.Since(agent.Ticker.LastUpdated) > m.config.SyncTimeoutREST &&
			agent.Ticker.IsUsingREST) ||
		(time.Since(agent.Ticker.LastUpdated) > m.config.SyncTimeoutWebsocket &&
			agent.Ticker.IsUsingWebsocket) {
		if agent.Ticker.IsUsingWebsocket {
			if time.Since(agent.Created) < m.config.SyncTimeoutWebsocket {
				return nil
			}

			if exch.SupportsREST() {
				err = m.setProcessing(agent.Exchange,
					agent.Pair,
					agent.AssetType,
					Ticker,
					true)
				if err != nil {
					return err
				}

				agent.Ticker.IsUsingWebsocket = false
				agent.Ticker.IsUsingREST = true
				log.Warnf(log.SyncMgr,
					"%s %s %s: No ticker update after %s, switching from websocket to rest",
					agent.Exchange,
					m.FormatCurrency(agent.Pair).String(),
					strings.ToUpper(agent.AssetType.String()),
					m.config.SyncTimeoutWebsocket,
				)
				switchedToRest = true
				err = m.setProcessing(agent.Exchange,
					agent.Pair,
					agent.AssetType,
					Ticker,
					false)
				if err != nil {
					return err
				}
			}
		}

		if agent.Ticker.IsUsingREST {
			err = m.setProcessing(agent.Exchange,
				agent.Pair,
				agent.AssetType,
				Ticker,
				true)
			if err != nil {
				return err
			}
			var result *ticker.Price
			var err error

			if tickerBatch {
				m.mtx.Lock()
				batchLastDone, ok := m.tickerBatchLastRequested[agent.Exchange]
				if !ok {
					m.tickerBatchLastRequested[agent.Exchange] = time.Time{}
				}
				m.mtx.Unlock()

				if batchLastDone.IsZero() ||
					time.Since(batchLastDone) > m.config.SyncTimeoutREST {
					m.mtx.Lock()
					if m.config.Verbose {
						log.Debugf(log.SyncMgr,
							"Initialising %s REST ticker batching",
							agent.Exchange)
					}
					result, err = exch.UpdateTicker(agent.Pair, agent.AssetType)
					m.tickerBatchLastRequested[agent.Exchange] = time.Now()
					m.mtx.Unlock()
				} else {
					if m.config.Verbose {
						log.Debugf(log.SyncMgr,
							"%s Using recent batching cache",
							agent.Exchange)
					}
					result, err = exch.FetchTicker(agent.Pair, agent.AssetType)
				}
			} else {
				result, err = exch.UpdateTicker(agent.Pair, agent.AssetType)
			}
			m.PrintTickerSummary(result, "REST", err)
			if err == nil {
				if m.remoteConfig.WebsocketRPC.Enabled {
					relayWebsocketEvent(result,
						"ticker_update",
						agent.AssetType.String(),
						agent.Exchange)
				}
			}
			updateErr := m.Update(agent.Exchange,
				agent.Pair,
				agent.AssetType,
				Ticker,
				err)
			if updateErr != nil {
				log.Error(log.SyncMgr, updateErr)
			}
		}
	} else {
		time.Sleep(time.Millisecond * 50)
	}

	return nil
}

// processTrade processes trade things
func (m *syncManager) processTrade(agent *syncAgent) error {
	proc, err := m.isProcessing(agent.Exchange, agent.Pair, agent.AssetType, Trade)
	if err != nil {
		return err
	}

	if proc {
		return nil
	}

	if agent.Trade.LastUpdated.IsZero() ||
		time.Since(agent.Trade.LastUpdated) > m.config.SyncTimeoutREST {
		err = m.setProcessing(agent.Exchange,
			agent.Pair,
			agent.AssetType,
			Trade,
			true)
		if err != nil {
			return err
		}
		err = m.Update(agent.Exchange,
			agent.Pair,
			agent.AssetType,
			Trade,
			nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func printCurrencyFormat(price float64, displayCurrency currency.Code) string {
	displaySymbol, err := currency.GetSymbolByCurrencyName(displayCurrency)
	if err != nil {
		log.Errorf(log.SyncMgr, "Failed to get display symbol: %s", err)
	}

	return fmt.Sprintf("%s%.8f", displaySymbol, price)
}

func printConvertCurrencyFormat(origCurrency currency.Code, origPrice float64, displayCurrency currency.Code) string {
	conv, err := currency.ConvertCurrency(origPrice,
		origCurrency,
		displayCurrency)
	if err != nil {
		log.Errorf(log.SyncMgr, "Failed to convert currency: %s", err)
	}

	displaySymbol, err := currency.GetSymbolByCurrencyName(displayCurrency)
	if err != nil {
		log.Errorf(log.SyncMgr, "Failed to get display symbol: %s", err)
	}

	origSymbol, err := currency.GetSymbolByCurrencyName(origCurrency)
	if err != nil {
		log.Errorf(log.SyncMgr, "Failed to get original currency symbol for %s: %s",
			origCurrency,
			err)
	}

	return fmt.Sprintf("%s%.2f %s (%s%.2f %s)",
		displaySymbol,
		conv,
		displayCurrency,
		origSymbol,
		origPrice,
		origCurrency,
	)
}

// PrintTickerSummary outputs the ticker results
func (m *syncManager) PrintTickerSummary(result *ticker.Price, protocol string, err error) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return
	}
	if err != nil {
		if err == common.ErrNotYetImplemented {
			log.Warnf(log.SyncMgr, "Failed to get %s ticker. Error: %s",
				protocol,
				err)
			return
		}
		log.Errorf(log.SyncMgr, "Failed to get %s ticker. Error: %s",
			protocol,
			err)
		return
	}

	// ignoring error as not all tickers have volume populated and error is not actionable
	_ = stats.Add(result.ExchangeName, result.Pair, result.AssetType, result.Last, result.Volume)

	if result.Pair.Quote.IsFiatCurrency() &&
		result.Pair.Quote != m.fiatDisplayCurrency &&
		!m.fiatDisplayCurrency.IsEmpty() {
		origCurrency := result.Pair.Quote.Upper()
		log.Infof(log.Ticker, "%s %s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f",
			result.ExchangeName,
			protocol,
			m.FormatCurrency(result.Pair),
			strings.ToUpper(result.AssetType.String()),
			printConvertCurrencyFormat(origCurrency, result.Last, m.fiatDisplayCurrency),
			printConvertCurrencyFormat(origCurrency, result.Ask, m.fiatDisplayCurrency),
			printConvertCurrencyFormat(origCurrency, result.Bid, m.fiatDisplayCurrency),
			printConvertCurrencyFormat(origCurrency, result.High, m.fiatDisplayCurrency),
			printConvertCurrencyFormat(origCurrency, result.Low, m.fiatDisplayCurrency),
			result.Volume)
	} else {
		if result.Pair.Quote.IsFiatCurrency() &&
			result.Pair.Quote == m.fiatDisplayCurrency &&
			!m.fiatDisplayCurrency.IsEmpty() {
			log.Infof(log.Ticker, "%s %s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f",
				result.ExchangeName,
				protocol,
				m.FormatCurrency(result.Pair),
				strings.ToUpper(result.AssetType.String()),
				printCurrencyFormat(result.Last, m.fiatDisplayCurrency),
				printCurrencyFormat(result.Ask, m.fiatDisplayCurrency),
				printCurrencyFormat(result.Bid, m.fiatDisplayCurrency),
				printCurrencyFormat(result.High, m.fiatDisplayCurrency),
				printCurrencyFormat(result.Low, m.fiatDisplayCurrency),
				result.Volume)
		} else {
			log.Infof(log.Ticker, "%s %s %s %s: TICKER: Last %.8f Ask %.8f Bid %.8f High %.8f Low %.8f Volume %.8f",
				result.ExchangeName,
				protocol,
				m.FormatCurrency(result.Pair),
				strings.ToUpper(result.AssetType.String()),
				result.Last,
				result.Ask,
				result.Bid,
				result.High,
				result.Low,
				result.Volume)
		}
	}
}

// FormatCurrency is a method that formats and returns a currency pair
// based on the user currency display preferences
func (m *syncManager) FormatCurrency(p currency.Pair) currency.Pair {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return p
	}
	return p.Format(m.delimiter, m.uppercase)
}

const (
	book = "%s %s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s"
)

// PrintOrderbookSummary outputs orderbook results
func (m *syncManager) PrintOrderbookSummary(result *orderbook.Base, protocol string, err error) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return
	}
	if err != nil {
		if result == nil {
			log.Errorf(log.OrderBook, "Failed to get %s orderbook. Error: %s",
				protocol,
				err)
			return
		}
		if err == common.ErrNotYetImplemented {
			log.Warnf(log.OrderBook, "Failed to get %s orderbook for %s %s %s. Error: %s",
				protocol,
				result.Exchange,
				result.Pair,
				result.Asset,
				err)
			return
		}
		log.Errorf(log.OrderBook, "Failed to get %s orderbook for %s %s %s. Error: %s",
			protocol,
			result.Exchange,
			result.Pair,
			result.Asset,
			err)
		return
	}

	bidsAmount, bidsValue := result.TotalBidsAmount()
	asksAmount, asksValue := result.TotalAsksAmount()

	var bidValueResult, askValueResult string
	switch {
	case result.Pair.Quote.IsFiatCurrency() && result.Pair.Quote != m.fiatDisplayCurrency && !m.fiatDisplayCurrency.IsEmpty():
		origCurrency := result.Pair.Quote.Upper()
		bidValueResult = printConvertCurrencyFormat(origCurrency, bidsValue, m.fiatDisplayCurrency)
		askValueResult = printConvertCurrencyFormat(origCurrency, asksValue, m.fiatDisplayCurrency)
	case result.Pair.Quote.IsFiatCurrency() && result.Pair.Quote == m.fiatDisplayCurrency && !m.fiatDisplayCurrency.IsEmpty():
		bidValueResult = printCurrencyFormat(bidsValue, m.fiatDisplayCurrency)
		askValueResult = printCurrencyFormat(asksValue, m.fiatDisplayCurrency)
	default:
		bidValueResult = strconv.FormatFloat(bidsValue, 'f', -1, 64)
		askValueResult = strconv.FormatFloat(asksValue, 'f', -1, 64)
	}

	log.Infof(log.OrderBook, book,
		result.Exchange,
		protocol,
		m.FormatCurrency(result.Pair),
		strings.ToUpper(result.Asset.String()),
		len(result.Bids),
		bidsAmount,
		result.Pair.Base,
		bidValueResult,
		len(result.Asks),
		asksAmount,
		result.Pair.Base,
		askValueResult,
	)
}

// WaitForInitialSync allows for a routine to wait for an initial sync to be
// completed without exposing the underlying type. This needs to be called in a
// separate routine.
func (m *syncManager) WaitForInitialSync() error {
	if m == nil {
		return fmt.Errorf("sync manager %w", ErrNilSubsystem)
	}

	m.inService.Wait()
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("sync manager %w", ErrSubSystemNotStarted)
	}

	m.initSyncWG.Wait()
	return nil
}

func relayWebsocketEvent(result interface{}, event, assetType, exchangeName string) {
	evt := WebsocketEvent{
		Data:      result,
		Event:     event,
		AssetType: assetType,
		Exchange:  exchangeName,
	}
	err := BroadcastWebsocketMessage(evt)
	if !errors.Is(err, ErrWebsocketServiceNotRunning) {
		log.Errorf(log.APIServerMgr, "Failed to broadcast websocket event %v. Error: %s",
			event, err)
	}
}
