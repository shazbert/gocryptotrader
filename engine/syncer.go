package engine

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// const holds the sync item types
const (
	SyncItemTicker = iota
	SyncItemOrderbook
	SyncItemTrade
	SyncItemHistoryTrade

	defaultSyncerWorkers = 30
	defaultSyncerTimeout = time.Second * 15
)

var (
	createdCounter = 0
	removedCounter = 0
)

// NewSyncManager returns a new configured SyncManager
func NewSyncManager(c SyncConfig) (*SyncManager, error) {
	if !c.SyncOrderbook &&
		!c.SyncTicker &&
		!c.SyncTrades &&
		!c.SyncHistoricTrades {
		return nil, errors.New("no sync items enabled")
	}

	if c.NumOfWorkers <= 0 {
		log.Warnf(log.SyncMgr,
			"Invalid sync worker amount defaulting to %d",
			defaultSyncerWorkers)
		c.NumOfWorkers = defaultSyncerWorkers
	}

	s := &SyncManager{Config: c, tickerBatchLastRequested: make(map[string]time.Time)}

	log.Debugf(log.SyncMgr,
		"Exchange currency pair syncer config: continuous: %v ticker: %v"+
			" orderbook: %v trades: %v historicTrades:%v workers: %v verbose: %v\n",
		s.Config.SyncContinuously,
		s.Config.SyncTicker,
		s.Config.SyncOrderbook,
		s.Config.SyncTrades,
		s.Config.SyncHistoricTrades,
		s.Config.NumOfWorkers,
		s.Config.Verbose)
	return s, nil
}

// Start starts an exchange currency pair syncer
func (e *SyncManager) Start() {
	log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer started.")

	for x := range Bot.Exchanges {
		if !Bot.Exchanges[x].IsEnabled() {
			continue
		}

		b := Bot.Exchanges[x].GetBase()

		for y := range b.CurrencyPairs.AssetTypes {
			pairs := b.CurrencyPairs.GetPairs(b.CurrencyPairs.AssetTypes[y], true)
			for z := range pairs {
				e.LoadSyncAgent(b.Name,
					pairs[z],
					b.CurrencyPairs.AssetTypes[y],
					b.Features)
			}
		}
	}

	log.Debugf(log.SyncMgr,
		"Sync Manager: Initial sync started. %d items to process.\n",
		createdCounter)

	e.initSyncStartTime = time.Now()

	for i := 0; i < e.Config.NumOfWorkers; i++ {
		go e.worker()
	}

	go func() {
		e.initSync.Wait()

		log.Debugln(log.SyncMgr, "Sync Manager: Initial sync is complete.")
		log.Debugf(log.SyncMgr,
			"Sync Manager: Initial sync took %v [%v sync items].\n",
			time.Now().Sub(e.initSyncStartTime),
			createdCounter)

		if !e.Config.SyncContinuously {
			log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopping.")
			e.Stop()
			Bot.Stop()
			return
		}
	}()
}

// LoadSyncAgent derives a lovely sync agent
func (e *SyncManager) LoadSyncAgent(exchangeName string, p currency.Pair, a asset.Item, f *protocol.Features) {
	c := &SyncAgent{
		AssetType: a,
		Exchange:  exchangeName,
		Pair:      p,
		Features:  f,
	}

	if e.Config.SyncTicker {
		c.Ticker = &SyncBase{}
	}

	if e.Config.SyncOrderbook {
		c.Orderbook = &SyncBase{}
	}

	if e.Config.SyncTrades {
		c.Trade = &SyncBase{}
	}

	if e.Config.SyncHistoricTrades {
		c.HistoricTrade = &SyncBase{}
	}

	e.add(c)
}

// Stop shuts down the exchange currency pair syncer
func (e *SyncManager) Stop() {
	stopped := atomic.CompareAndSwapInt32(&e.shutdown, 0, 1)
	if stopped {
		log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopped.")
	} else {
		// already shut down mate
	}
}

func (e *SyncManager) get(exchangeName string, p currency.Pair, a asset.Item) (*SyncAgent, error) {
	e.RLock()
	defer e.RUnlock()
	for x := range e.SyncAgents {
		if e.SyncAgents[x].Exchange == exchangeName &&
			e.SyncAgents[x].Pair.Equal(p) &&
			e.SyncAgents[x].AssetType == a {
			return e.SyncAgents[x], nil
		}
	}
	return nil, errors.New("exchange currency pair syncer not found")
}

func (e *SyncManager) add(c *SyncAgent) {
	e.Lock()
	defer e.Unlock()

	select {
	case <-e.initialChan:
	default:
		pair := FormatCurrency(c.Pair).String()
		ws := c.Features.Websocket.ProtocolSupported()
		rest := c.Features.REST.ProtocolSupported()
		fix := c.Features.Fix.ProtocolSupported()
		if e.Config.SyncTicker {
			if e.Config.Verbose {
				log.Debugf(log.SyncMgr,
					"%s: Added ticker sync item %v: using websocket: %v using REST: %v using FIX: %v\n",
					c.Exchange,
					pair,
					ws,
					rest,
					fix)
			}
			e.initSync.Add(1)
			createdCounter++
		}

		if e.Config.SyncOrderbook {
			if e.Config.Verbose {
				log.Debugf(log.SyncMgr,
					"%s: Added orderbook sync item %v: using websocket: %v using REST: %v using FIX: %v\n",
					c.Exchange,
					pair,
					ws,
					rest,
					fix)
			}
			e.initSync.Add(1)
			createdCounter++
		}

		if e.Config.SyncTrades {
			if e.Config.Verbose {
				log.Debugf(log.SyncMgr,
					"%s: Added trade sync item %v: using websocket: %v using REST: %v using FIX: %v\n",
					c.Exchange,
					pair,
					ws,
					rest,
					fix)
			}
			e.initSync.Add(1)
			createdCounter++
		}

		if e.Config.SyncHistoricTrades {
			if e.Config.Verbose {
				log.Debugf(log.SyncMgr,
					"%s: Added historic trade sync item %v: using websocket: %v using REST: %v using FIX: %v\n",
					c.Exchange,
					pair,
					ws,
					rest,
					fix)
			}
			e.initSync.Add(1)
			createdCounter++
		}
	}

	c.Created = time.Now()
	e.SyncAgents = append(e.SyncAgents, c)
}

func (e *SyncManager) remove(c *SyncAgent) {
	e.Lock()
	defer e.Unlock()

	for x := range e.SyncAgents {
		if e.SyncAgents[x].Exchange == c.Exchange &&
			e.SyncAgents[x].Pair.Equal(c.Pair) &&
			e.SyncAgents[x].AssetType == c.AssetType {
			e.SyncAgents = append(e.SyncAgents[:x], e.SyncAgents[x+1:]...)
			return
		}
	}
}

func (e *SyncManager) worker() {
	defer log.Debugln(log.SyncMgr,
		"Exchange CurrencyPairSyncer worker shutting down.")

	for atomic.LoadInt32(&e.shutdown) != 1 {
		for x := range Bot.Exchanges {
			if !Bot.Exchanges[x].IsEnabled() {
				continue
			}

			exchangeName := Bot.Exchanges[x].GetName()
			assetTypes := Bot.Exchanges[x].GetAssetTypes()
			// supportsREST := Bot.Exchanges[x].SupportsREST()
			// supportsRESTTickerBatching := Bot.Exchanges[x].SupportsRESTTickerBatchUpdates()

			var switchedToRest bool

			_, usingWebsocket := e.GetFunctionality(Bot.Exchanges[x])

			for y := range assetTypes {
				for _, p := range Bot.Exchanges[x].GetEnabledPairs(assetTypes[y]) {
					if atomic.LoadInt32(&e.shutdown) == 1 {
						return
					}

					e.LoadSyncAgent(exchangeName,
						p,
						assetTypes[y],
						&protocol.Features{})

					c, err := e.get(exchangeName, p, assetTypes[y])
					if err != nil {
						log.Errorf(log.SyncMgr, "failed to get item. Err: %s\n", err)
						continue
					}

					if switchedToRest && usingWebsocket {
						log.Infof(log.SyncMgr,
							"%s %s: Websocket re-enabled, switching from rest to websocket\n",
							c.Exchange,
							FormatCurrency(p).String())
						switchedToRest = false
					}
					if e.Config.SyncTicker {
						if atomic.LoadInt32(&e.SyncAgents[x].Ticker.IsProcessing) == 0 {
							if c.Ticker.LastUpdated.IsZero() ||
								time.Since(c.Ticker.LastUpdated) > defaultSyncerTimeout {
								// if c.Ticker.IsUsingWebsocket {
								// 	if time.Since(c.Created) < defaultSyncerTimeout {
								// 		continue
								// 	}

								// 	if supportsREST {
								// 		atomic.StoreInt32(&e.SyncAgents[x].Ticker.IsProcessing, 1)
								// 		c.Ticker.IsUsingWebsocket = false
								// 		c.Ticker.IsUsingREST = true
								// 		log.Warnf(log.SyncMgr,
								// 			"%s %s: No ticker update after 10 seconds, switching from websocket to rest\n",
								// 			c.Exchange, FormatCurrency(p).String())
								// 		switchedToRest = true
								// 		atomic.StoreInt32(&e.SyncAgents[x].Ticker.IsProcessing, 0)
								// 	}
								// }

								// if c.Ticker.IsUsingREST {
								// 	atomic.StoreInt32(&e.SyncAgents[x].Ticker.IsProcessing, 1)
								// 	var result ticker.Price
								// 	var err error

								// 	if supportsRESTTickerBatching {
								// 		e.Lock()
								// 		batchLastDone, ok := e.tickerBatchLastRequested[exchangeName]
								// 		if !ok {
								// 			e.tickerBatchLastRequested[exchangeName] = time.Time{}
								// 		}
								// 		e.Unlock()

								// 		if batchLastDone.IsZero() ||
								// 			time.Since(batchLastDone) > defaultSyncerTimeout {
								// 			e.Lock()
								// 			if e.Config.Verbose {
								// 				log.Debugf(log.SyncMgr,
								// 					"%s Init'ing REST ticker batching\n",
								// 					exchangeName)
								// 			}
								// 			result, err = Bot.Exchanges[x].UpdateTicker(c.Pair, c.AssetType)
								// 			e.tickerBatchLastRequested[exchangeName] = time.Now()
								// 			e.Unlock()
								// 		} else {
								// 			if e.Config.Verbose {
								// 				log.Debugf(log.SyncMgr,
								// 					"%s Using recent batching cache\n",
								// 					exchangeName)
								// 			}
								// 			result, err = Bot.Exchanges[x].FetchTicker(c.Pair, c.AssetType)
								// 		}
								// 	} else {
								// 		result, err = Bot.Exchanges[x].UpdateTicker(c.Pair, c.AssetType)
								// 	}
								// 	printTickerSummary(&result,
								// 		c.Pair,
								// 		c.AssetType,
								// 		exchangeName,
								// 		err)
								// 	if err == nil {
								// 		//nolint:gocritic Bot.CommsRelayer.StageTickerData(exchangeName, c.AssetType, result)
								// 		if Bot.Config.RemoteControl.WebsocketRPC.Enabled {
								// 			relayWebsocketEvent(result, "ticker_update", c.AssetType.String(), exchangeName)
								// 		}
								// 	}
								// 	e.update(c.Exchange,
								// 		c.Pair,
								// 		c.AssetType,
								// 		SyncItemTicker,
								// 		err)
								// }
							} else {
								time.Sleep(time.Millisecond * 50)
							}
						}
					}

					if e.Config.SyncOrderbook {
						if atomic.LoadInt32(&e.SyncAgents[x].Orderbook.IsProcessing) == 0 {
							if c.Orderbook.LastUpdated.IsZero() ||
								time.Since(c.Orderbook.LastUpdated) > defaultSyncerTimeout {
								// if c.Orderbook.IsUsingWebsocket {
								// 	if time.Since(c.Created) < defaultSyncerTimeout {
								// 		continue
								// 	}
								// 	if supportsREST {
								// 		atomic.StoreInt32(&e.SyncAgents[x].Orderbook.IsProcessing, 1)
								// 		c.Orderbook.IsUsingWebsocket = false
								// 		c.Orderbook.IsUsingREST = true
								// 		log.Warnf(log.SyncMgr,
								// 			"%s %s: No orderbook update after 15 seconds, switching from websocket to rest\n",
								// 			c.Exchange, FormatCurrency(c.Pair).String())
								// 		switchedToRest = true
								// 		atomic.StoreInt32(&e.SyncAgents[x].Orderbook.IsProcessing, 0)
								// 	}
								// }

								atomic.StoreInt32(&e.SyncAgents[x].Orderbook.IsProcessing, 1)
								result, err := Bot.Exchanges[x].UpdateOrderbook(c.Pair, c.AssetType)
								printOrderbookSummary(&result,
									c.Pair,
									c.AssetType,
									exchangeName,
									err)
								if err == nil {
									//nolint:gocritic Bot.CommsRelayer.StageOrderbookData(exchangeName, c.AssetType, result)
									if Bot.Config.RemoteControl.WebsocketRPC.Enabled {
										relayWebsocketEvent(result, "orderbook_update", c.AssetType.String(), exchangeName)
									}
								}
								e.update(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, err)
							} else {
								time.Sleep(time.Millisecond * 50)
							}
						}

						if e.Config.SyncTrades {
							if atomic.LoadInt32(&e.SyncAgents[x].Trade.IsProcessing) == 0 {
								if c.Trade.LastUpdated.IsZero() ||
									time.Since(c.Trade.LastUpdated) > defaultSyncerTimeout {
									atomic.StoreInt32(&e.SyncAgents[x].Trade.IsProcessing, 1)
									e.update(c.Exchange, c.Pair, c.AssetType, SyncItemTrade, nil)
								}
							}
						}
					}
				}
			}
		}
	}
}

func (e *SyncManager) update(exchangeName string, p currency.Pair, a asset.Item, syncType int, err error) {
	// if atomic.LoadInt32(&e.initSyncStarted) != 1 {
	// 	return
	// }

	switch syncType {
	case SyncItemOrderbook, SyncItemTrade, SyncItemTicker:
		if !e.Config.SyncOrderbook && syncType == SyncItemOrderbook {
			return
		}

		if !e.Config.SyncTicker && syncType == SyncItemTicker {
			return
		}

		if !e.Config.SyncTrades && syncType == SyncItemTrade {
			return
		}
	default:
		log.Warnf(log.SyncMgr,
			"ExchangeCurrencyPairSyncer: unknown sync item %v\n",
			syncType)
		return
	}

	e.Lock()
	defer e.Unlock()

	for x := range e.SyncAgents {
		if e.SyncAgents[x].Exchange == exchangeName &&
			e.SyncAgents[x].Pair.Equal(p) &&
			e.SyncAgents[x].AssetType == a {
			switch syncType {
			case SyncItemTicker:
				// origHadData := e.SyncAgents[x].Ticker.HaveData
				e.SyncAgents[x].Ticker.LastUpdated = time.Now()
				if err != nil {
					e.SyncAgents[x].Ticker.NumErrors++
				}
				e.SyncAgents[x].Ticker.HaveData = true
				atomic.StoreInt32(&e.SyncAgents[x].Ticker.IsProcessing, 0)
				// if atomic.LoadInt32(&e.initSyncCompleted) != 1 && !origHadData {
				// 	removedCounter++
				// 	log.Debugf(log.SyncMgr,
				// 		"%s ticker sync complete %v [%d/%d].\n",
				// 		exchangeName,
				// 		FormatCurrency(p).String(),
				// 		removedCounter,
				// 		createdCounter)
				// 	e.initSyncWG.Done()
				// }

			case SyncItemOrderbook:
				// origHadData := e.SyncAgents[x].Orderbook.HaveData
				e.SyncAgents[x].Orderbook.LastUpdated = time.Now()
				if err != nil {
					e.SyncAgents[x].Orderbook.NumErrors++
				}
				e.SyncAgents[x].Orderbook.HaveData = true
				atomic.StoreInt32(&e.SyncAgents[x].Orderbook.IsProcessing, 0)
				// if atomic.LoadInt32(&e.initSyncCompleted) != 1 && !origHadData {
				// 	removedCounter++
				// 	log.Debugf(log.SyncMgr,
				// 		"%s orderbook sync complete %v [%d/%d].\n",
				// 		exchangeName,
				// 		FormatCurrency(p).String(),
				// 		removedCounter,
				// 		createdCounter)
				// 	e.initSyncWG.Done()
				// }

			case SyncItemTrade:
				// origHadData := e.SyncAgents[x].Trade.HaveData
				e.SyncAgents[x].Trade.LastUpdated = time.Now()
				if err != nil {
					e.SyncAgents[x].Trade.NumErrors++
				}
				e.SyncAgents[x].Trade.HaveData = true
				atomic.StoreInt32(&e.SyncAgents[x].Trade.IsProcessing, 0)
				// if atomic.LoadInt32(&e.initSyncCompleted) != 1 && !origHadData {
				// 	removedCounter++
				// 	log.Debugf(log.SyncMgr,
				// 		"%s trade sync complete %v [%d/%d].\n",
				// 		exchangeName,
				// 		FormatCurrency(p).String(),
				// 		removedCounter,
				// 		createdCounter)
				// 	e.initSyncWG.Done()
				// }
			}
		}
	}
}

// GetFunctionality returns exchange protocol functionality
func (e *SyncManager) GetFunctionality(i exchange.IBotExchange) (websocket bool, rest bool) {
	if i.IsWebsocketEnabled() {
		ws, err := i.GetWebsocket()
		if err != nil {
			log.Errorf(log.SyncMgr,
				"%s failed to get websocket. Err: %s\n",
				i.GetName(),
				err)
			rest = true
		}

		if !ws.IsConnected() && !ws.IsConnecting() {
			go WebsocketDataHandler(ws)

			err = ws.Connect()
			if err != nil {
				log.Errorf(log.SyncMgr,
					"%s websocket failed to connect. Err: %s\n",
					i.GetName(),
					err)
				rest = true
			} else {
				websocket = true
			}
		} else {
			websocket = true
		}
	} else if i.SupportsREST() {
		rest = true
	}
	return
}
