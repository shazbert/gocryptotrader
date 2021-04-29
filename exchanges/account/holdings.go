package account

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Undefined is a placeholder ID in memory for claims matching when limit orders
// are added via rest or websocket fetching they can then be matched within the
// locked funds and the actual internal ids can be assigned. In future we can
// cancel via these attributes.
var Undefined, _ = uuid.NewV4()

// Defined is a placeholder ID in memory
var Defined, _ = uuid.NewV4()

// Main defines the main account name
const Main = "main"

// var errCannotClaim = errors.New("cannot claim total amount required")
// var errIDNotFound = errors.New("id not found when releasing claim")
// var errCannotMatch = errors.New("cannot match amount with claims")
var errAccountNameUnset = errors.New("account name unset")
var errCurrencyIsEmpty = errors.New("currency is empty")

// Holdings define exchange account holdings
type Holdings struct {
	Exchange string
	funds    map[string]map[asset.Item]map[*currency.Item]*Holding
	mux      *dispatch.Mux
	id       uuid.UUID
	m        sync.Mutex

	// available accounts is left out so we can attach a RW mutex and it doesn't
	// involve other systems for a check
	// This will eventually be updated to include different key usages
	availableAccounts []string
	accMtx            sync.RWMutex
}

// GetAccounts returns the loaded accounts in usage and with balance
func (h *Holdings) GetAccounts() ([]string, error) {
	h.accMtx.RLock()
	defer h.accMtx.RUnlock()

	amount := len(h.availableAccounts)
	if amount == 0 {
		return nil, errors.New("accounts not loaded")
	}

	acc := make([]string, amount)
	copy(acc, h.availableAccounts)
	return acc, nil
}

// AccountValid
func (h *Holdings) AccountValid(acc string) error {
	h.accMtx.RLock()
	defer h.accMtx.RUnlock()

	for x := range h.availableAccounts {
		if h.availableAccounts[x] == acc {
			return nil
		}
	}
	return fmt.Errorf("account validation error: %s not found in available accounts list: %s",
		acc,
		h.availableAccounts)
}

func (h *Holdings) LoadAccount(acc string) {
	h.accMtx.Lock()
	defer h.accMtx.Unlock()

	for x := range h.availableAccounts {
		if h.availableAccounts[x] == acc {
			return
		}
	}
	h.availableAccounts = append(h.availableAccounts, acc)
}

// GetHolding returns the holding for a specific currency tied to an account
func (h *Holdings) GetHolding(account string, a asset.Item, c currency.Code) (*Holding, error) {
	if account == "" {
		return nil, fmt.Errorf("cannot get holding for %s %s %s %s: %w",
			h.Exchange,
			account,
			a,
			c,
			errAccountNameUnset)
	}

	if !a.IsValid() {
		return nil, fmt.Errorf("cannot get holding for %s %s %s %s: %w",
			h.Exchange,
			account,
			a,
			c,
			asset.ErrNotSupported)
	}

	if c.IsEmpty() {
		return nil, fmt.Errorf("cannot get holding for %s %s %s %s: %w",
			h.Exchange,
			account,
			a,
			c,
			errCurrencyIsEmpty)
	}

	h.m.Lock()
	defer h.m.Unlock()
	// Below we create the map contents if not found because if we have a
	// strategy waiting for funds on an exchange or even transfer between
	// accounts, this will set up the requirements in memory to be updated when
	// the funds come in.
	m1, ok := h.funds[account]
	if !ok {
		m1 = make(map[asset.Item]map[*currency.Item]*Holding)
		h.funds[account] = m1
	}

	m2, ok := m1[a]
	if !ok {
		m2 = make(map[*currency.Item]*Holding)
		m1[a] = m2
	}

	holding, ok := m2[c.Item]
	if !ok {
		holding = &Holding{}
		m2[c.Item] = holding
	}

	return holding, nil
}

// Value defines amount levels on an exchange account for a currency holding
type Value struct {
	Total  float64
	Locked float64
}

// LoadHoldings flushes the entire amounts with the supplied values account
// values, anything contained in the holdings funds that is not part of the
// supplied values list will be readjusted to zero value.
func (h *Holdings) LoadHoldings(account string, a asset.Item, values map[*currency.Item]Value) error {
	if account == "" {
		return fmt.Errorf("cannot load holdings for %s %s %s: %w",
			h.Exchange,
			account,
			a,
			errAccountNameUnset)
	}

	account = strings.ToLower(account)

	if !a.IsValid() {
		return fmt.Errorf("cannot load holdings for %s %s %s: %w",
			h.Exchange,
			account,
			a,
			asset.ErrNotSupported)
	}

	h.m.Lock()
	defer h.m.Unlock()
	m1, ok := h.funds[account]
	if !ok {
		m1 = make(map[asset.Item]map[*currency.Item]*Holding)
		h.funds[account] = m1
		// Loads instance of account name for other sub-system interactions
		h.LoadAccount(account)
	}

	m2, ok := m1[a]
	if !ok {
		m2 = make(map[*currency.Item]*Holding)
		m1[a] = m2
	}

	// Add/Change
	for code, val := range values {
		total := decimal.NewFromFloat(val.Total)
		locked := decimal.NewFromFloat(val.Locked)
		free := total.Sub(locked)
		holding, ok := m2[code]
		if !ok {
			m2[code] = &Holding{
				total:  total,
				locked: locked,
				free:   free,
			}
			continue
		}
		holding.setAmounts(total, locked)
	}

	// Reset dangling values to zero
	for code, holding := range m2 {
		_, ok := values[code]
		if ok {
			continue
		}
		holding.setAmounts(decimal.Zero, decimal.Zero)
	}

	log.Debugf(log.Accounts,
		"Exchange:%s Account:%s Asset:%s Holdings Loaded",
		h.Exchange,
		account,
		a)

	// publish change to dispatch system and portfolio
	h.publish()
	return nil
}

// type Detail struct {
// 	Total           float64
// 	Locked          float64
// 	StrategyClaimed float64
// }

// type Snapshot map[string]map[asset.Item]map[currency.Code]Detail

// publish publishes update to the dispatch mux to be called in a go routine
func (h *Holdings) publish() {
	// log.Errorf(log.ExchangeSys, "account cannot publish data %v", err)
	// var ss Snapshot = make(map[string]map[asset.Item]map[currency.Code]Detail)
	// h.m.Lock()
	// defer h.m.Unlock()
	// for account, funds := range h.funds {
	// 	for assets, currencies := range funds {
	// 		for code, holding := range currencies {
	// 			m1, ok := ss[account]
	// 			if !ok {
	// 				m1 = make(map[asset.Item]map[currency.Code]Detail)
	// 				ss[account] = m1
	// 			}

	// 			m2, ok := m1[assets]
	// 			if !ok {
	// 				m2 = make(map[currency.Code]Detail)
	// 				m1[assets] = m2
	// 			}

	// 			m2[currency.Code{Item: code}] = Detail{
	// 				// Total:  holding.GetTotal(),
	// 				// Locked: holding.GetLocked(),
	// 			}
	// 		}
	// 	}
	// }
	// return h.mux.Publish(ss, h.id)
}

// func (h *Holdings) GetSnapshot() Snapshot {
// 	return Snapshot{}
// }

var errCurrencyCodeEmpty = errors.New("currrency code cannot be empty")
var errAmountCannotBeZero = errors.New("amount cannot be zero")
var errAccountNotFound = errors.New("account not found in holdings")
var errAssetTypeNotFound = errors.New("asset type not found in holdings")
var errCurrencyItemNotFound = errors.New("currency not found in holdings")

// AdjustByBalance matches with currency currency holding and decreases or
// increases on value change. i.e. if negative will decrease current holdings
// if positive will increase current holdings
func (h *Holdings) AdjustByBalance(account string, a asset.Item, c currency.Code, amount float64) error {
	if account == "" {
		return fmt.Errorf("cannot adjust holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			errAccountNameUnset)
	}
	account = strings.ToLower(account)
	if !a.IsValid() {
		return fmt.Errorf("cannot adjust holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			asset.ErrNotSupported)
	}
	if c.IsEmpty() {
		return fmt.Errorf("cannot adjust holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			errCurrencyCodeEmpty)
	}
	if amount == 0 {
		return fmt.Errorf("cannot adjust holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			errAmountCannotBeZero)
	}

	h.m.Lock()
	defer h.m.Unlock()

	m1, ok := h.funds[account]
	if !ok {
		return fmt.Errorf("cannot adjust holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			errAccountNotFound)
	}

	m2, ok := m1[a]
	if !ok {
		return fmt.Errorf("cannot adjust holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			errAssetTypeNotFound)
	}

	holding, ok := m2[c.Item]
	if !ok {
		return fmt.Errorf("cannot adjust holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			errCurrencyItemNotFound)
	}

	err := holding.adjustByBalance(amount)
	if err != nil {
		return fmt.Errorf("cannot adjust holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			err)
	}

	log.Debugf(log.Accounts,
		"Exchange:%s Account:%s Asset:%s Currency:%s Balance Adjusted by %f Current Free Holdings:%f Current Total Holdings:%f",
		h.Exchange,
		account,
		a,
		c,
		amount,
		holding.GetFree(),
		holding.GetTotal())
	return err
}

var errAmountCannotBeLessOrEqualToZero = errors.New("amount cannot be less or equal to zero")

func (h *Holdings) Claim(account string, a asset.Item, c currency.Code, amount float64, totalRequired bool) (*Claim, error) {
	if account == "" {
		return nil, fmt.Errorf("cannot claim holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			errAccountNameUnset)
	}
	account = strings.ToLower(account) // TODO: This crap is slow RPC input we can lower it and remove this junk
	if !a.IsValid() {
		return nil, fmt.Errorf("cannot claim holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			asset.ErrNotSupported)
	}
	if c.IsEmpty() {
		return nil, fmt.Errorf("cannot claim holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			errCurrencyCodeEmpty)
	}
	if amount <= 0 {
		return nil, fmt.Errorf("cannot claim holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			errAmountCannotBeLessOrEqualToZero)
	}

	err := h.AccountValid(account)
	if err != nil {
		return nil, fmt.Errorf("cannot claim holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			err)
	}

	h.m.Lock()
	defer h.m.Unlock()

	m1, ok := h.funds[account]
	if !ok {
		return nil, fmt.Errorf("cannot claim holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			errAccountNotFound)
	}

	m2, ok := m1[a]
	if !ok {
		return nil, fmt.Errorf("cannot claim holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			errAssetTypeNotFound)
	}

	holding, ok := m2[c.Item]
	if !ok {
		return nil, fmt.Errorf("cannot claim holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			errCurrencyItemNotFound)
	}

	claim, err := holding.Claim(amount,
		totalRequired,
		Detail{Exchange: h.Exchange, Account: account, Asset: a, Currency: c})
	if err != nil {
		log.Errorf(log.Accounts,
			"Exchange:%s Account:%s Asset:%s Currency:%s total required %v, could not claim %f on holdings, Free Holdings:%f Total Holdings:%f",
			h.Exchange,
			account,
			a,
			c,
			totalRequired,
			amount,
			holding.GetFree(),
			holding.GetTotal(),
		)
		return nil, fmt.Errorf("cannot claim holdings for %s %s %s %s by %f: %w",
			h.Exchange,
			account,
			a,
			c,
			amount,
			err)
	}

	log.Debugf(log.Accounts,
		"Exchange:%s Account:%s Asset:%s Currency:%s total required: %v, amount %f claimed on holdings with amount requested %f Free Holdings:%f Total Holdings:%f",
		h.Exchange,
		account,
		a,
		c,
		totalRequired,
		claim.GetAmount(),
		amount,
		holding.GetFree(),
		holding.GetTotal())
	return claim, err
}

var errNoHoldings = errors.New("initial holdings not loaded")

func (h *Holdings) GetHoldings() (*Holdings, error) {
	h.m.Lock()
	defer h.m.Unlock()

	if h.funds == nil {
		return nil, errNoHoldings
	}

	return h, nil
}

// AdjustPendingFree adjusts the free and pending amounts on order
func (h *Holdings) AdjustPendingFree(account string, a asset.Item, c currency.Code, free, pending float64) {
	// TODO: WOW
}
