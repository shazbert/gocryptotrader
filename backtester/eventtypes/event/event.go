package event

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
	// TODO: Shift to backtester common
	errExchangeNameUnset          = errors.New("exchange name unset")
	errInvalidInterval            = errors.New("invalid interval")
	errInvalidOffset              = errors.New("invalid offset")
	errTimeUnset                  = errors.New("time is unset")
	errTimeShouldBeUTC            = errors.New("time should be UTC")
	errTimeNotTruncatedToInterval = errors.New("time not truncated to interval")
)

// Base is the underlying event across all actions that occur for the backtester
// Data, fill, order events all contain the base event and store important and
// consistent information
type Base struct {
	Offset         int64             `json:"-"` // TODO: RM
	Exchange       string            `json:"exchange"`
	Time           time.Time         `json:"timestamp"`
	Interval       gctkline.Interval `json:"interval-size"`
	CurrencyPair   currency.Pair     `json:"pair"`
	UnderlyingPair currency.Pair     `json:"underlying"`
	AssetType      asset.Item        `json:"asset"`
	Reasons        []string          `json:"reasons"`
}

// NewBaseFromKline
func NewBaseFromKline(k *gctkline.Item, t time.Time, offset int64) (*Base, error) {
	if k == nil {
		return nil, fmt.Errorf("%w for %T", gctcommon.ErrNilPointer, k)
	}

	if k.Exchange == "" {
		return nil, fmt.Errorf("%w for %T", errExchangeNameUnset, k)
	}

	if k.Pair.IsEmpty() {
		return nil, fmt.Errorf("%w for %T", currency.ErrCurrencyPairEmpty, k)
	}

	if !k.Asset.IsValid() {
		return nil, fmt.Errorf("%w for %T", asset.ErrNotSupported, k)
	}

	if k.Interval <= 0 {
		return nil, fmt.Errorf("%w for %T", errInvalidInterval, k)
	}

	if t.IsZero() {
		return nil, errTimeUnset
	}

	if t.Location() != time.UTC {
		return nil, errTimeShouldBeUTC
	}

	if !t.Equal(t.Truncate(k.Interval.Duration())) {
		return nil, errTimeNotTruncatedToInterval
	}

	if offset < 0 {
		return nil, errInvalidOffset
	}

	return &Base{
		Offset:         offset,
		Exchange:       k.Exchange,
		Time:           t,
		Interval:       k.Interval,
		CurrencyPair:   k.Pair,
		AssetType:      k.Asset,
		UnderlyingPair: k.UnderlyingPair,
	}, nil
}

// GetOffset returns the offset
func (b *Base) GetOffset() int64 {
	return b.Offset
}

// SetOffset sets the offset
func (b *Base) SetOffset(o int64) {
	b.Offset = o
}

// IsEvent returns whether the event is an event
func (b *Base) IsEvent() bool {
	return true
}

// GetTime returns the time
func (b *Base) GetTime() time.Time {
	return b.Time.UTC()
}

// Pair returns the currency pair
func (b *Base) Pair() currency.Pair {
	return b.CurrencyPair
}

// GetUnderlyingPair returns the currency pair
func (b *Base) GetUnderlyingPair() currency.Pair {
	return b.UnderlyingPair
}

// GetExchange returns the exchange
func (b *Base) GetExchange() string {
	return strings.ToLower(b.Exchange)
}

// GetAssetType returns the asset type
func (b *Base) GetAssetType() asset.Item {
	return b.AssetType
}

// GetInterval returns the interval
func (b *Base) GetInterval() gctkline.Interval {
	return b.Interval
}

// AppendReason adds reasoning for a decision being made
func (b *Base) AppendReason(y string) {
	b.Reasons = append(b.Reasons, y)
}

// AppendReasonf adds reasoning for a decision being made
// but with formatting
func (b *Base) AppendReasonf(y string, addons ...interface{}) {
	y = fmt.Sprintf(y, addons...)
	b.Reasons = append(b.Reasons, y)
}

// GetConcatReasons returns the why
func (b *Base) GetConcatReasons() string {
	return strings.Join(b.Reasons, ". ")
}

// GetReasons returns each individual reason
func (b *Base) GetReasons() []string {
	return b.Reasons
}

// GetBase returns the underlying base
func (b *Base) GetBase() *Base {
	return b
}
