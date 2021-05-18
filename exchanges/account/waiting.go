package account

import "github.com/shopspring/decimal"

// Waiting is a (POC) used for the matching of liquidity that a pending strategy
// or subsystem is waiting for e.g. when a potential amount is coming from a
// different exchange or wallet.
// WARNING: This is experimental until integration
type Waiting struct {
	h      *Holding
	amount decimal.Decimal
	C      chan *Claim
}

// Done can be defered to release the potential claim
// WARNING: This is experimental until integration
func (w *Waiting) Done() error {
	return w.h.cancelWait(w)
}

// ClaimAndWait claims an amount on an exchange for the purpose of withdrawal
// and sending it to another exchange for utilisation in structural arbitrage
// runs.
// WARNING: This is experimental until integration
func (h *Holding) ClaimAndWait(c *Claim) (*Waiting, error) {
	// Checks if the passed in claim has been released due to a successful
	// withdrawal
	if c.HasClaim() {
		return nil, errCannotWait
	}
	w := &Waiting{h: h, amount: c.amount, C: make(chan *Claim)}
	h.m.Lock()
	h.waiting = append(h.waiting, w)
	h.m.Unlock()
	return w, nil
}

// TODO: Add in alerting system to first check exact match then apply to first
// FIFO

// cancelWait removes waiting
// WARNING: This is experimental until integration
func (h *Holding) cancelWait(w *Waiting) error {
	h.m.Lock()
	defer h.m.Unlock()
	for x := range h.waiting {
		if h.waiting[x] == w {
			close(h.waiting[x].C)
			h.waiting[x] = h.waiting[len(h.waiting)-1]
			h.waiting[len(h.waiting)-1] = nil
			h.waiting = h.waiting[:len(h.waiting)-1]
			return nil
		}
	}
	return errCannotCancelWait
}
