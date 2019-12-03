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

	DefaultSyncerWorkers = 15
	DefaultSyncerTimeout = time.Second * 15
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

	if c.NumWorkers <= 0 {
		c.NumWorkers = DefaultSyncerWorkers
	}

	if c.SyncTimeout <= time.Duration(0) {
		c.SyncTimeout = DefaultSyncerTimeout
	}

	s := ExchangeCurrencyPairSyncer{
		Cfg: CurrencyPairSyncerConfig{
			SyncTicker:       c.SyncTicker,
			SyncOrderbook:    c.SyncOrderbook,
			SyncTrades:       c.SyncTrades,
			SyncContinuously: c.SyncContinuously,
			SyncTimeout:      c.SyncTimeout,
			NumWorkers:       c.NumWorkers,
		},
	}

	s.tickerBatchLastRequested = make(map[string]time.Time)

	log.Debugf(log.SyncMgr,
		"Exchange currency pair syncer config: continuous: %v ticker: %v"+
			" orderbook: %v trades: %v workers: %v verbose: %v timeout: %v\n",
		s.Cfg.SyncContinuously, s.Cfg.SyncTicker, s.Cfg.SyncOrderbook,
		s.Cfg.SyncTrades, s.Cfg.NumWorkers, s.Cfg.Verbose, s.Cfg.SyncTimeout)
	return &s, nil
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
					if e.Cfg.SyncTicker {
						if !e.isProcessing(exchangeName, c.Pair, c.AssetType, SyncItemTicker) {
							if c.Ticker.LastUpdated.IsZero() || time.Since(c.Ticker.LastUpdated) > e.Cfg.SyncTimeout {
								if c.Ticker.IsUsingWebsocket {
									if time.Since(c.Created) < e.Cfg.SyncTimeout {
										continue
									}

									if supportsREST {
										e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, true)
										c.Ticker.IsUsingWebsocket = false
										c.Ticker.IsUsingREST = true
										log.Warnf(log.SyncMgr,
											"%s %s: No ticker update after 10 seconds, switching from websocket to rest\n",
											c.Exchange, FormatCurrency(p).String())
										switchedToRest = true
										e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, false)
									}
								}

								if c.Ticker.IsUsingREST {
									e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, true)
									var result ticker.Price
									var err error

									if supportsRESTTickerBatching {
										e.mux.Lock()
										batchLastDone, ok := e.tickerBatchLastRequested[exchangeName]
										if !ok {
											e.tickerBatchLastRequested[exchangeName] = time.Time{}
										}
										e.mux.Unlock()

										if batchLastDone.IsZero() || time.Since(batchLastDone) > e.Cfg.SyncTimeout {
											e.mux.Lock()
											if e.Cfg.Verbose {
												log.Debugf(log.SyncMgr, "%s Init'ing REST ticker batching\n", exchangeName)
											}
											result, err = Bot.Exchanges[x].UpdateTicker(c.Pair, c.AssetType)
											e.tickerBatchLastRequested[exchangeName] = time.Now()
											e.mux.Unlock()
										} else {
											if e.Cfg.Verbose {
												log.Debugf(log.SyncMgr, "%s Using recent batching cache\n", exchangeName)
											}
											result, err = Bot.Exchanges[x].FetchTicker(c.Pair, c.AssetType)
										}
									} else {
										result, err = Bot.Exchanges[x].UpdateTicker(c.Pair, c.AssetType)
									}
									printTickerSummary(&result, c.Pair, c.AssetType, exchangeName, err)
									if err == nil {
										//nolint:gocritic Bot.CommsRelayer.StageTickerData(exchangeName, c.AssetType, result)
										if Bot.Config.RemoteControl.WebsocketRPC.Enabled {
											relayWebsocketEvent(result, "ticker_update", c.AssetType.String(), exchangeName)
										}
									}
									e.update(c.Exchange, c.Pair, c.AssetType, SyncItemTicker, err)
								}
							} else {
								time.Sleep(time.Millisecond * 50)
							}
						}
					}

					if e.Cfg.SyncOrderbook {
						if !e.isProcessing(exchangeName, c.Pair, c.AssetType, SyncItemOrderbook) {
							if c.Orderbook.LastUpdated.IsZero() || time.Since(c.Orderbook.LastUpdated) > e.Cfg.SyncTimeout {
								if c.Orderbook.IsUsingWebsocket {
									if time.Since(c.Created) < e.Cfg.SyncTimeout {
										continue
									}
									if supportsREST {
										e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, true)
										c.Orderbook.IsUsingWebsocket = false
										c.Orderbook.IsUsingREST = true
										log.Warnf(log.SyncMgr,
											"%s %s: No orderbook update after 15 seconds, switching from websocket to rest\n",
											c.Exchange, FormatCurrency(c.Pair).String())
										switchedToRest = true
										e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, false)
									}
								}

								e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemOrderbook, true)
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
						if e.Cfg.SyncTrades {
							if !e.isProcessing(exchangeName, c.Pair, c.AssetType, SyncItemTrade) {
								if c.Trade.LastUpdated.IsZero() || time.Since(c.Trade.LastUpdated) > e.Cfg.SyncTimeout {
									e.setProcessing(c.Exchange, c.Pair, c.AssetType, SyncItemTrade, true)
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

	if atomic.CompareAndSwapInt32(&e.initSyncStarted, 0, 1) {
		log.Debugf(log.SyncMgr,
			"Exchange CurrencyPairSyncer initial sync started. %d items to process.\n",
			createdCounter)
		e.initSyncStartTime = time.Now()
	}

	go func() {
		e.initSyncWG.Wait()
		if atomic.CompareAndSwapInt32(&e.initSyncCompleted, 0, 1) {
			log.Debugf(log.SyncMgr, "Exchange CurrencyPairSyncer initial sync is complete.\n")
			completedTime := time.Now()
			log.Debugf(log.SyncMgr, "Exchange CurrencyPairSyncer initiial sync took %v [%v sync items].\n",
				completedTime.Sub(e.initSyncStartTime), createdCounter)

			if !e.Cfg.SyncContinuously {
				log.Debugln(log.SyncMgr, "Exchange CurrencyPairSyncer stopping.")
				e.Stop()
				return
			}
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
