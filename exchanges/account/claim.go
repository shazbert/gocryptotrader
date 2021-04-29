package account

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Claim is a type representing the claim on current amount in holdings, this
// will be utilised for withdrawals and trading activity between multiple
// strategies. This allows us to make sure, we have the funds available and we
// cannot execute a double spend which will result in exchange errors.
type Claim struct {
	// amount is the successfully claimed amount requested
	amount decimal.Decimal
	// h is the pointer to the holding for releasing this claim when finished
	h *Holding
	// t is the time at which the claim was successfully called
	t time.Time

	Exchange string
	Asset    asset.Item
	Currency currency.Code

	Detail

	m sync.Mutex
}

// GetAmount returns the amount that has been claimed as a float64
func (c *Claim) GetAmount() float64 {
	c.m.Lock()
	defer c.m.Unlock()
	amt, _ := c.amount.Float64()
	return amt
}

// getAmount returns the amount as a decimal for internal use
func (c *Claim) getAmount() decimal.Decimal {
	c.m.Lock()
	defer c.m.Unlock()
	return c.amount
}

// GetTime returns the time at which the claim was successfully called
func (c *Claim) GetTime() time.Time {
	c.m.Lock()
	defer c.m.Unlock()
	return c.t
}

// Release is when an order fails to execute or funds cannot be withdrawn, this
// will releases the funds back to holdings for further use.
func (c *Claim) Release() error {
	c.m.Lock()
	defer c.m.Unlock()
	err := c.h.Release(c)
	if err != nil {
		return err
	}

	f, _ := c.amount.Float64()

	log.Debugf(log.Accounts,
		"Exchange:%s Account:%s Asset:%s Currency:%s Claim of %f released",
		c.Exchange,
		c.Account,
		c.Asset,
		c.Currency,
		f)
	return nil
}

// ReleaseToPending is used when an order or withdrawal has been been submitted,
// this hands over funds to a pending bucket for account settlement, change of
// state will release these from pending.
func (c *Claim) ReleaseToPending() error {
	c.m.Lock()
	defer c.m.Unlock()
	err := c.h.ReleaseToPending(c)
	if err != nil {
		return err
	}

	f, _ := c.amount.Float64()

	log.Debugf(log.Accounts,
		"Exchange:%s Account:%s Asset:%s Currency:%s Claim of %f released to pending to match with balance change",
		c.Exchange,
		c.Account,
		c.Asset,
		c.Currency,
		f)
	return nil
}

// Float64 returns the amount as a float64 and runs a check to see if current
// claims and locked exceed total amount. This is for the rare event a client
// executes a limit order on an exchange front end on an account which reduces
// total. This system is not designed to handle front end and algo trading but
// this will ensure that there are no orders able to exceed total amount, this
// will release funds so other strategies can then continue operations as normal.
func (c *Claim) CheckAndGetAmount() (float64, error) {
	c.m.Lock()
	defer c.m.Unlock()
	amount, _ := c.amount.Float64()
	err := c.h.Validate(amount)
	if err != nil {
		// errR := c.h.release(c.id, c.amount)
		// if errR != nil {
		// 	log.Errorf(log.ExchangeSys, "converting claim to float64 error: %v", errR)
		// }
		return 0, err
	}
	return amount, nil
}

// HasClaim determines if a claim is still on an amount on a holding
func (c *Claim) HasClaim() bool {
	c.m.Lock()
	defer c.m.Unlock()
	return c.h.CheckClaim(c)
}

// ReleaseAndReduce this pending claim and reduce this amount and total holdings
// manually.
func (c *Claim) ReleaseAndReduce() error {
	c.m.Lock()
	defer c.m.Unlock()
	err := c.h.reduce(c)
	if err != nil {
		return err
	}

	f, _ := c.amount.Float64()

	log.Debugf(log.Accounts,
		"Exchange:%s Account:%s Asset:%s Currency:%s Claim of %f released total balance reduced",
		c.Exchange,
		c.Account,
		c.Asset,
		c.Currency,
		f)

	return nil
}
