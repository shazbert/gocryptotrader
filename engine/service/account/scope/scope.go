package scope

import (
	"errors"
	"fmt"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
)

// HoldingArea defines a set of currency destinations either an exchange,
// Cryptocurrency HOT/COLD Wallet, Bank
type HoldingArea string

// ServiceID defines a service unique identifier
type ServiceID uuid.UUID

// Monitor defines each claim or jurisdication for a segregated instance has
// over funds, exchanges or currencies
type Monitor struct {
	Balances map[HoldingArea]Balance
	sync.RWMutex
}

// GetAllExchangeServices returns all services actively monitoring an exchange
func (m *Monitor) GetAllExchangeServices() ([]Service, error) {
	return nil, common.ErrNotYetImplemented
}

// Balance keeps tracking of all balance and what claims what
type Balance struct {
	Currency currency.Code
	Total    float64
	Free     float64
	Claims   []ServiceClaim
	sync.RWMutex
}

// ServiceClaim denotes an amount that is claimed by the service
type ServiceClaim struct {
	ServiceID
	Amount float64
}

// HasClaim returns if a service has already made claim on a currency
func (b *Balance) HasClaim(uID ServiceID) bool {
	b.RLock()
	var hasClaim bool
	for i := range b.Claims {
		if b.Claims[i].ServiceID == uID {
			hasClaim = true
			break
		}
	}
	b.RUnlock()
	return hasClaim
}

// MakeClaim associates a service with a claimable amount
func (b *Balance) MakeClaim(uID ServiceID, amount float64) error {
	b.Lock()
	if new := b.Free - amount; new < 0 {
		return fmt.Errorf("Failed to make claim required amount: %f free: %f",
			amount, new)
	}

	b.Claims = append(b.Claims, ServiceClaim{ServiceID: uID, Amount: amount})
	b.Free -= amount

	// Checks all services to see if there is a discrepency in total balance
	var fullAmountCheck float64
	for i := range b.Claims {
		fullAmountCheck += b.Claims[i].Amount
	}

	if fullAmountCheck+b.Free != b.Total {
		b.Unlock()
		return errors.New("Can not reconciliate claims total balance")
	}

	b.Unlock()
	return nil
}

// ModifyClaim modifys an associated service with a claimable amount
func (b *Balance) ModifyClaim(uID ServiceID, amount float64) error {
	b.Lock()
	if amount == 0 {
		b.Unlock()
		return errors.New("no amount supplied")
	}

	if amount > 0 {
		if b.Free-amount < 0 {
			b.Unlock()
			return errors.New("not enough balance to make claim")
		}
	}

	for i := range b.Claims {
		if b.Claims[i].ServiceID == uID {
			b.Claims[i].Amount += amount
			b.Free -= amount

			// Checks all services to see if there is a discrepency in total
			// balance
			var fullAmountCheck float64
			for i := range b.Claims {
				fullAmountCheck += b.Claims[i].Amount
			}

			if fullAmountCheck+b.Free != b.Total {
				b.Unlock()
				return errors.New("Can not reconciliate claims total balance")
			}

			b.Unlock()
			return nil
		}
	}
	b.Unlock()
	return errors.New("service ID not found")
}

// DropClaim disassociates a service with a claimable amount
func (b *Balance) DropClaim(uID ServiceID) error {
	b.Lock()

	for i := range b.Claims {
		if b.Claims[i].ServiceID == uID {
			// Add amount to free to claim amount
			b.Free += b.Claims[i].Amount
			// Get rid of this instance completely
			b.Claims = append(b.Claims[:i], b.Claims[i+1:]...)

			// Checks all services to see if there is a discrepency in total
			// balance
			var fullAmountCheck float64
			for i := range b.Claims {
				fullAmountCheck += b.Claims[i].Amount
			}

			if fullAmountCheck+b.Free != b.Total {
				b.Unlock()
				return errors.New("Can not reconciliate claims total balance")
			}

			b.Unlock()
			return nil
		}
	}

	b.Unlock()
	return errors.New("cannot drop service id not found")
}

// MatchCurrency matches if the currency is the same
func (b *Balance) MatchCurrency(c currency.Code) bool {
	b.RLock()
	isMatched := b.Currency.Match(c)
	b.Unlock()
	return isMatched
}

// CanClaim checks to see if a service can claim the amount
func (b *Balance) CanClaim(amount float64) bool {
	b.RLock()
	canClaim := b.Free-amount < 0
	b.RUnlock()
	return canClaim
}

// GetTotal returns total balance
func (b *Balance) GetTotal() float64 {
	b.RLock()
	total := b.Total
	b.RUnlock()
	return total
}

// DecreaseTotal increases amount to total balance
func (b *Balance) DecreaseTotal(amount float64) error {
	b.Lock()
	if b.Free-amount < 0 {
		b.Unlock()
		return errors.New("amount surpasses free amount, de-allocate service then retry")
	}

	b.Free -= amount
	b.Total -= amount

	// Checks all services to see if there is a discrepency in total balance
	var fullAmountCheck float64
	for i := range b.Claims {
		fullAmountCheck += b.Claims[i].Amount
	}

	if fullAmountCheck+b.Free != b.Total {
		b.Unlock()
		return errors.New("Can not reconciliate claims total balance")
	}

	b.Unlock()
	return nil
}

// IncreaseTotal increases total balance
func (b *Balance) IncreaseTotal(amount float64) error {
	b.Lock()

	b.Free += amount
	b.Total += amount

	// Checks all services to see if there is a discrepency in total balance
	var fullAmountCheck float64
	for i := range b.Claims {
		fullAmountCheck += b.Claims[i].Amount
	}

	if fullAmountCheck+b.Free != b.Total {
		b.Unlock()
		return errors.New("Can not reconciliate claims total balance")
	}

	b.Unlock()
	return nil
}

// Service defines a service that is spawned and administered by an engine
type Service struct {
	UID     uuid.UUID
	Name    string
	Version string
	Start   time.Time
}
