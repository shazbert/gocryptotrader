package engine

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Reset BackTest values to default
func (bt *BackTest) Reset() error {
	if bt == nil {
		return gctcommon.ErrNilPointer
	}
	var err error
	if bt.orderManager != nil {
		err = bt.orderManager.Stop()
		if err != nil {
			return err
		}
	}
	if bt.databaseManager != nil {
		err = bt.databaseManager.Stop()
		if err != nil {
			return err
		}
	}
	err = bt.EventQueue.Reset()
	if err != nil {
		return err
	}
	err = bt.DataHolder.Reset()
	if err != nil {
		return err
	}
	err = bt.Portfolio.Reset()
	if err != nil {
		return err
	}
	err = bt.Statistic.Reset()
	if err != nil {
		return err
	}
	err = bt.Exchange.Reset()
	if err != nil {
		return err
	}
	err = bt.Funding.Reset()
	if err != nil {
		return err
	}
	bt.exchangeManager = nil
	bt.orderManager = nil
	bt.databaseManager = nil
	return nil
}

// RunLive is a proof of concept function that does not yet support multi currency usage
// It tasks by constantly checking for new live datas and running through the list of events
// once new data is processed. It will run until application close event has been received
func (bt *BackTest) RunLive() error {
	if bt.LiveDataHandler == nil {
		return errLiveOnly
	}
	var err error
	if bt.LiveDataHandler.IsRealOrders() {
		err = bt.LiveDataHandler.UpdateFunding(false)
		if err != nil {
			return err
		}
	}
	err = bt.LiveDataHandler.Start()
	if err != nil {
		return err
	}
	bt.wg.Add(1)
	go func() {
		err = bt.liveCheck()
		if err != nil {
			log.Error(common.LiveStrategy, err)
		}
		bt.wg.Done()
	}()

	return nil
}

func (bt *BackTest) liveCheck() error {
	for {
		select {
		case <-bt.shutdown:
			return bt.LiveDataHandler.Stop()
		case <-bt.LiveDataHandler.HasShutdownFromError():
			return bt.Stop()
		case <-bt.LiveDataHandler.HasShutdown():
			return nil
		case <-bt.LiveDataHandler.Updated():
			err := bt.Run()
			if err != nil {
				return err
			}
		}
	}
}

// ExecuteStrategy executes the strategy using the provided configs
func (bt *BackTest) ExecuteStrategy(waitForOfflineCompletion bool) error {
	if bt == nil {
		return gctcommon.ErrNilPointer
	}
	bt.m.Lock()
	if bt.MetaData.DateLoaded.IsZero() {
		bt.m.Unlock()
		return errNotSetup
	}
	if !bt.MetaData.Closed && !bt.MetaData.DateStarted.IsZero() {
		bt.m.Unlock()
		return fmt.Errorf("%w %v %v", errTaskIsRunning, bt.MetaData.ID, bt.MetaData.Strategy)
	}
	if bt.MetaData.Closed {
		bt.m.Unlock()
		return fmt.Errorf("%w %v %v", errAlreadyRan, bt.MetaData.ID, bt.MetaData.Strategy)
	}
	if waitForOfflineCompletion && bt.MetaData.LiveTesting {
		bt.m.Unlock()
		return fmt.Errorf("%w cannot wait for a live task to finish", errCannotHandleRequest)
	}

	bt.MetaData.DateStarted = time.Now()
	liveTesting := bt.MetaData.LiveTesting
	bt.m.Unlock()

	var err error
	switch {
	case waitForOfflineCompletion && !liveTesting:
		err = bt.Run()
		if err != nil {
			log.Error(common.Backtester, err)
		}
		return bt.Stop()
	case !waitForOfflineCompletion && liveTesting:
		return bt.RunLive()
	case !waitForOfflineCompletion && !liveTesting:
		go func() {
			err = bt.Run()
			if err != nil {
				log.Error(common.Backtester, err)
			}
			err = bt.Stop()
			if err != nil {
				log.Error(common.Backtester, err)
			}
		}()
	}
	return nil
}

// Run will iterate over loaded data events
// save them and then handle the event based on its type
func (bt *BackTest) Run() error {
	if bt.MetaData.DateLoaded.IsZero() {
		return errNotSetup
	}

	// doubleNil allows the run function to exit if no new data is detected on a
	// live run.
	var doubleNil bool

eventcheck:
	for {
		events := bt.EventQueue.NextEvents()
		if events != nil {
			doubleNil = false
			err := bt.handleEvents(events)
			if err != nil {
				log.Error(common.Backtester, err)
				fmt.Println("ONSIGNAL ERROR")
			}
			if !bt.hasProcessedAnEvent {
				bt.hasProcessedAnEvent = true
			}
			continue
		}

		if bt.hasShutdown {
			return nil
		}

		if doubleNil {
			if bt.verbose {
				log.Info(common.Backtester, "No new data on second check")
			}
			return nil
		}

		doubleNil = true
		// TODO: Will probably need to combine a slice of handlers into one.
		dataHandlers, err := bt.DataHolder.GetAllData()
		if err != nil {
			return err
		}

		for _, intervalHandlers := range dataHandlers {
			var intervalSpecicEvents []common.Event
			var aligned time.Time
			// NOTE: Intervals should be ascending e.g. 1hr -> 3hr -> 6hr
			// Everything will be aligned functionally on the smallest time
			// interval for charting.
			for x := range intervalHandlers {
				var event common.Event
				if aligned.IsZero() {
					event, err = intervalHandlers[x].Next()
				} else {
					event, err = intervalHandlers[x].NextByTime(aligned)
				}

				if err != nil {
					if errors.Is(err, data.ErrEndOfData) {
						return nil
					}
					return err
				}

				if event == nil {
					if !bt.hasProcessedAnEvent && bt.LiveDataHandler == nil {
						var details data.Details
						details, err = intervalHandlers[x].GetDetails()
						if err != nil {
							return err
						}
						log.Errorf(common.Backtester, "Unable to perform `Next` for %v %v %v %v",
							details.ExchangeName,
							details.Asset,
							details.Pair,
							details.Interval)
					}
					return nil
				}

				aligned = event.GetTime()

				o := event.GetOffset()

				if bt.Strategy.UsingSimultaneousProcessing() && bt.hasProcessedDataAtOffset[o] {
					// only append one event, as simultaneous processing
					// will retrieve all relevant events to process under
					// processSimultaneousDataEvents()
					continue eventcheck // TODO: Rethink this.
				}

				if !bt.hasProcessedDataAtOffset[o] {
					bt.hasProcessedDataAtOffset[o] = true
				}
			}
			// Input block of events
			bt.EventQueue.AppendEvents(intervalSpecicEvents)
		}
	}
}

// handleEvents is the main processor of data for the backtester after data has
// been loaded and Run has appended data events to the queue, handle event will
// process events and add further events to the queue if they are required.
func (bt *BackTest) handleEvents(events []common.Event) error {
	if events == nil {
		return fmt.Errorf("cannot handle event %w", errNilData)
	}

	// NOTE: This will range over first event to dictate event type then if
	// needed will range across the rest of the data.
	for x := range events {
		funds, err := bt.Funding.GetFundingForEvent(events[x]) // TODO: Rethink this.
		if err != nil {
			return err
		}

		switch eType := events[x].(type) { // NOTE: All evnts should be the same type.
		case kline.Event:
			// using kline.Event as signal.Event also matches data.Event
			if bt.Strategy.UsingSimultaneousProcessing() {
				err = bt.processSimultaneousDataEvents()
			} else {
				var conv []data.Event // TODO: Obviously change this
				for y := range events {
					conv = append(conv, events[y].(data.Event))
				}
				err = bt.processDataEvents(conv, funds.FundReleaser())
			}
		case signal.Event:
			err = bt.processSignalEvent(eType, funds.FundReserver())
		case order.Event:
			err = bt.processOrderEvent(eType, funds.FundReleaser())
		case fill.Event:
			err = bt.processFillEvent(eType, funds.FundReleaser())
			if bt.LiveDataHandler != nil {
				// output log data per interval instead of at the end
				result, logErr := bt.Statistic.CreateLog(eType)
				if logErr != nil {
					return logErr
				}
				if err != nil {
					return err
				}
				log.Info(common.LiveStrategy, result)
			}
		default:
			err = fmt.Errorf("handleEvent %w %T received, could not process",
				errUnhandledDatatype,
				events) // TODO: redo
		}
		if err != nil {
			return err
		}

		return bt.Funding.CreateSnapshot(events[x].GetTime()) // TODO: RETHINK THIS.

	}
	return errors.New("no data") // TODO: RETHINK THIS.
}

// processDataEvents will pass the events to the strategy and determine how
// it should be handled
func (bt *BackTest) processDataEvents(events data.Events, funds funding.IFundReleaser) error {

	for x := range events {
		fmt.Println("processing a block of data events:", events[x].GetTime(), events[x].GetInterval()) // TODO: RETHINK

		err := bt.updateStatsForDataEvent(events[x], funds) // TODO: Actually implement
		if err != nil {
			return err
		}
		d, err := bt.DataHolder.GetDataForCurrency(events[0]) // TODO: Actually implement
		if err != nil {
			return err
		}
		signalEvents, err := bt.Strategy.OnSignal(d, bt.Funding, bt.Portfolio)
		if err != nil {
			if errors.Is(err, base.ErrTooMuchBadData) {
				// too much bad data is a severe error and backtesting must cease
				return err
			}
			log.Errorf(common.Backtester, "OnSignal %v", err)
			return nil
		}

		eventBlock := make([]common.Event, len(signalEvents))
		for y := range signalEvents {
			// TODO: Maybe Set multiple signal events for offset?
			err = bt.Statistic.SetEventForOffset(signalEvents[y])
			if err != nil {
				log.Errorf(common.Backtester, "SetEventForOffset %v", err) // Return error?
			}

			eventBlock[y] = signalEvents[y]
		}

		bt.EventQueue.AppendEvents(eventBlock) // RETHINK THIS? // Might have an interface that returns the conforming slice?
	}

	return nil
}

// processSimultaneousDataEvents determines what signal events are generated and appended
// to the event queue. It will pass all currency events to the strategy to determine what
// currencies to act upon
func (bt *BackTest) processSimultaneousDataEvents() error {
	dataHolders, err := bt.DataHolder.GetAllData()
	if err != nil {
		return err
	}

	fullDataEvents := make(data.AssetSegregated, 0, len(dataHolders))
events:
	for _, intervalHolder := range dataHolders {
		intervalEvents := make([]data.Handler, 0, len(intervalHolder))
		for x := range intervalHolder {
			var latestData data.Event
			latestData, err = intervalHolder[x].Latest()
			if err != nil {
				return err
			}
			var funds funding.IFundingPair
			funds, err = bt.Funding.GetFundingForEvent(latestData)
			if err != nil {
				return err
			}
			err = bt.updateStatsForDataEvent(latestData, funds.FundReleaser())
			if err != nil {
				switch {
				case errors.Is(err, statistics.ErrAlreadyProcessed):
					if !bt.MetaData.Closed || !bt.MetaData.ClosePositionsOnStop {
						// Closing positions on close reuses existing events and doesn't need to be logged
						// any other scenario, this should be logged
						log.Warnf(common.LiveStrategy, "%v %v", latestData.GetOffset(), err)
					}
					continue events // TODO: RETHINK THIS
				case errors.Is(err, gctorder.ErrPositionLiquidated):
					return nil // TODO: What about other positions?
				default:
					log.Error(common.Backtester, err)
				}
			}
			intervalEvents = append(intervalEvents, intervalHolder[x])
		}
		fullDataEvents = append(fullDataEvents, intervalEvents)
	}

	assetSignals, err := bt.Strategy.OnSimultaneousSignals(fullDataEvents, bt.Funding, bt.Portfolio)
	if err != nil {
		switch {
		case errors.Is(err, base.ErrTooMuchBadData):
			// too much bad data is a severe error and backtesting must cease
			return err
		case errors.Is(err, base.ErrNoDataToProcess) && bt.MetaData.Closed && bt.MetaData.ClosePositionsOnStop:
			// event queue is being cleared with no data events to process
			return nil
		default:
			log.Errorf(common.Backtester, "OnSimultaneousSignals %v", err) // Return error?
			return nil
		}
	}
	for i := range assetSignals {
		for j := range assetSignals[i] {
			err = bt.Statistic.SetEventForOffset(assetSignals[i][j])
			if err != nil {
				log.Errorf(common.Backtester, "SetEventForOffset %v %v %v %v %v",
					assetSignals[i][j].GetExchange(),
					assetSignals[i][j].GetAssetType(),
					assetSignals[i][j].Pair(),
					assetSignals[i][j].GetInterval(),
					err)
				// continue TODO: ????
			}
			bt.EventQueue.AppendEvents([]common.Event{assetSignals[i][j]})
		}
	}
	return nil
}

// updateStatsForDataEvent makes various systems aware of price movements from
// data events
func (bt *BackTest) updateStatsForDataEvent(ev data.Event, funds funding.IFundReleaser) error {
	if ev == nil {
		return common.ErrNilEvent
	}
	if funds == nil {
		return fmt.Errorf("%v %v %v %w missing fund releaser", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), gctcommon.ErrNilPointer)
	}
	// update statistics with the latest price
	err := bt.Statistic.SetEventForOffset(ev)
	if err != nil {
		if errors.Is(err, statistics.ErrAlreadyProcessed) {
			return err
		}
		log.Errorf(common.Backtester, "SetEventForOffset %v", err)
	}
	// update portfolio manager with the latest price
	err = bt.Portfolio.UpdateHoldings(ev, funds)
	if err != nil {
		log.Errorf(common.Backtester, "UpdateHoldings %v", err)
	}

	if ev.GetAssetType().IsFutures() {
		var cr funding.ICollateralReleaser
		cr, err = funds.CollateralReleaser()
		if err != nil {
			return err
		}

		err = bt.Portfolio.UpdatePNL(ev, ev.GetClosePrice())
		if err != nil {
			if errors.Is(err, gctorder.ErrPositionNotFound) {
				// if there is no position yet, there's nothing to update
				return nil
			}
			if !errors.Is(err, gctorder.ErrPositionLiquidated) {
				return fmt.Errorf("UpdatePNL %v", err)
			}
		}
		var pnl *portfolio.PNLSummary
		pnl, err = bt.Portfolio.GetLatestPNLForEvent(ev)
		if err != nil {
			return err
		}

		if pnl.Result.IsLiquidated {
			return nil
		}
		if bt.LiveDataHandler == nil || (bt.LiveDataHandler != nil && !bt.LiveDataHandler.IsRealOrders()) {
			err = bt.Portfolio.CheckLiquidationStatus(ev, cr, pnl)
			if err != nil {
				if errors.Is(err, gctorder.ErrPositionLiquidated) {
					liquidErr := bt.triggerLiquidationsForExchange(ev, pnl)
					if liquidErr != nil {
						return liquidErr
					}
				}
				return err
			}
		}

		return bt.Statistic.AddPNLForTime(pnl)
	}

	return nil
}

// processSignalEvent receives an event from the strategy for processing under the portfolio
func (bt *BackTest) processSignalEvent(ev signal.Event, funds funding.IFundReserver) error {
	if ev == nil {
		return common.ErrNilEvent
	}
	if funds == nil {
		return fmt.Errorf("%w funds", gctcommon.ErrNilPointer)
	}
	cs, err := bt.Exchange.GetCurrencySettings(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	if err != nil {
		log.Errorf(common.Backtester, "GetCurrencySettings %v", err) // TODO: Need to log? RM?
		return fmt.Errorf("GetCurrencySettings %w", err)
	}
	var o *order.Order
	o, err = bt.Portfolio.OnSignal(ev, &cs, funds)
	if err != nil {
		log.Errorf(common.Backtester, "OnSignal %v", err) // <--- ?
		return fmt.Errorf("OnSignal %v %v %v %v %w", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), ev.GetInterval(), err)
	}
	err = bt.Statistic.SetEventForOffset(o)
	if err != nil {
		return fmt.Errorf("SetEventForOffset %w", err)
	}

	bt.EventQueue.AppendEvents([]common.Event{o}) // TODO: ?
	return nil
}

func (bt *BackTest) processOrderEvent(ev order.Event, funds funding.IFundReleaser) error {
	if ev == nil {
		return common.ErrNilEvent
	}
	if funds == nil {
		return fmt.Errorf("%w funds", gctcommon.ErrNilPointer)
	}
	d, err := bt.DataHolder.GetDataForCurrency(ev)
	if err != nil {
		return err
	}
	f, err := bt.Exchange.ExecuteOrder(ev, d[0], bt.orderManager, funds) // TODO: Correctly implement, Lowest interval.
	if err != nil {
		if f == nil {
			log.Errorf(common.Backtester, "ExecuteOrder fill event should always be returned, please fix, %v", err)
			return fmt.Errorf("ExecuteOrder fill event should always be returned, please fix, %v", err)
		}
		if !errors.Is(err, exchange.ErrCannotTransact) {
			log.Errorf(common.Backtester, "ExecuteOrder %v %v %v %v", f.GetExchange(), f.GetAssetType(), f.Pair(), err)
		}
	}
	err = bt.Statistic.SetEventForOffset(f)
	if err != nil {
		log.Errorf(common.Backtester, "SetEventForOffset %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}
	bt.EventQueue.AppendEvents([]common.Event{f}) // TODO: Variadic param?
	return nil
}

func (bt *BackTest) processFillEvent(ev fill.Event, funds funding.IFundReleaser) error {
	_, err := bt.Portfolio.OnFill(ev, funds)
	if err != nil {
		return fmt.Errorf("OnFill %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}
	err = bt.Funding.UpdateCollateralForEvent(ev, false)
	if err != nil {
		return fmt.Errorf("UpdateCollateralForEvent %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}
	holding, err := bt.Portfolio.ViewHoldingAtTimePeriod(ev)
	if err != nil {
		log.Error(common.Backtester, err)
	}
	err = bt.Statistic.AddHoldingsForTime(holding)
	if err != nil {
		log.Errorf(common.Backtester, "AddHoldingsForTime %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}

	snap, err := bt.Portfolio.GetLatestComplianceSnapshot(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	if err != nil {
		log.Errorf(common.Backtester, "GetLatestComplianceSnapshot %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}
	err = bt.Statistic.AddComplianceSnapshotForTime(snap, ev)
	if err != nil {
		log.Errorf(common.Backtester, "AddComplianceSnapshotForTime %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}

	fde := ev.GetFillDependentEvent()
	if fde != nil && !fde.IsNil() {
		// some events can only be triggered on a successful fill event
		fde.SetOffset(ev.GetOffset())
		err = bt.Statistic.SetEventForOffset(fde)
		if err != nil {
			log.Errorf(common.Backtester, "SetEventForOffset %v %v %v %v %v",
				fde.GetExchange(),
				fde.GetAssetType(),
				fde.Pair(),
				fde.GetInterval(),
				err)
			// TODO: Return and above on error?
		}
		od := ev.GetOrder()
		if fde.MatchOrderAmount() && od != nil {
			fde.SetAmount(ev.GetAmount())
		}
		fde.AppendReasonf("raising event after %v %v %v %v fill",
			ev.GetExchange(),
			ev.GetAssetType(),
			ev.Pair(),
			ev.GetInterval())
		bt.EventQueue.AppendEvents([]common.Event{fde}) // <---- WHAT!????
	}
	if ev.GetAssetType().IsFutures() {
		return bt.processFuturesFillEvent(ev, funds)
	}

	return nil
}

func (bt *BackTest) processFuturesFillEvent(ev fill.Event, funds funding.IFundReleaser) error {
	if ev.GetOrder() == nil {
		return nil
	}
	pnl, err := bt.Portfolio.TrackFuturesOrder(ev, funds)
	if err != nil && !errors.Is(err, gctorder.ErrSubmissionIsNil) {
		return fmt.Errorf("TrackFuturesOrder %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}

	var exch gctexchange.IBotExchange
	exch, err = bt.exchangeManager.GetExchangeByName(ev.GetExchange())
	if err != nil {
		return fmt.Errorf("GetExchangeByName %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}

	rPNL := pnl.GetRealisedPNL()
	if !rPNL.PNL.IsZero() {
		var receivingCurrency currency.Code
		var receivingAsset asset.Item
		receivingCurrency, receivingAsset, err = exch.GetCurrencyForRealisedPNL(ev.GetAssetType(), ev.Pair())
		if err != nil {
			return fmt.Errorf("GetCurrencyForRealisedPNL %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
		}
		err = bt.Funding.RealisePNL(ev.GetExchange(), receivingAsset, receivingCurrency, rPNL.PNL)
		if err != nil {
			return fmt.Errorf("RealisePNL %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
		}
	}

	err = bt.Statistic.AddPNLForTime(pnl)
	if err != nil {
		return fmt.Errorf("AddPNLForTime %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}
	err = bt.Funding.UpdateCollateralForEvent(ev, false)
	if err != nil {
		return fmt.Errorf("UpdateCollateralForEvent %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
	}
	return nil
}

// Stop shuts down the live data loop
func (bt *BackTest) Stop() error {
	if bt == nil {
		return gctcommon.ErrNilPointer
	}
	bt.m.Lock()
	defer bt.m.Unlock()
	if bt.MetaData.Closed {
		return errAlreadyRan
	}
	close(bt.shutdown)
	bt.MetaData.Closed = true
	bt.MetaData.DateEnded = time.Now()
	if bt.MetaData.ClosePositionsOnStop {
		err := bt.CloseAllPositions()
		if err != nil {
			log.Errorf(common.Backtester, "Could not close all positions on stop: %s", err)
		}
	}
	err := bt.Statistic.CalculateAllResults()
	if err != nil {
		return err
	}
	err = bt.Reports.GenerateReport()
	if err != nil {
		return err
	}
	return nil
}

func (bt *BackTest) triggerLiquidationsForExchange(ev data.Event, pnl *portfolio.PNLSummary) error {
	if ev == nil {
		return common.ErrNilEvent
	}
	if pnl == nil {
		return fmt.Errorf("%w pnl summary", gctcommon.ErrNilPointer)
	}
	orders, err := bt.Portfolio.CreateLiquidationOrdersForExchange(ev, bt.Funding)
	if err != nil {
		return err
	}
	for i := range orders {
		// these orders are raising events for event offsets
		// which may not have been processed yet
		// this will create and store stats for each order
		// then liquidate it at the funding level
		var datas []data.Handler
		datas, err = bt.DataHolder.GetDataForCurrency(orders[i])
		if err != nil {
			return err
		}
		var latest data.Event
		latest, err = datas[0].Latest() // TODO: More correctly implemented. This should be the smallest time scale.
		if err != nil {
			return err
		}
		err = bt.Statistic.SetEventForOffset(latest)
		if err != nil && !errors.Is(err, statistics.ErrAlreadyProcessed) {
			return err
		}
		bt.EventQueue.AppendEvents([]common.Event{orders[i]})
		err = bt.Statistic.SetEventForOffset(orders[i])
		if err != nil {
			log.Errorf(common.Backtester, "SetEventForOffset %v %v %v %v", ev.GetExchange(), ev.GetAssetType(), ev.Pair(), err)
		}
		err = bt.Funding.Liquidate(orders[i])
		if err != nil {
			return err
		}
	}
	pnl.Result.IsLiquidated = true
	pnl.Result.Status = gctorder.Liquidated
	return bt.Statistic.AddPNLForTime(pnl)
}

// CloseAllPositions will close sell any positions held on closure
// can only be with live testing and where a strategy supports it
func (bt *BackTest) CloseAllPositions() error {
	if bt.LiveDataHandler == nil {
		return errLiveOnly
	}
	err := bt.LiveDataHandler.UpdateFunding(true)
	if err != nil {
		return err
	}
	dataHolders, err := bt.DataHolder.GetAllData()
	if err != nil {
		return err
	}
	latestPrices := make([]data.Event, len(dataHolders))
	for i := range dataHolders {
		var latest data.Event
		latest, err = dataHolders[0][i].Latest() // TODO: Correctly implement. This should be the smallest time scale.
		if err != nil {
			return err
		}
		latestPrices[i] = latest
	}
	events, err := bt.Strategy.CloseAllPositions(bt.Portfolio.GetLatestHoldingsForAllCurrencies(), latestPrices)
	if err != nil {
		if errors.Is(err, gctcommon.ErrFunctionNotSupported) {
			log.Warnf(common.LiveStrategy, "Closing all positions is not supported by strategy %v", bt.Strategy.Name())
			return nil
		}
		return err
	}
	if len(events) == 0 {
		return nil
	}
	err = bt.LiveDataHandler.SetDataForClosingAllPositions(events...)
	if err != nil {
		return err
	}
	for i := range events {
		k := events[i].ToKline()
		err = bt.Statistic.SetEventForOffset(k)
		if err != nil {
			return err
		}
		bt.EventQueue.AppendEvents([]common.Event{events[i]})
	}
	err = bt.Run()
	if err != nil {
		return err
	}

	err = bt.LiveDataHandler.UpdateFunding(true)
	if err != nil {
		return err
	}

	err = bt.Funding.CreateSnapshot(events[0].GetTime())
	if err != nil {
		return err
	}
	for i := range events {
		var funds funding.IFundingPair
		funds, err = bt.Funding.GetFundingForEvent(events[i])
		if err != nil {
			return err
		}
		err = bt.Portfolio.SetHoldingsForEvent(funds.FundReader(), events[i])
		if err != nil {
			return err
		}
	}
	her := bt.Portfolio.GetLatestHoldingsForAllCurrencies()
	for i := range her {
		err = bt.Statistic.AddHoldingsForTime(&her[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// GenerateSummary creates a summary of a strategy task
// this summary contains many details of a task
func (bt *BackTest) GenerateSummary() (*TaskSummary, error) {
	if bt == nil {
		return nil, gctcommon.ErrNilPointer
	}
	bt.m.Lock()
	defer bt.m.Unlock()
	return &TaskSummary{
		MetaData: bt.MetaData,
	}, nil
}

// SetupMetaData will populate metadata fields
func (bt *BackTest) SetupMetaData() error {
	if bt == nil {
		return gctcommon.ErrNilPointer
	}
	bt.m.Lock()
	defer bt.m.Unlock()
	if !bt.MetaData.ID.IsNil() && !bt.MetaData.DateLoaded.IsZero() {
		// already setup
		return nil
	}
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	bt.MetaData.ID = id
	bt.MetaData.DateLoaded = time.Now()
	return nil
}

// IsRunning checks if the task is running
func (bt *BackTest) IsRunning() bool {
	if bt == nil {
		return false
	}
	bt.m.Lock()
	defer bt.m.Unlock()
	return !bt.MetaData.DateStarted.IsZero() && !bt.MetaData.Closed
}

// HasRan checks if the task has been executed
func (bt *BackTest) HasRan() bool {
	if bt == nil {
		return false
	}
	bt.m.Lock()
	defer bt.m.Unlock()
	return bt.MetaData.Closed
}

// Equal checks if the incoming task matches
func (bt *BackTest) Equal(bt2 *BackTest) bool {
	if bt == nil || bt2 == nil {
		return false
	}
	bt.m.Lock()
	btM := bt.MetaData
	bt.m.Unlock()
	// if they are actually the same pointer
	// locks must be handled separately
	bt2.m.Lock()
	btM2 := bt2.MetaData
	bt2.m.Unlock()
	return btM == btM2
}

// MatchesID checks if the backtesting run's ID matches the supplied
func (bt *BackTest) MatchesID(id uuid.UUID) bool {
	if bt == nil {
		return false
	}
	if id.IsNil() {
		return false
	}
	bt.m.Lock()
	defer bt.m.Unlock()
	if bt.MetaData.ID.IsNil() {
		return false
	}
	return bt.MetaData.ID == id
}
