package eventholder

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
)

// Reset returns struct to defaults
func (h *Holder) Reset() error {
	if h == nil {
		return gctcommon.ErrNilPointer
	}
	h.Queue = nil
	return nil
}

// AppendEvent adds and event to the queue
func (h *Holder) AppendEvents(events []common.Event) { // TODO: Return errors?
	// runtime.Breakpoint()
	// fmt.Println("appending event", i.GetTime(), i.GetInterval())
	h.Queue = append(h.Queue, events)
}

// NextEvent removes the current event and returns the next event in the queue
// TODO: Rethink this design.
func (h *Holder) NextEvents() (events []common.Event) { // TODO: Return error
	if len(h.Queue) == 0 {
		return nil
	}

	events = h.Queue[0]
	h.Queue = h.Queue[1:]
	// TODO: Use integer to iterate through events so we don't need to resize
	// queue system
	return events
}
