package key

import (
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/currency"
)

// Keys is some super keys to monitor
type Keys struct {
	Chain []Key
	sync.RWMutex
}

// Add
func (k *Keys) Add(key Key) error {
	k.Lock()
	k.Chain = append(k.Chain, key)
	k.Unlock()
	return nil
}

// Allowance defines keys ability to do things and stuff
type Allowance struct {
	Read     bool
	Write    bool
	Withdraw bool
	Pairs    currency.Pairs // If none can use all pairs
}

// Key defines a key and relative info and things.
type Key struct {
	Operational bool
	Exchange    string
	Main        string
	Secret      string
	Created     time.Time
	AccessLevel Allowance
}

// Revoke revokes a keys usage
func (k *Key) Revoke() {
	k.Operational = false
}

// Activate allows a keys usage
func (k *Key) Activate() {
	k.Operational = true
}
