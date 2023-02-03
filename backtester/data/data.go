package data

import (
	"errors"
	"fmt"
	"time"

	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	errExchangeNameUnset             = errors.New("exchange name is unset")
	errHandlerIsNil                  = errors.New("data handler is nil")
	errDataHandlerAlreadySet         = errors.New("data handler should only be set once")
	errOffsetShifted                 = errors.New("offset has shifted beyond events slice")
	errEventsNotTimeAligned          = errors.New("events not time aligned")
	errEventsAlreadySet              = errors.New("events have already been set")
	errLatestEventHasNotBeenSet      = errors.New("latest e has not been set")
	errDataFeedTypeHasAlreadyBeenSet = errors.New("data feed type has already been set")
	errDuplicateEvent                = errors.New("duplicate event")
	errNoDataEventsLoaded            = errors.New("no data events loaded")
	errTimeMustBeUTC                 = errors.New("event time must be UTC")
	errEventTimeIntervalMismatch     = errors.New("event time not truncated to time interval")
)

// NewHandlerHolder returns a new HandlerHolder
func NewHandlerHolder() *HandlerHolder {
	return &HandlerHolder{
		data: make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]Handler),
	}
}

// SetDataForCurrency assigns a Data Handler to the Data map by exchange, asset and currency
func (h *HandlerHolder) SetDataForCurrency(exchangeName string, a asset.Item, p currency.Pair, k Handler) error {
	if h == nil {
		return fmt.Errorf("%w handler holder", gctcommon.ErrNilPointer)
	}

	if exchangeName == "" {
		return errExchangeNameUnset
	}

	if !a.IsValid() {
		return fmt.Errorf("[%v] %w", a, asset.ErrNotSupported)
	}

	if p.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}

	if k == nil {
		return errHandlerIsNil
	}

	h.m.Lock()
	defer h.m.Unlock()

	if h.data == nil {
		h.data = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]Handler)
	}

	m1, ok := h.data[exchangeName]
	if !ok {
		m1 = make(map[asset.Item]map[*currency.Item]map[*currency.Item]Handler)
		h.data[exchangeName] = m1
	}

	m2, ok := m1[a]
	if !ok {
		m2 = make(map[*currency.Item]map[*currency.Item]Handler)
		m1[a] = m2
	}

	m3, ok := m2[p.Base.Item]
	if !ok {
		m3 = make(map[*currency.Item]Handler)
		m2[p.Base.Item] = m3
	}

	_, ok = m3[p.Quote.Item]
	if ok {
		return errDataHandlerAlreadySet
	}

	m3[p.Quote.Item] = k
	return nil
}

// GetAllData returns all set Data in the Data map
func (h *HandlerHolder) GetAllData() ([]Handler, error) {
	if h == nil {
		return nil, fmt.Errorf("%w handler holder", gctcommon.ErrNilPointer)
	}
	h.m.Lock()
	defer h.m.Unlock()
	var resp []Handler
	for _, exchMap := range h.data {
		for _, assetMap := range exchMap {
			for _, baseMap := range assetMap {
				for _, handler := range baseMap {
					resp = append(resp, handler)
				}
			}
		}
	}
	return resp, nil
}

// GetDataForCurrency returns the Handler for a specific exchange, asset, currency
func (h *HandlerHolder) GetDataForCurrency(exchangeName string, a asset.Item, p currency.Pair) (Handler, error) {
	if h == nil {
		return nil, fmt.Errorf("%w handler holder", gctcommon.ErrNilPointer)
	}

	if exchangeName == "" {
		return nil, errExchangeNameUnset
	}

	if !a.IsValid() {
		return nil, fmt.Errorf("[%v] %w", a, asset.ErrNotSupported)
	}

	if p.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}

	h.m.Lock()
	defer h.m.Unlock()

	handler, ok := h.data[exchangeName][a][p.Base.Item][p.Quote.Item]
	if !ok {
		return nil, fmt.Errorf("%s %s %s %w", exchangeName, a, p, ErrHandlerNotFound)
	}

	return handler, nil
}

// Reset returns the struct to defaults
func (h *HandlerHolder) Reset() error {
	if h == nil {
		return fmt.Errorf("%w handler holder", gctcommon.ErrNilPointer)
	}
	h.m.Lock()
	defer h.m.Unlock()
	h.data = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]Handler)
	return nil
}

// GetDetails returns identification details about the base holder
func (b *Base) GetDetails() (Details, error) {
	if b == nil {
		return Details{}, fmt.Errorf("%w base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()

	if b.latest == nil {
		return Details{}, errLatestEventHasNotBeenSet
	}

	return Details{
		b.latest.GetExchange(),
		b.latest.GetAssetType(),
		b.latest.Pair(),
		b.latest.GetInterval(),
	}, nil
}

// Reset loaded data to blank state
func (b *Base) Reset() error {
	if b == nil {
		return fmt.Errorf("%w base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()
	b.stream = nil
	b.latest = nil
	b.offset = 0
	b.isLiveData = false
	return nil
}

// GetStream will return entire data list
func (b *Base) GetStream() (Events, error) {
	if b == nil {
		return nil, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()
	stream := make([]Event, len(b.stream))
	copy(stream, b.stream)
	return stream, nil
}

// Offset returns the current iteration of candle data the backtester is
// assessing.
func (b *Base) Offset() (int64, error) {
	if b == nil {
		return 0, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()
	return int64(b.offset), nil
}

// SetStream sets the Data stream for candle analysis
func (b *Base) SetStream(events []Event) error {
	if b == nil {
		return fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}

	if len(events) == 0 {
		return fmt.Errorf("cannot set stream, %w", ErrEmptyEventSlice)
	}

	b.m.Lock()
	defer b.m.Unlock()

	if len(b.stream) > 0 {
		return errEventsAlreadySet
	}

	bucket := make([]Event, len(events)) // separates incoming slice from b.stream
	for x := range events {
		err := ValidateEvent(events[x])
		if err != nil {
			return err
		}

		if x != 0 {
			err = CompareEvent(events[x-1], events[x])
			if err != nil {
				return err
			}
		}
		// Due to the Next() function, we cannot take stream offsets as is, and
		// we re-set them.
		events[x].SetOffset(int64(x) + 1)
		bucket[x] = events[x]
	}
	b.stream = bucket
	return nil
}

// AppendStream appends new data onto the stream, however, will not add
// duplicates. Used for live analysis.
func (b *Base) AppendStream(events ...Event) error {
	if b == nil {
		return fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}

	if len(events) == 0 {
		return errNothingToAdd
	}
	b.m.Lock()
	defer b.m.Unlock()

	bucket := make([]Event, len(b.stream), len(b.stream)+len(events))
	copy(bucket, b.stream)

	for x := range events {
		err := ValidateEvent(events[x])
		if err != nil {
			return err
		}

		if len(bucket) > 0 {
			err = CompareEvent(bucket[len(bucket)-1], events[x])
			if err != nil {
				return err
			}
		}
		bucket = append(bucket, events[x])
		events[x].SetOffset(int64(len(bucket) + 1))
	}
	b.stream = bucket
	return nil
}

// Next will return the next e in the list and also shifts the offset by one
func (b *Base) Next() (Event, error) {
	if b == nil {
		return nil, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}

	b.m.Lock()
	defer b.m.Unlock()

	if len(b.stream) < b.offset {
		return nil, errOffsetShifted
	}

	if len(b.stream) == b.offset {
		return nil, fmt.Errorf("%w data length %v offset %v", ErrEndOfData, len(b.stream), b.offset)
	}

	b.latest = b.stream[b.offset]
	b.offset++
	return b.latest, nil
}

// History will return all previous data events that have happened
func (b *Base) History() (Events, error) {
	if b == nil {
		return nil, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()

	if len(b.stream) < b.offset {
		return nil, errOffsetShifted
	}

	return b.stream[:b.offset], nil
}

// Latest will return latest Data event
func (b *Base) Latest() (Event, error) {
	if b == nil {
		return nil, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()

	if len(b.stream) == 0 {
		return nil, errNoDataEventsLoaded
	}
	if b.latest == nil {
		b.latest = b.stream[0]
	}
	return b.latest, nil
}

// List returns all future Data events from the current iteration. Ill-advised
// to use this in strategies because you don't know the future in real life.
func (b *Base) List() (Events, error) {
	if b == nil {
		return nil, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()

	if len(b.stream) < b.offset {
		return nil, errOffsetShifted
	}

	return b.stream[b.offset:], nil
}

// IsLastEvent determines whether the latest e is the last e for live
// data, this will be false, as all appended data is the latest available data
// and this signal cannot be completely relied upon.
func (b *Base) IsLastEvent() (bool, error) {
	if b == nil {
		return false, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()
	return !b.isLiveData && b.latest != nil && b.latest == b.stream[len(b.stream)-1], nil
}

// IsLive returns if the Data source is a live one. Less scrutiny on checks is
// required on live data sourcing.
func (b *Base) IsLive() (bool, error) {
	if b == nil {
		return false, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()
	return b.isLiveData, nil
}

// SetLive sets if the Data source is a live one
// less scrutiny on checks is required on live Data sourcing
func (b *Base) SetLive(isLive bool) error {
	if b == nil {
		return fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()
	if b.isLiveData == isLive {
		return errDataFeedTypeHasAlreadyBeenSet
	}
	b.isLiveData = isLive
	return nil
}

// First returns the first element of a slice
func (e Events) First() (Event, error) {
	if len(e) == 0 {
		return nil, ErrEmptyEventSlice
	}
	return e[0], nil
}

// Last returns the last element of a slice
func (e Events) Last() (Event, error) {
	if len(e) == 0 {
		return nil, ErrEmptyEventSlice
	}
	return e[len(e)-1], nil
}

// ValidateEvent validates incoming e type
// NOTE: This is added for a future method NextByTime(time.Time) (Handler,error)
func ValidateEvent(e Event) error {
	if e == nil {
		return fmt.Errorf("%w Event", gctcommon.ErrNilPointer)
	}

	eventTime := e.GetTime()
	eventInterval := e.GetInterval()
	if e.GetExchange() == "" ||
		!e.GetAssetType().IsValid() ||
		e.Pair().IsEmpty() ||
		eventTime.IsZero() ||
		eventInterval == 0 {
		return ErrInvalidEventSupplied
	}

	// This should always be UTC when (*event.Base).GetTime() is called.
	// TODO: Remove event time conversion.
	if eventTime.Location() != time.UTC {
		return errTimeMustBeUTC
	}

	if !eventTime.Equal(eventTime.Truncate(eventInterval.Duration())) {
		return errEventTimeIntervalMismatch
	}

	return nil
}

// CompareEvent determines if the previous and current are in the correct order
// (time aligned) and have the same currency details.
// NOTE: This is added for a future method NextByTime(time.Time) (Handler,error)
func CompareEvent(prev, current Event) error {
	if prev.GetTime().Equal(current.GetTime()) {
		return errDuplicateEvent
	}

	if prev.GetTime().After(current.GetTime()) {
		return errEventsNotTimeAligned
	}

	if prev.GetExchange() != current.GetExchange() ||
		prev.GetAssetType() != current.GetAssetType() ||
		!prev.Pair().Equal(current.Pair()) ||
		prev.GetInterval() != current.GetInterval() {
		return fmt.Errorf("%w cannot set base stream from %v %v %v %v to %v %v %v %v",
			errMisMatchedEvent,
			current.GetExchange(),
			current.GetAssetType(),
			current.Pair(),
			current.GetInterval(),
			prev.GetExchange(),
			prev.GetAssetType(),
			prev.Pair(),
			prev.GetInterval())
	}
	return nil
}
