package account

import "github.com/shopspring/decimal"

// Waiting is used for the matching of liquidity that a pending routine is
// waiting for e.g. when a potential amount is coming from a different exchange
// or wallet.
type Waiting struct {
	h      *Holding
	amount decimal.Decimal
	C      chan *Claim
}

// Done can be defered to release the potential claim
func (w *Waiting) Done() error {
	return w.h.cancelWait(w)
}
