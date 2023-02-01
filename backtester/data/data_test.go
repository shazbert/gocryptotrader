package data

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const exch = "binance"

type fakeEvent struct {
	secretID int64
	*event.Base
}

func (f fakeEvent) GetClosePrice() decimal.Decimal { return decimal.Zero }
func (f fakeEvent) GetHighPrice() decimal.Decimal  { return decimal.Zero }
func (f fakeEvent) GetLowPrice() decimal.Decimal   { return decimal.Zero }
func (f fakeEvent) GetOpenPrice() decimal.Decimal  { return decimal.Zero }
func (f fakeEvent) GetVolume() decimal.Decimal     { return decimal.Zero }
func (f fakeEvent) GetTime() time.Time             { return f.Time }
func (f fakeEvent) GetOffset() int64 {
	if f.secretID > 0 {
		return f.secretID
	}
	return f.Offset
}

type fakeHandler struct{ Handler }

var (
	p                   = currency.NewPair(currency.BTC, currency.USD)
	tnUTCOneHourAligned = time.Now().UTC().Truncate(time.Hour)
	validEvents         = []Event{
		&fakeEvent{Base: &event.Base{
			Offset:       1337,
			Time:         tnUTCOneHourAligned.Add(-time.Hour),
			Exchange:     exch,
			AssetType:    asset.Spot,
			Interval:     gctkline.OneHour,
			CurrencyPair: p,
		}},
		&fakeEvent{Base: &event.Base{
			Offset:       2048,
			Time:         tnUTCOneHourAligned,
			Exchange:     exch,
			AssetType:    asset.Spot,
			Interval:     gctkline.OneHour,
			CurrencyPair: p,
		}},
	}
)

func TestNewHandlerHolder(t *testing.T) {
	t.Parallel()
	holder := NewHandlerHolder()
	if holder == nil {
		t.Errorf("received '%v' expected '%v'", holder, "not nil")
	}

	if holder.data == nil {
		t.Errorf("received '%v' expected '%v'", holder.data, "not nil")
	}
}

func TestSetDataForCurrency(t *testing.T) {
	t.Parallel()
	var d *HandlerHolder
	err := d.SetDataForCurrency(exch, asset.Spot, p, gctkline.OneHour, nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	d = &HandlerHolder{}

	err = d.SetDataForCurrency("", asset.Spot, p, gctkline.OneHour, nil)
	if !errors.Is(err, errExchangeNameUnset) {
		t.Errorf("received '%v' expected '%v'", err, errExchangeNameUnset)
	}

	err = d.SetDataForCurrency(exch, 0, p, gctkline.OneHour, nil)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received '%v' expected '%v'", err, asset.ErrNotSupported)
	}

	err = d.SetDataForCurrency(exch, asset.Spot, currency.EMPTYPAIR, gctkline.OneHour, nil)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("received '%v' expected '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	err = d.SetDataForCurrency(exch, asset.Spot, p, 0, nil)
	if !errors.Is(err, gctkline.ErrInvalidInterval) {
		t.Errorf("received '%v' expected '%v'", err, gctkline.ErrInvalidInterval)
	}

	err = d.SetDataForCurrency(exch, asset.Spot, p, gctkline.OneHour, nil)
	if !errors.Is(err, errHandlerIsNil) {
		t.Errorf("received '%v' expected '%v'", err, errHandlerIsNil)
	}

	err = d.SetDataForCurrency(exch, asset.Spot, p, gctkline.OneHour, &fakeHandler{})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	if d.data == nil {
		t.Error("expected not nil")
	}

	if d.data[exch][asset.Spot][p.Base.Item][p.Quote.Item][gctkline.OneHour] == nil {
		t.Error("should not be nil")
	}

	err = d.SetDataForCurrency(exch, asset.Spot, p, gctkline.OneHour, &fakeHandler{})
	if !errors.Is(err, errDataHandlerAlreadySet) {
		t.Errorf("received '%v' expected '%v'", err, errDataHandlerAlreadySet)
	}
}

func TestGetAllData(t *testing.T) {
	t.Parallel()
	var d *HandlerHolder
	_, err := d.GetAllData()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	err = d.SetDataForCurrency(exch, asset.Spot, p, gctkline.OneHour, &fakeHandler{})
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	d = &HandlerHolder{}
	err = d.SetDataForCurrency(exch, asset.Spot, p, gctkline.OneHour, &fakeHandler{})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = d.SetDataForCurrency(exch, asset.Spot, currency.NewPair(currency.BTC, currency.DOGE), gctkline.OneHour, &fakeHandler{})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = d.SetDataForCurrency(exch, asset.Spot, currency.NewPair(currency.BTC, currency.DOGE), gctkline.ThreeHour, &fakeHandler{})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	result, err := d.GetAllData()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	if len(result) != 2 {
		t.Error("expected 2")
	}

	if len(result[1]) != 2 {
		t.Error("expected 2")
	}
}

func TestGetDataForCurrency(t *testing.T) {
	t.Parallel()

	var d *HandlerHolder
	_, err := d.GetDataForCurrency("", 0, currency.EMPTYPAIR)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	d = &HandlerHolder{}
	err = d.SetDataForCurrency(exch, asset.Spot, p, gctkline.OneHour, &fakeHandler{})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	_, err = d.GetDataForCurrency("", 0, currency.EMPTYPAIR)
	if !errors.Is(err, errExchangeNameUnset) {
		t.Errorf("received '%v' expected '%v'", err, errExchangeNameUnset)
	}

	_, err = d.GetDataForCurrency("lol", 0, currency.EMPTYPAIR)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received '%v' expected '%v'", err, asset.ErrNotSupported)
	}

	_, err = d.GetDataForCurrency("lol", asset.USDTMarginedFutures, currency.EMPTYPAIR)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Errorf("received '%v' expected '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	_, err = d.GetDataForCurrency("lol", asset.USDTMarginedFutures, currency.NewPair(currency.EMB, currency.DOGE))
	if !errors.Is(err, ErrHandlerNotFound) {
		t.Errorf("received '%v' expected '%v'", err, ErrHandlerNotFound)
	}

	_, err = d.GetDataForCurrency(exch, asset.Spot, p)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
}

func TestReset(t *testing.T) {
	t.Parallel()
	d := &HandlerHolder{}
	err := d.SetDataForCurrency(exch, asset.Spot, p, gctkline.OneHour, &fakeHandler{})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = d.SetDataForCurrency(exch, asset.Spot, currency.NewPair(currency.BTC, currency.DOGE), gctkline.OneHour, &fakeHandler{})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	err = d.Reset()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if d.data == nil {
		t.Error("expected a map")
	}
	d = nil
	err = d.Reset()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestBaseReset(t *testing.T) {
	t.Parallel()
	b := &Base{offset: 1}
	err := b.Reset()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if b.offset != 0 {
		t.Errorf("received '%v' expected '%v'", b.offset, 0)
	}
	b = nil
	err = b.Reset()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestGetDetails(t *testing.T) {
	t.Parallel()
	var b *Base
	_, err := b.GetDetails()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	b = &Base{}
	_, err = b.GetDetails()
	// This might need to be changed, maybe find the latest in the event slice.
	if !errors.Is(err, errLatestEventHasNotBeenSet) {
		t.Errorf("received '%v' expected '%v'", err, errLatestEventHasNotBeenSet)
	}

	b.latest = fakeEvent{
		Base: &event.Base{
			Exchange:     exch,
			CurrencyPair: p,
			AssetType:    asset.Spot,
			Interval:     gctkline.TenMin,
		},
	}
	deets, err := b.GetDetails()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	if deets.ExchangeName != exch {
		t.Errorf("received '%v' expected '%v'", deets.ExchangeName, exch)
	}

	if deets.Asset != asset.Spot {
		t.Errorf("received '%v' expected '%v'", deets.Asset, asset.Spot)
	}

	if !deets.Pair.Equal(p) {
		t.Errorf("received '%v' expected '%v'", deets.Pair, p)
	}

	if deets.Interval != gctkline.TenMin {
		t.Errorf("received '%v' expected '%v'", deets.Interval, gctkline.TenMin)
	}
}

func TestGetStream(t *testing.T) {
	t.Parallel()
	b := &Base{}
	resp, err := b.GetStream()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(resp) != 0 {
		t.Errorf("received '%v' expected '%v'", len(resp), 0)
	}
	b.stream = []Event{
		&fakeEvent{
			Base: &event.Base{
				Offset: 2048,
				Time:   time.Now(),
			},
		},
		&fakeEvent{
			Base: &event.Base{
				Offset: 1337,
				Time:   time.Now().Add(-time.Hour),
			},
		},
	}
	resp, err = b.GetStream()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(resp) != 2 {
		t.Errorf("received '%v' expected '%v'", len(resp), 2)
	}

	b = nil
	_, err = b.GetStream()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestOffset(t *testing.T) {
	t.Parallel()
	b := &Base{}
	o, err := b.Offset()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if o != 0 {
		t.Errorf("received '%v' expected '%v'", o, 0)
	}
	b.offset = 1337
	o, err = b.Offset()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if o != 1337 {
		t.Errorf("received '%v' expected '%v'", o, 1337)
	}

	b = nil
	_, err = b.Offset()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestSetStream(t *testing.T) {
	t.Parallel()
	b := &Base{}
	err := b.SetStream(nil)
	if !errors.Is(err, ErrEmptySlice) {
		t.Fatalf("received '%v' expected '%v'", err, ErrEmptySlice)
	}

	containsInvalidEvent := []Event{
		&fakeEvent{Base: &event.Base{
			Offset: 2048,
		}},
	}

	err = b.SetStream(containsInvalidEvent)
	if !errors.Is(err, ErrInvalidEventSupplied) {
		t.Fatalf("received '%v' expected '%v'", err, ErrInvalidEventSupplied)
	}

	tn := time.Now().UTC().Truncate(gctkline.OneHour.Duration())

	unalignedEvents := []Event{
		&fakeEvent{Base: &event.Base{
			Offset:       2048,
			Time:         tn,
			Exchange:     exch,
			AssetType:    asset.Spot,
			Interval:     gctkline.OneHour,
			CurrencyPair: p,
		}},
		&fakeEvent{Base: &event.Base{
			Offset:       1337,
			Time:         tn.Add(-time.Hour),
			Exchange:     exch,
			AssetType:    asset.Spot,
			Interval:     gctkline.OneHour,
			CurrencyPair: p,
		}},
	}

	err = b.SetStream(unalignedEvents)
	if !errors.Is(err, errEventsNotTimeAligned) {
		t.Fatalf("received '%v' expected '%v'", err, errEventsNotTimeAligned)
	}

	err = b.SetStream(validEvents)
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}

	if len(b.stream) != 2 {
		t.Fatalf("received '%v' expected '%v'", len(b.stream), 2)
	}
	if b.stream[0].GetOffset() != 1 {
		t.Fatalf("received '%v' expected '%v'", b.stream[0].GetOffset(), 1)
	}

	err = b.SetStream(validEvents)
	if !errors.Is(err, errEventsAlreadySet) {
		t.Fatalf("received '%v' expected '%v'", err, errEventsAlreadySet)
	}

	b = nil
	err = b.SetStream(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Fatalf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestNext(t *testing.T) {
	t.Parallel()
	b := &Base{}
	err := b.SetStream(validEvents)
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}
	resp, err := b.Next()
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}
	if resp != b.stream[0] {
		t.Fatalf("received '%v' expected '%v'", resp, b.stream[0])
	}
	if b.offset != 1 {
		t.Fatalf("received '%v' expected '%v'", b.offset, 1)
	}
	_, err = b.Next()
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}
	resp, err = b.Next()
	if !errors.Is(err, ErrEndOfData) {
		t.Fatalf("received '%v' expected '%v'", err, ErrEndOfData)
	}
	if resp != nil {
		t.Fatalf("received '%v' expected '%v'", resp, nil)
	}

	b.offset = 420 // <- offset went out and got on the beers
	_, err = b.Next()
	if !errors.Is(err, errOffsetShifted) {
		t.Fatalf("received '%v' expected '%v'", err, errOffsetShifted)
	}

	b = nil
	_, err = b.Next()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Fatalf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestNextByTime(t *testing.T) {
	t.Parallel()
	b := &Base{}
	err := b.SetStream(validEvents)
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}

	_, err = b.NextByTime(time.Time{})
	if !errors.Is(err, errTimeIsUnset) {
		t.Fatalf("received '%v' expected '%v'", err, errTimeIsUnset)
	}

	_, err = b.NextByTime(time.Now())
	if !errors.Is(err, errTimeMustBeUTC) {
		t.Fatalf("received '%v' expected '%v'", err, errTimeMustBeUTC)
	}

	firstEventTime := tnUTCOneHourAligned.Add(-time.Hour)
	resp, err := b.NextByTime(firstEventTime)
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}
	if resp != b.stream[0] {
		t.Fatalf("received '%v' expected '%v'", resp, b.stream[0])
	}
	if b.offset != 1 {
		t.Fatalf("received '%v' expected '%v'", b.offset, 1)
	}
	_, err = b.NextByTime(tnUTCOneHourAligned)
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}
	resp, err = b.NextByTime(tnUTCOneHourAligned)
	if !errors.Is(err, ErrEndOfData) {
		t.Fatalf("received '%v' expected '%v'", err, ErrEndOfData)
	}
	if resp != nil {
		t.Fatalf("received '%v' expected '%v'", resp, nil)
	}

	b.offset = 420
	_, err = b.NextByTime(tnUTCOneHourAligned)
	if !errors.Is(err, errOffsetShifted) {
		t.Fatalf("received '%v' expected '%v'", err, errOffsetShifted)
	}

	b = nil
	_, err = b.NextByTime(tnUTCOneHourAligned)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Fatalf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestHistory(t *testing.T) {
	t.Parallel()
	b := &Base{}
	err := b.SetStream(validEvents)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	resp, err := b.History()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(resp) != 0 {
		t.Errorf("received '%v' expected '%v'", len(resp), 0)
	}

	_, err = b.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	resp, err = b.History()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(resp) != 1 {
		t.Errorf("received '%v' expected '%v'", len(resp), 1)
	}

	b.offset = 420
	_, err = b.History()
	if !errors.Is(err, errOffsetShifted) {
		t.Fatalf("received '%v' expected '%v'", err, errOffsetShifted)
	}

	b = nil
	_, err = b.History()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestLatest(t *testing.T) {
	t.Parallel()
	b := &Base{}
	_, err := b.Latest()
	if !errors.Is(err, errNoDataEventsLoaded) {
		t.Errorf("received '%v' expected '%v'", err, errNoDataEventsLoaded)
	}
	err = b.SetStream(validEvents)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	resp, err := b.Latest()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if resp != b.stream[0] {
		t.Errorf("received '%v' expected '%v'", resp, b.stream[0])
	}
	_, err = b.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	resp, err = b.Latest()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if resp != b.stream[0] {
		t.Errorf("received '%v' expected '%v'", resp, b.stream[0])
	}

	_, err = b.Next()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	resp, err = b.Latest()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if resp != b.stream[1] {
		t.Errorf("received '%v' expected '%v'", resp, b.stream[1])
	}

	// b.offset = 420
	// _, err = b.Latest()
	// if !errors.Is(err, errOffsetShifted) {
	// 	t.Fatalf("received '%v' expected '%v'", err, errOffsetShifted)
	// }

	b = nil
	_, err = b.Latest()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestList(t *testing.T) {
	t.Parallel()
	b := &Base{}
	err := b.SetStream(validEvents)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	list, err := b.List()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if len(list) != 2 {
		t.Errorf("received '%v' expected '%v'", len(list), 2)
	}

	b.offset = 420
	_, err = b.List()
	if !errors.Is(err, errOffsetShifted) {
		t.Fatalf("received '%v' expected '%v'", err, errOffsetShifted)
	}

	b = nil
	_, err = b.List()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestIsLastEvent(t *testing.T) {
	t.Parallel()
	b := &Base{}
	err := b.SetStream(validEvents)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}

	isLastEvent, err := b.IsLastEvent()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if isLastEvent {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	b.isLiveData = true
	isLastEvent, err = b.IsLastEvent()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if isLastEvent {
		t.Errorf("received '%v' expected '%v'", false, true)
	}

	b = nil
	_, err = b.IsLastEvent()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestIsLive(t *testing.T) {
	t.Parallel()
	b := &Base{}
	isLive, err := b.IsLive()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if isLive {
		t.Error("expected false")
	}
	b.isLiveData = true
	isLive, err = b.IsLive()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !isLive {
		t.Error("expected true")
	}

	b = nil
	_, err = b.IsLive()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestSetLive(t *testing.T) {
	t.Parallel()
	b := &Base{}
	err := b.SetLive(true)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if !b.isLiveData {
		t.Error("expected true")
	}

	err = b.SetLive(true)
	if !errors.Is(err, errDataFeedTypeHasAlreadyBeenSet) {
		t.Errorf("received '%v' expected '%v'", err, errDataFeedTypeHasAlreadyBeenSet)
	}

	err = b.SetLive(false)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if b.isLiveData {
		t.Error("expected false")
	}

	b = nil
	err = b.SetLive(false)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestAppendStream(t *testing.T) {
	t.Parallel()
	b := &Base{}
	err := b.AppendStream(&fakeEvent{Base: &event.Base{}})
	if !errors.Is(err, ErrInvalidEventSupplied) {
		t.Errorf("received '%v' expected '%v'", err, ErrInvalidEventSupplied)
	}
	if len(b.stream) != 0 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 0)
	}

	err = b.AppendStream(validEvents[0], validEvents[0])
	if !errors.Is(err, errDuplicateEvent) {
		t.Fatalf("received '%v' expected '%v'", err, errDuplicateEvent)
	}
	if len(b.stream) != 0 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 1)
	}

	err = b.AppendStream(validEvents[0])
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}
	if len(b.stream) != 1 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 1)
	}

	err = b.AppendStream(validEvents[0]) // <-- This is duplicate from last appended event
	if !errors.Is(err, errDuplicateEvent) {
		t.Fatalf("received '%v' expected '%v'", err, errDuplicateEvent)
	}
	if len(b.stream) != 1 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 1)
	}

	err = b.AppendStream(validEvents[1])
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}
	if len(b.stream) != 2 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 2)
	}

	err = b.AppendStream(&fakeEvent{Base: &event.Base{
		Exchange:     "mismatch",
		CurrencyPair: currency.NewPair(currency.BTC, currency.DOGE),
		AssetType:    asset.Futures,
		Time:         tnUTCOneHourAligned.Add(time.Hour * 2),
		Interval:     gctkline.OneHour,
	}})
	if !errors.Is(err, errMisMatchedEvent) {
		t.Fatalf("received '%v' expected '%v'", err, errMisMatchedEvent)
	}
	if len(b.stream) != 2 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 2)
	}

	err = b.AppendStream(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Fatalf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
	if len(b.stream) != 2 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 2)
	}

	err = b.AppendStream()
	if !errors.Is(err, errNothingToAdd) {
		t.Fatalf("received '%v' expected '%v'", err, errNothingToAdd)
	}
	if len(b.stream) != 2 {
		t.Errorf("received '%v' expected '%v'", len(b.stream), 2)
	}

	b = nil
	err = b.AppendStream()
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Errorf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}
}

func TestFirst(t *testing.T) {
	t.Parallel()

	var e Events
	_, err := e.First()
	if !errors.Is(err, ErrEmptySlice) {
		t.Errorf("received '%v' expected '%v'", err, ErrEmptySlice)
	}

	e = Events{fakeEvent{secretID: 1}, fakeEvent{secretID: 2}, fakeEvent{secretID: 3}}
	first, err := e.First()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if first.GetOffset() != 1 {
		t.Errorf("received '%v' expected '%v'", first.GetOffset(), 1)
	}
}

func TestLast(t *testing.T) {
	t.Parallel()

	var e Events
	_, err := e.Last()
	if !errors.Is(err, ErrEmptySlice) {
		t.Errorf("received '%v' expected '%v'", err, ErrEmptySlice)
	}

	e = Events{fakeEvent{secretID: 1}, fakeEvent{secretID: 2}, fakeEvent{secretID: 3}}
	last, err := e.Last()
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expected '%v'", err, nil)
	}
	if last.GetOffset() != 3 {
		t.Errorf("received '%v' expected '%v'", last.GetOffset(), 3)
	}
}

func TestValidateEvent(t *testing.T) {
	t.Parallel()

	err := ValidateEvent(nil)
	if !errors.Is(err, gctcommon.ErrNilPointer) {
		t.Fatalf("received '%v' expected '%v'", err, gctcommon.ErrNilPointer)
	}

	err = ValidateEvent(fakeEvent{Base: &event.Base{}})
	if !errors.Is(err, ErrInvalidEventSupplied) {
		t.Fatalf("received '%v' expected '%v'", err, ErrInvalidEventSupplied)
	}

	err = ValidateEvent(fakeEvent{Base: &event.Base{
		Exchange:     exch,
		AssetType:    asset.Spot,
		CurrencyPair: p,
		Interval:     gctkline.OneHour,
		Time:         time.Now().Local(),
	}})
	if !errors.Is(err, errTimeMustBeUTC) {
		t.Fatalf("received '%v' expected '%v'", err, errTimeMustBeUTC)
	}

	err = ValidateEvent(fakeEvent{Base: &event.Base{
		Exchange:     exch,
		AssetType:    asset.Spot,
		CurrencyPair: p,
		Interval:     gctkline.OneHour,
		Time:         time.Now().UTC(),
	}})
	if !errors.Is(err, errEventTimeIntervalMismatch) {
		t.Fatalf("received '%v' expected '%v'", err, errEventTimeIntervalMismatch)
	}

	err = ValidateEvent(fakeEvent{Base: &event.Base{
		Exchange:     exch,
		AssetType:    asset.Spot,
		CurrencyPair: p,
		Interval:     gctkline.OneHour,
		Time:         time.Now().UTC().Truncate(time.Hour),
	}})
	if !errors.Is(err, nil) {
		t.Fatalf("received '%v' expected '%v'", err, nil)
	}
}
