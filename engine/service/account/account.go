package account

import (
	uuid "github.com/satori/go.uuid"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	key "github.com/thrasher-/gocryptotrader/engine/account/keys"
)

// Manager manages the full account suite
type Manager struct {
	Accounts []Account
}

// Start starts a new instance of an account manager
func Start(c *config.ExchangeConfig) (*Manager, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	acc := Account{ID: id, Name: "testing", Keys: key.Keys{}}
	return &Manager{Accounts: []Account{acc}}, nil
}

// Manage syncs account deets and stuff
func Manage() {

}

// Account and things
type Account struct {
	ID   uuid.UUID
	Name string
	Keys key.Keys
}

// Balance and things
type Balance struct {
	Currency currency.Code
	Amount   float64
}

type something {}