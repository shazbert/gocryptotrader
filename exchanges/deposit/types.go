package deposit

import "github.com/thrasher-corp/gocryptotrader/exchanges/chain"

// Address holds a deposit address
type Address struct {
	Address string
	Tag     string // Represents either a tag or memo
	Chain   chain.Item
}
