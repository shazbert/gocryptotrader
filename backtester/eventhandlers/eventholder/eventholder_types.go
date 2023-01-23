package eventholder

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
)

// Holder contains the event queue for backtester processing
type Holder struct {
	Queue [][]common.Event // Temp bucket for segregation of time intervals
}

// EventHolder interface details what is expected of an event holder to perform
type EventHolder interface {
	Reset() error
	// AppendEvents allows single or mutliple events separated by intervals to
	// be loaded. 1hr, 3hr, 1 week.
	AppendEvents([]common.Event)
	// NextEvents returns the next lot of events
	NextEvents() []common.Event
}
