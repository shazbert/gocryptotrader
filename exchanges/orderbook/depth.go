package orderbook

import (
	"sync"
	"sync/atomic"
)

// Depth defines a linked list of orderbook items
type Depth struct {
	ask linkedList
	bid linkedList

	Options
	Identifier

	// Unexported stack of nodes
	stack Stack

	// Change of state to re-check depth list
	wait    chan struct{}
	waiting *uint32
	wMtx    sync.Mutex
	// -----

	sync.Mutex
}

// LenAsk returns length of asks
func (d *Depth) LenAsk() int {
	d.Lock()
	defer d.Unlock()
	return d.ask.length
}

// LenBids returns length of bids
func (d *Depth) LenBids() int {
	d.Lock()
	defer d.Unlock()
	return d.bid.length
}

// AddBid adds a bid to the list
func (d *Depth) AddBid(i Item) error {
	d.Lock()
	defer d.Unlock()
	return d.bid.Add(func(i Item) bool { return true }, i, &d.stack)
}

// // AddBids adds a collection of bids to the linked list
// func (d *Depth) AddBids(i Item) error {
// 	d.Lock()
// 	defer d.Unlock()
// 	n := d.stack.Pop()
// 	n.value = i
// 	d.bid.Add(func(i Item) bool { return true }, n)
// 	return nil
// }

// RemoveBidByPrice removes a bid
func (d *Depth) RemoveBidByPrice(price float64) error {
	// d.Lock()
	// defer d.Unlock()
	// n, err := d.bid.Remove(func(i Item) bool { return i.Price == price })
	// if err != nil {
	// 	return err
	// }
	// d.stack.Push(n)
	return nil
}

// DisplayBids does a helpful display!!! YAY!
func (d *Depth) DisplayBids() {
	d.Lock()
	defer d.Unlock()
	d.bid.Display()
}

// alert establishes state change for depth to all waiting routines
func (d *Depth) alert() {
	if atomic.LoadUint32(d.waiting) == 0 {
		// return if no waiting routines
		return
	}
	d.wMtx.Lock()
	close(d.wait)
	d.wait = make(chan struct{})
	atomic.SwapUint32(d.waiting, 0)
	d.wMtx.Unlock()
}

// Wait pauses routine until depth change has been established
func (d *Depth) Wait() {
	d.wMtx.Lock()
	atomic.SwapUint32(d.waiting, 1)
	d.wMtx.Unlock()
	<-d.wait
}

// TotalBidsAmount returns the total amount of bids and the total orderbook
// bids value
func (d *Depth) TotalBidsAmount() (liquidity, value float64) {
	d.Lock()
	defer d.Unlock()
	return d.bid.Amount()
}

// TotalAsksAmount returns the total amount of asks and the total orderbook
// asks value
func (d *Depth) TotalAsksAmount() (liquidity, value float64) {
	d.Lock()
	defer d.Unlock()
	return d.ask.Amount()
}

// // Update updates the bids and asks
// func (d *Depth) Update(bids, asks []Item) error {
// 	d.Lock()
// 	defer d.Unlock()

// 	err := d.bid.Load(bids, &d.stack)
// 	if err != nil {
// 		return err
// 	}

// 	err = d.ask.Load(asks, &d.stack)
// 	if err != nil {
// 		return err
// 	}
// 	// Update occurred, alert routines
// 	d.alert()
// 	return nil
// }

// Process processes incoming orderbook snapshots
func (d *Depth) Process(bid, ask Items, fundingRate, notAggregated bool) error {
	err := bid.verifyBids(fundingRate, notAggregated)
	if err != nil {
		return err
	}

	err = ask.verifyAsks(fundingRate, notAggregated)
	if err != nil {
		return err
	}

	// return service.Update(b)
	return nil
}

// invalidate will pop entire bid and ask node chain onto stack when an error
// occurs, so as to not be able to traverse potential invalid books.
func (d *Depth) invalidate() {

}
