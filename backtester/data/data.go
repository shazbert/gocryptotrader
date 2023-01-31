package data

import (
	"errors"
	"fmt"
	"strings"
	"time"

	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var (
	errExchangeNameUnset             = errors.New("exchange name is unset")
	errHandlerIsNil                  = errors.New("data handler is nil")
	errDataHandlerAlreadySet         = errors.New("data handler should only be set once")
	errOffsetShifted                 = errors.New("offset has shifted beyond events slice")
	errEventsNotTimeAligned          = errors.New("events not time aligned")
	errEventsAlreadySet              = errors.New("events have already been set")
	errLatestEventHasNotBeenSet      = errors.New("latest event has not been set")
	errDataFeedTypeHasAlreadyBeenSet = errors.New("data feed type has already been set")
	errDuplicateEvent                = errors.New("duplicate event")
)

// NewHandlerHolder returns a new HandlerHolder
func NewHandlerHolder() *HandlerHolder {
	return &HandlerHolder{
		data: make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]map[gctkline.Interval]Handler),
	}
}

// SetDataForCurrency assigns a Data Handler to the Data map by exchange, asset and currency
func (h *HandlerHolder) SetDataForCurrency(e string, a asset.Item, p currency.Pair, in gctkline.Interval, k Handler) error {
	if h == nil {
		return fmt.Errorf("%w handler holder", gctcommon.ErrNilPointer)
	}

	if e == "" {
		return errExchangeNameUnset
	}

	if !a.IsValid() {
		return fmt.Errorf("[%v] %w", a, asset.ErrNotSupported)
	}

	if p.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}

	if in <= 0 {
		return gctkline.ErrInvalidInterval
	}

	if k == nil {
		return errHandlerIsNil
	}

	h.m.Lock()
	defer h.m.Unlock()

	if h.data == nil {
		h.data = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]map[gctkline.Interval]Handler)
	}
	e = strings.ToLower(e)
	m1, ok := h.data[e]
	if !ok {
		m1 = make(map[asset.Item]map[*currency.Item]map[*currency.Item]map[gctkline.Interval]Handler)
		h.data[e] = m1
	}

	m2, ok := m1[a]
	if !ok {
		m2 = make(map[*currency.Item]map[*currency.Item]map[gctkline.Interval]Handler)
		m1[a] = m2
	}

	m3, ok := m2[p.Base.Item]
	if !ok {
		m3 = make(map[*currency.Item]map[gctkline.Interval]Handler)
		m2[p.Base.Item] = m3
	}

	m4, ok := m3[p.Quote.Item]
	if !ok {
		m4 = make(map[gctkline.Interval]Handler)
		m3[p.Quote.Item] = m4
	}

	_, ok = m4[in]
	if ok {
		return errDataHandlerAlreadySet
	}

	m4[in] = k
	return nil
}

// GetAllData returns all set Data in the Data map
func (h *HandlerHolder) GetAllData() (AssetSegregated, error) {
	if h == nil {
		return nil, fmt.Errorf("%w handler holder", gctcommon.ErrNilPointer)
	}

	h.m.Lock()
	defer h.m.Unlock()

	var resp AssetSegregated
	for _, exchMap := range h.data {
		for _, assetMap := range exchMap {
			for _, baseMap := range assetMap {
				for _, quoteMap := range baseMap {
					multiIntervals := make([]Handler, 0, len(quoteMap))
					for _, handler := range quoteMap {
						multiIntervals = append(multiIntervals, handler)
					}
					resp = append(resp, multiIntervals)
				}
			}
		}
	}
	return resp, nil
}

// GetDataForCurrency returns the Handler for a specific exchange, asset, currency
func (h *HandlerHolder) GetDataForCurrency(exch string, a asset.Item, p currency.Pair) (IntervalSegregated, error) {
	if h == nil {
		return nil, fmt.Errorf("%w handler holder", gctcommon.ErrNilPointer)
	}

	if exch == "" {
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

	intervalCurrencyData, ok := h.data[exch][a][p.Base.Item][p.Quote.Item]
	if !ok {
		return nil, fmt.Errorf("%s %s %s %w", exch, a, p, ErrHandlerNotFound)
	}

	handlers := make([]Handler, 0, len(intervalCurrencyData))
	for _, handler := range intervalCurrencyData {
		handlers = append(handlers, handler)
	}

	return handlers, nil
}

// Reset returns the struct to defaults
func (h *HandlerHolder) Reset() error {
	if h == nil {
		return fmt.Errorf("%w handler holder", gctcommon.ErrNilPointer)
	}
	h.m.Lock()
	defer h.m.Unlock()
	h.data = make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]map[gctkline.Interval]Handler)
	return nil
}

// GetDetails returns data about the Base Holder
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
	*b = Base{}
	return nil
}

// GetStream will return entire Data list
// TODO: Change name from GetStream to GetAllEvents
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
	return b.offset, nil
}

// SetStream sets the Data stream for candle analysis
// TODO: Change name from SetStream to SetAllEvents
func (b *Base) SetStream(events []Event) error {
	if b == nil {
		return fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}

	if len(events) == 0 {
		return ErrEmptySlice
	}

	b.m.Lock()
	defer b.m.Unlock()

	if len(b.stream) > 0 {
		return errEventsAlreadySet
	}

	b.stream = make([]Event, len(events))
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
		b.stream[x] = events[x]
	}
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

	bucket := make([]Event, 0, len(b.stream)+len(events))
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

// Next will return the next event in the list and also shifts the offset by one
func (b *Base) Next() (Event, error) {
	if b == nil {
		return nil, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}

	b.m.Lock()
	defer b.m.Unlock()

	if int64(len(b.stream)) < b.offset {
		return nil, errOffsetShifted
	}

	if int64(len(b.stream)) == b.offset {
		return nil, fmt.Errorf("%w data length %v offset %v", ErrEndOfData, len(b.stream), b.offset)
	}

	b.latest = b.stream[b.offset]
	b.offset++
	return b.latest, nil
}

// Next will return the next event in the list if matched and also shift the
// offset one. If not matched it will not increment offset and keep returning
// last event.
func (b *Base) NextByTime(et time.Time) (Event, error) {
	if b == nil {
		return nil, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	if et.IsZero() {
		return nil, errors.New("time is empty")
	}
	b.m.Lock()
	defer b.m.Unlock()

	if int64(len(b.stream)) < b.offset {
		return nil, errOffsetShifted
	}

	if int64(len(b.stream)) == b.offset {
		return nil, fmt.Errorf("%w data length %v offset %v", ErrEndOfData, len(b.stream), b.offset)
	}

	ret := b.stream[b.offset]
	if ret.GetTime().Equal(et) {
		b.offset++
		b.latest = ret
	}
	return ret, nil
}

// History will return all previous data events that have happened
func (b *Base) History() (Events, error) {
	if b == nil {
		return nil, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()

	if int64(len(b.stream)) < b.offset {
		return nil, errOffsetShifted
	}

	return b.stream[:b.offset], nil
}

var errNoDataEventsLoaded = errors.New("no data events loaded")

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

	if int64(len(b.stream)) < b.offset {
		return nil, errOffsetShifted
	}

	return b.stream[b.offset:], nil
}

// IsLastEvent determines whether the latest event is the last event for live
// data, this will be false, as all appended data is the latest available data
// and this signal cannot be completely relied upon.
func (b *Base) IsLastEvent() (bool, error) {
	if b == nil {
		return false, fmt.Errorf("%w Base", gctcommon.ErrNilPointer)
	}
	b.m.Lock()
	defer b.m.Unlock()
	return !b.isLiveData && b.latest != nil && b.latest == b.stream[len(b.stream)], nil
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
		return nil, ErrEmptySlice
	}
	return e[0], nil
}

// Last returns the last element of a slice
func (e Events) Last() (Event, error) {
	if len(e) == 0 {
		return nil, ErrEmptySlice
	}
	return e[len(e)-1], nil
}

// ValidateEvent validates incoming event type
func ValidateEvent(event Event) error {
	if event == nil {
		return fmt.Errorf("%w Event", gctcommon.ErrNilPointer)
	}
	if event.GetExchange() == "" ||
		event.GetAssetType().IsValid() ||
		event.Pair().IsEmpty() ||
		event.GetTime().IsZero() ||
		event.GetInterval() == 0 {
		return ErrInvalidEventSupplied
	}
	return nil
}

// CompareEvent determines if the previous and current are in the correct order
// (time aligned) and have the same currency details.
func CompareEvent(prev, curr Event) error {
	if prev.GetTime().Equal(curr.GetTime()) {
		return errDuplicateEvent
	}

	if prev.GetTime().After(curr.GetTime()) {
		return errEventsNotTimeAligned
	}

	if prev.GetExchange() != curr.GetExchange() ||
		prev.GetAssetType() != curr.GetAssetType() ||
		!prev.Pair().Equal(curr.Pair()) ||
		prev.GetInterval() != curr.GetInterval() {
		return fmt.Errorf("%w cannot set base stream from %v %v %v %v to %v %v %v %v",
			errMisMatchedEvent,
			curr.GetExchange(),
			curr.GetAssetType(),
			curr.Pair(),
			curr.GetInterval(),
			prev.GetExchange(),
			prev.GetAssetType(),
			prev.Pair(),
			prev.GetInterval())
	}
	return nil
}
