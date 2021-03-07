package buffer

import (
	"errors"
	"fmt"
	"sort"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

const packageError = "websocket orderbook buffer error: %w"

var (
	errUnsetExchangeName            = errors.New("exchange name unset")
	errUnsetDataHandler             = errors.New("datahandler unset")
	errIssueBufferEnabledButNoLimit = errors.New("buffer enabled but no limit set")
	errUpdateIsNil                  = errors.New("update is nil")
	errUpdateNoTargets              = errors.New("update bid/ask targets cannot be nil")
)

// Setup sets private variables
func (w *Orderbook) Setup(obBufferLimit int,
	bufferEnabled,
	sortBuffer,
	sortBufferByUpdateIDs,
	updateEntriesByID bool, exchangeName string, dataHandler chan interface{}) error {
	if exchangeName == "" {
		return fmt.Errorf(packageError, errUnsetExchangeName)
	}
	if dataHandler == nil {
		return fmt.Errorf(packageError, errUnsetDataHandler)
	}
	if bufferEnabled && obBufferLimit < 1 {
		return fmt.Errorf(packageError, errIssueBufferEnabledButNoLimit)
	}
	w.obBufferLimit = obBufferLimit
	w.bufferEnabled = bufferEnabled
	w.sortBuffer = sortBuffer
	w.sortBufferByUpdateIDs = sortBufferByUpdateIDs
	w.updateEntriesByID = updateEntriesByID
	w.exchangeName = exchangeName
	w.dataHandler = dataHandler
	w.ob = make(map[currency.Code]map[currency.Code]map[asset.Item]*orderbookHolder)
	return nil
}

// validate validates update against setup values
func (w *Orderbook) validate(u *Update) error {
	if u == nil {
		return fmt.Errorf(packageError, errUpdateIsNil)
	}
	if len(u.Bids) == 0 && len(u.Asks) == 0 {
		return fmt.Errorf(packageError, errUpdateNoTargets)
	}
	return nil
}

// Update updates a local buffer using bid targets and ask targets then updates
// main orderbook
// Volume == 0; deletion at price target
// Price target not found; append of price target
// Price target found; amend volume of price target
func (w *Orderbook) Update(u *Update) error {
	if err := w.validate(u); err != nil {
		return err
	}
	w.m.Lock()
	defer w.m.Unlock()
	obLookup, ok := w.ob[u.Pair.Base][u.Pair.Quote][u.Asset]
	if !ok {
		return fmt.Errorf("ob.Base could not be found for Exchange %s CurrencyPair: %s AssetType: %s",
			w.exchangeName,
			u.Pair,
			u.Asset)
	}

	if w.bufferEnabled {
		processed, err := w.processBufferUpdate(obLookup, u)
		if err != nil {
			return err
		}

		if !processed {
			return nil
		}
	} else {
		err := w.processObUpdate(obLookup, u)
		if err != nil {
			return err
		}
	}

	// Send pointer to orderbook.Depth to datahandler for logging purposes
	select {
	case w.dataHandler <- obLookup.ob:
	default:
	}
	return nil
}

// processBufferUpdate stores update into buffer, when buffer at capacity as
// defined by w.obBufferLimit it well then sort and apply updates.
func (w *Orderbook) processBufferUpdate(o *orderbookHolder, u *Update) (bool, error) {
	*o.buffer = append(*o.buffer, *u)
	if len(*o.buffer) < w.obBufferLimit {
		return false, nil
	}

	if w.sortBuffer {
		// sort by last updated to ensure each update is in order
		if w.sortBufferByUpdateIDs {
			sort.Slice(*o.buffer, func(i, j int) bool {
				return (*o.buffer)[i].UpdateID < (*o.buffer)[j].UpdateID
			})
		} else {
			sort.Slice(*o.buffer, func(i, j int) bool {
				return (*o.buffer)[i].UpdateTime.Before((*o.buffer)[j].UpdateTime)
			})
		}
	}
	for i := range *o.buffer {
		err := w.processObUpdate(o, &(*o.buffer)[i])
		if err != nil {
			return false, err
		}
	}
	// clear buffer of old updates
	*o.buffer = nil
	return true, nil
}

// processObUpdate processes updates either by its corresponding id or by
// price level
func (w *Orderbook) processObUpdate(o *orderbookHolder, u *Update) error {
	// TODO: Check to see if UpdateID assignment is needed and purge from system
	// if not
	// o.ob.LastUpdateID = u.UpdateID
	if w.updateEntriesByID {
		return o.updateByIDAndAction(u, w.exchangeName, false) // TODO: FIX
	}
	return o.updateByPrice(u)
}

// updateByPrice ammends amount if match occurs by price, deletes if amount is
// zero or less and inserts if not found.
func (o *orderbookHolder) updateByPrice(updts *Update) error {
	return o.ob.UpdateBidAskByPrice(updts.Bids, updts.Asks, updts.MaxDepth)
}

// updateByIDAndAction will receive an action to execute against the orderbook
// it will then match by IDs instead of price to perform the action
func (o *orderbookHolder) updateByIDAndAction(updts *Update, exch string, isFundingRate bool) (err error) {
	switch updts.Action {
	case Amend:
		err = o.ob.UpdateBidAskByID(updts.Bids, updts.Asks)
	case Delete:
		// edge case for Bitfinex as their streaming endpoint duplicates deletes
		bypassErr := exch == "Bitfinex" && isFundingRate
		err = o.ob.DeleteBidAskByID(updts.Bids, updts.Asks, bypassErr)
	case Insert:
		o.ob.InsertBidAskByID(updts.Bids, updts.Asks)
	case UpdateInsert:
	// updateBids:
	// 	for x := range updts.Bids {
	// 		for target := range o.ob.Bids { // First iteration finds ID matches
	// 			if o.ob.Bids[target].ID == updts.Bids[x].ID {
	// 				if o.ob.Bids[target].Price != updts.Bids[x].Price {
	// 					// Price change occurred so correct bid alignment is
	// 					// needed - delete instance and insert into correct
	// 					// price level
	// 					o.ob.Bids = append(o.ob.Bids[:target], o.ob.Bids[target+1:]...)
	// 					break
	// 				}
	// 				o.ob.Bids[target].Amount = updts.Bids[x].Amount
	// 				continue updateBids
	// 			}
	// 		}
	// 		insertBid(updts.Bids[x], &o.ob.Bids)
	// 	}
	// updateAsks:
	// for x := range updts.Asks {
	// 	for target := range o.ob.Asks {
	// 		if o.ob.Asks[target].ID == updts.Asks[x].ID {
	// 			if o.ob.Asks[target].Price != updts.Asks[x].Price {
	// 				// Price change occurred so correct ask alignment is
	// 				// needed - delete instance and insert into correct
	// 				// price level
	// 				o.ob.Asks = append(o.ob.Asks[:target], o.ob.Asks[target+1:]...)
	// 				break
	// 			}
	// 			o.ob.Asks[target].Amount = updts.Asks[x].Amount
	// 			continue updateAsks
	// 		}
	// 	}
	// 	insertAsk(updts.Asks[x], &o.ob.Asks)
	// }
	default:
		return fmt.Errorf("invalid action [%s]", updts.Action)
	}
	return
}

// // applyUpdates amends amount by ID and returns an error if not found
// func applyUpdates(updts, book []orderbook.Item) error {
// updates:
// 	for x := range updts {
// 		for y := range book {
// 			if book[y].ID == updts[x].ID {
// 				book[y].Amount = updts[x].Amount
// 				continue updates
// 			}
// 		}
// 		return fmt.Errorf("update cannot be applied id: %d not found",
// 			updts[x].ID)
// 	}
// 	return nil
// }

// deleteUpdates removes updates from orderbook and returns an error if not
// found
func deleteUpdates(updt []orderbook.Item, book *orderbook.Items, bypassErr bool) error {
updates:
	for x := range updt {
		for y := range *book {
			if (*book)[y].ID == updt[x].ID {
				*book = append((*book)[:y], (*book)[y+1:]...) // nolint:gocritic
				continue updates
			}
		}
		// bypassErr is for expected duplication from endpoint.
		if !bypassErr {
			return fmt.Errorf("update cannot be deleted id: %d not found",
				updt[x].ID)
		}
	}
	return nil
}

// func insertAsk(updt orderbook.Item, book *orderbook.Items) {
// 	for target := range *book {
// 		if updt.Price < (*book)[target].Price {
// 			insertItem(updt, book, target)
// 			return
// 		}
// 	}
// 	*book = append(*book, updt)
// }

// func insertBid(updt orderbook.Item, book *orderbook.Items) {
// 	for target := range *book {
// 		if updt.Price > (*book)[target].Price {
// 			insertItem(updt, book, target)
// 			return
// 		}
// 	}
// 	*book = append(*book, updt)
// }

// // insertUpdatesBid inserts on **correctly aligned** book at price level
// func insertUpdatesBid(updt []orderbook.Item, book *orderbook.Items) {
// updates:
// 	for x := range updt {
// 		for target := range *book {
// 			if updt[x].Price > (*book)[target].Price {
// 				insertItem(updt[x], book, target)
// 				continue updates
// 			}
// 		}
// 		*book = append(*book, updt[x])
// 	}
// }

// // insertUpdatesBid inserts on **correctly aligned** book at price level
// func insertUpdatesAsk(updt []orderbook.Item, book *orderbook.Items) {
// updates:
// 	for x := range updt {
// 		for target := range *book {
// 			if updt[x].Price < (*book)[target].Price {
// 				insertItem(updt[x], book, target)
// 				continue updates
// 			}
// 		}
// 		*book = append(*book, updt[x])
// 	}
// }

// // insertItem inserts item in slice by target element this is an optimization
// // to reduce the need for sorting algorithms
// func insertItem(update orderbook.Item, book *orderbook.Items, target int) {
// 	// TODO: extend slice by incoming update length before this gets hit
// 	*book = append(*book, orderbook.Item{})
// 	copy((*book)[target+1:], (*book)[target:])
// 	(*book)[target] = update
// }

// LoadSnapshot loads initial snapshot of ob data from websocket
func (w *Orderbook) LoadSnapshot(book *orderbook.Base) error {
	w.m.Lock()
	defer w.m.Unlock()

	err := book.Process()
	if err != nil {
		return err
	}

	m1, ok := w.ob[book.Pair.Base]
	if !ok {
		m1 = make(map[currency.Code]map[asset.Item]*orderbookHolder)
		w.ob[book.Pair.Base] = m1
	}
	m2, ok := m1[book.Pair.Quote]
	if !ok {
		m2 = make(map[asset.Item]*orderbookHolder)
		m1[book.Pair.Quote] = m2
	}
	m3, ok := m2[book.Asset]
	if !ok {
		// TODO: Shadow moon
		depth, err := orderbook.GetDepth(book.Exchange, book.Pair, book.Asset)
		if err != nil {
			return err
		}
		m3 = &orderbookHolder{ob: depth, buffer: &[]Update{}}
		m2[book.Asset] = m3
	} else {
		// TODO ADD THIS IN!!!
		// m3.ob.LastUpdateID = book.LastUpdateID
		err = m3.ob.LoadSnapshot(book.Bids, book.Asks)
		if err != nil {
			return err
		}
	}
	w.dataHandler <- m3.ob
	return nil
}

// GetOrderbook returns orderbook stored in current buffer
func (w *Orderbook) GetOrderbook(p currency.Pair, a asset.Item) *orderbook.Base {
	// TODO: Change to return depth pointer and deprecate orderbook base
	w.m.Lock()
	defer w.m.Unlock()
	book, ok := w.ob[p.Base][p.Quote][a]
	if !ok {
		return nil
	}

	bids, asks := book.ob.Retrieve()

	return &orderbook.Base{
		Exchange: w.exchangeName,
		Pair:     p,
		Asset:    a,
		Bids:     bids,
		Asks:     asks}
}

// FlushBuffer flushes w.ob data to be garbage collected and refreshed when a
// connection is lost and reconnected
func (w *Orderbook) FlushBuffer() {
	w.m.Lock()
	w.ob = make(map[currency.Code]map[currency.Code]map[asset.Item]*orderbookHolder)
	w.m.Unlock()
}

// FlushOrderbook flushes independent orderbook
func (w *Orderbook) FlushOrderbook(p currency.Pair, a asset.Item) error {
	w.m.Lock()
	defer w.m.Unlock()
	book, ok := w.ob[p.Base][p.Quote][a]
	if !ok {
		return fmt.Errorf("orderbook not associated with pair: [%s] and asset [%s]", p, a)
	}
	return book.ob.Flush()
}
