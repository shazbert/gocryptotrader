package account

import (
	"errors"
	"sync"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	errAmountExceedsHoldings = errors.New("amount exceeds current free amount")
	errUnableToReleaseClaim  = errors.New("unable to release claim holding amounts may be locked")
)

// Holding defines the total currency holdings for an account and what is
// currently in use.
type Holding struct {
	// Exchange side levels. Warning: This has some observational dilemma.
	total  decimal.Decimal
	locked decimal.Decimal
	// --------- These will only be altered by the exchange updates --------- //

	// free is the current free amount (total - (locked + claims + pending)).
	free decimal.Decimal

	// claims is the list of current internal claims on current liquidity.
	claims []*Claim

	// pending is a bucket for when we execute an order and liquidity is
	// potentially taken off the exchange but we cannot release the claim amount
	// to free until we can match it by an exchange update. This should reduce
	// when the total amount reduces.
	pending decimal.Decimal

	// waiting is a first in first out slice of potential future claims
	waiting []*Waiting
	m       sync.Mutex
}

// GetTotal returns the current total holdings
func (h *Holding) GetTotal() float64 {
	h.m.Lock()
	total, _ := h.total.Float64()
	h.m.Unlock()
	return total
}

// GetLocked returns the current locked holdings
func (h *Holding) GetLocked() float64 {
	h.m.Lock()
	locked, _ := h.locked.Float64()
	h.m.Unlock()
	return locked
}

// GetPending returns the current pending holdings
func (h *Holding) GetPending() float64 {
	h.m.Lock()
	pending, _ := h.pending.Float64()
	h.m.Unlock()
	return pending
}

// GetFree returns the current free holdings
func (h *Holding) GetFree() float64 {
	h.m.Lock()
	free, _ := h.free.Float64()
	h.m.Unlock()
	return free
}

// GetTotalClaims returns the total claims amount
func (h *Holding) GetTotalClaims() float64 {
	var total decimal.Decimal
	h.m.Lock()
	for x := range h.claims {
		total = total.Add(h.claims[x].getAmount())
	}
	h.m.Unlock()
	claims, _ := total.Float64()
	return claims
}

// setAmounts sets current account amounts in relation to exchange liqudity.
// These totals passed in are exchange items only, we will need to calculate our
// free amounts dependant on current claims, whats currently locked and whats
// pending.
func (h *Holding) setAmounts(total, locked decimal.Decimal) {
	// Determine total free on the exchange
	free := total.Sub(locked)
	h.m.Lock()

	// Determine full claimed amount
	var claimed decimal.Decimal
	for x := range h.claims {
		claimed = claimed.Add(h.claims[x].getAmount())
	}

	if !h.pending.LessThanOrEqual(decimal.Zero) {
		totalDifference := h.total.Sub(total)
		if totalDifference.GreaterThan(decimal.Zero) {
			// Reduce our pending claims which increases our free amount
			h.pending = h.pending.Sub(totalDifference)
		}
		// remove the residual pending amount from the free amount
		remaining := h.pending.Sub(locked)
		if remaining.GreaterThan(decimal.Zero) {
			free = free.Sub(h.pending)
		}
	}
	h.total = total
	h.locked = locked
	h.free = free.Sub(claimed) // Remove any claims on free amounts
	h.m.Unlock()
}

var errInvalidHoldings = errors.New("invalid holdings and claims")

// ValidateAmounts checks to see if the free holdings are in the negative, which
// means a silly person using these API keys on that account did an order via
// front end or moved some assets around.
func (h *Holding) Validate(amount float64) error {
	h.m.Lock()
	defer h.m.Unlock()
	if h.free.LessThan(decimal.Zero) {
		return errInvalidHoldings
	}
	return nil
}

var ErrNoBalance = errors.New("no balance on holdings")

type Detail struct {
	Exchange, Account string
	Asset             asset.Item
	Currency          currency.Code
}

// Claim returns a claim to an amount for the exchange account holding. Allows
// strategies to segregate their own funds from each other while executing in
// parallel. If total amount is required, this will return an error else the
// remaining/free amount will be claimed and returned.
func (h *Holding) Claim(amount float64, totalRequired bool, iden Detail) (*Claim, error) {
	amt := decimal.NewFromFloat(amount)
	h.m.Lock()
	defer h.m.Unlock()
	if h.free.Equal(decimal.Zero) {
		return nil, ErrNoBalance
	}
	remainder := h.free.Sub(amt)
	if remainder.LessThan(decimal.Zero) {
		if totalRequired {
			return nil, errAmountExceedsHoldings
		}
		// Claims the total free amount
		freeClaim := &Claim{amount: h.free, h: h}
		// sets free amount to zero
		h.free = decimal.Zero
		// Adds claim for tracking
		h.claims = append(h.claims, freeClaim)
		return freeClaim, nil
	}
	// sets the remainder to the new free amount
	h.free = remainder
	amountClaim := &Claim{amount: amt, h: h}
	h.claims = append(h.claims, amountClaim)
	// return the full requested amount
	return amountClaim, nil
}

// Release is a protected exported function to release funds that has not
// been successful or is not used
func (h *Holding) Release(c *Claim) error {
	h.m.Lock()
	defer h.m.Unlock()
	return h.release(c, false)
}

// ReleaseToPending is a protected exported function to release funds and shift
// them to pending when an order or a withdrawal opperation has succeeded.
func (h *Holding) ReleaseToPending(c *Claim) error {
	h.m.Lock()
	defer h.m.Unlock()
	return h.release(c, true)
}

// release releases the funds either to pending or free.
func (h *Holding) release(c *Claim, pending bool) error {
	for x := range h.claims {
		if h.claims[x] == c {
			// Remove claim from claims slice
			h.claims[x] = h.claims[len(h.claims)-1]
			h.claims[len(h.claims)-1] = nil
			h.claims = h.claims[:len(h.claims)-1]

			if pending {
				// Change pending amount to be re-adjusted when a new update
				// comes through
				h.pending = h.pending.Add(c.amount)
				return nil
			}
			// Change free amount NOTE: not changing locked amount as this is
			// done by the exchange update
			h.free = h.free.Add(c.amount)
			return nil
		}
	}
	return errUnableToReleaseClaim
}

// CheckClaim determines if a claim is still on an currency holding
func (h *Holding) CheckClaim(c *Claim) bool {
	h.m.Lock()
	defer h.m.Unlock()
	for x := range h.claims {
		if h.claims[x] == c {
			return true
		}
	}
	return false
}

var errCannotWait = errors.New("cannot wait supplied claim is not released")

// ClaimAndWait claims an amount on an exchange for the purpose of withdrawal
// and sending it to another exchange for utilisation in structural arbitrage
// runs.
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

var errCannotCancelWait = errors.New("failed to cancel waiting claim, not found")

// cancelWait removes waiting
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

// adjustByBalance defines a way in which the entire holdings can be adjusted by
// a balance change in reference to pending amounts. TODO: Add in a lot of tests
func (h *Holding) adjustByBalance(amount float64) error {
	if amount == 0 {
		return errAmountCannotBeZero
	}

	h.m.Lock()
	defer h.m.Unlock()
	amt := decimal.NewFromFloat(amount)
	amountPending := h.pending.GreaterThan(decimal.Zero)
	if amt.GreaterThan(decimal.Zero) {
		// Adds to holdings
		if amountPending {
			remaining := h.pending.Sub(amt)
			if remaining.GreaterThanOrEqual(decimal.Zero) {
				h.pending = remaining
				// Increase total holdings for the remainder
				h.total = h.total.Sub(remaining)
			} else {
				h.pending = decimal.Zero
			}
		} else {
			h.free = h.free.Add(amt)
			if h.free.GreaterThan(h.total) {
				// Step up amount
				h.total = h.free
			}
			// Decrease locked amount
			h.locked = h.locked.Sub(amt)
		}
	} else {
		// Remove from holdings
		if amountPending {
			remaining := h.pending.Add(amt)
			if remaining.GreaterThanOrEqual(decimal.Zero) {
				h.pending = remaining
				// Decrease total holdings for the remainder
				h.total = h.total.Add(amt)
			} else {
				h.pending = decimal.Zero
			}
		} else {
			h.free = h.free.Add(amt)
			if h.free.GreaterThan(h.total) {
				// Step up amount
				h.total = h.free
			}
			// Increase locked amount
			h.locked = h.locked.Sub(amt)
		}
	}
	return nil
}

var errUnableToReduceClaim = errors.New("unable to reduce claim, claim not found")

// reduce reduces holdings by claim
func (h *Holding) reduce(c *Claim) error {
	h.m.Lock()
	defer h.m.Unlock()
	for x := range h.claims {
		if h.claims[x] == c {
			// Remove claim from claims slice
			h.claims[x] = h.claims[len(h.claims)-1]
			h.claims[len(h.claims)-1] = nil
			h.claims = h.claims[:len(h.claims)-1]

			// Reduce total amount
			h.total = h.total.Sub(c.amount)
			return nil
		}
	}
	return errUnableToReduceClaim
}
