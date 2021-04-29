package account

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/dispatch"
)

// Vars for the ticker package
var (
	service *Service
)

// Service holds ticker information for each individual exchange
type Service struct {
	accounts map[string]*Holdings
	mux      *dispatch.Mux
	sync.Mutex
}

// // Accounts holds a stream ID and a pointer to the exchange holdings
// type Accounts struct {
// 	h  *Holdings
// 	ID uuid.UUID
// }

// // // Holdings is a generic type to hold each exchange's holdings for all enabled
// // // currencies
// // type Holdings struct {
// // 	Exchange string
// // 	Accounts []SubAccount
// // }

// // SubAccount defines a singular account type with asocciated currency balances
// type SubAccount struct {
// 	ID         string
// 	AssetType  asset.Item
// 	Currencies []Balance
// }

// // Balance is a sub type to store currency name and individual totals
// type Balance struct {
// 	CurrencyName currency.Code
// 	TotalValue   float64
// 	Hold         float64
// }
