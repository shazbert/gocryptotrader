package engine

import (
	"fmt"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/thrasher-/gocryptotrader/currency"
	key "github.com/thrasher-/gocryptotrader/engine/service/account/keys"
)

// GetAccountSyncer returns a new account syncer
func GetAccountSyncer() (*AccountSyncer, error) {
	return new(AccountSyncer), nil
}

// AccountSyncer is a meow syncer MEEEEEEEOOOOOOOOWWWWWW
type AccountSyncer struct {
	Accounts []Account
	sync.Mutex
}

// Start starts the sync routine
func (a *AccountSyncer) Start() error {
	go func() {
		for {
			for i := range Bot.Exchanges {
				if !Bot.Exchanges[i].IsEnabled() {
					continue
				}

				accInfo, err := Bot.Exchanges[i].GetAccountInfo()
				if err != nil {
					return
				}

				fmt.Println(accInfo)

			}
			time.Sleep(10 * time.Second)
		}
	}()
	return nil
}

// Stop stops the sync routine
func (a *AccountSyncer) Stop() error {
	return nil
}

// Account and things
type Account struct {
	ID   uuid.UUID
	Name string
	Keys key.Keys
	Balances
}

// GetServiceID meow
func (a *Account) GetServiceID() {}

// GetName meow
func (a *Account) GetName() {}

// Balances and stuff and things
type Balances []Balance

// GetFullBalance returns full balance
func (b *Balances) GetFullBalance() {}

// Balance and things
type Balance struct {
	Currency currency.Code
	Amount   float64
}

// NewAccountSyncer starts a new account syncer
func NewAccountSyncer() (*AccountSyncer, error) {
	a := new(AccountSyncer)

	for i := range Bot.Config.Exchanges {
		if Bot.Config.Exchanges[i].Enabled &&
			*Bot.Config.Exchanges[i].AuthenticatedAPISupport {

		}
	}
	return a, nil
}
