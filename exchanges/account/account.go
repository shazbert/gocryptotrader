package account

import (
	"sync"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var accounts []Account

// Account defines full account system settings
type Account struct {
	ClientID     uuid.UUID
	ExchangeName string
	keys         []*Key
	depositAddr  map[*Key]map[currency.Code]string
	Orders       []Order

	sync.RWMutex
}

// Fee defines accounts current exchange fee structure
type Fee struct {
	Maker struct {
		C2C float64
		C2F float64
	}

	Taker struct {
		C2C float64
		C2F float64
	}
}

// Tranche defines an account exchange levels to determine money flow reduction
// or increase based off current exchange client/account volume levels.
type Tranche struct {
	Fee struct {
	}

	WithdrawalLimits struct {
	}

	DepositLimits struct {
	}
}

// Order ...
type Order struct {
	ClientID uuid.UUID
	Pair     currency.Pair
	Asset    asset.Item
	Trades   []order.Trade
	State    string
}

// Key defines an account exchange keyset
type Key struct {
	READ        bool
	WRITE       bool
	WITHDRAW    bool
	APISECRET   string
	APIKEY      string
	OTHERTHINGS string
}

// Balance ...
type Balance struct {
	Pair    currency.Pair
	Asset   asset.Item
	Total   float64
	Claimed map[uuid.UUID]float64
	Free    float64
}
