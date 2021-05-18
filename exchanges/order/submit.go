package order

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/validate"
)

// Submit contains all properties of an order that may be required
// for an order to be created on an exchange
// Each exchange has their own requirements, so not all fields
// are required to be populated
type Submit struct {
	// Time In Force
	FillOrKill        bool
	ImmediateOrCancel bool

	HiddenOrder bool
	PostOnly    bool
	ReduceOnly  bool

	Leverage float64
	Price    float64 // Only needed in a limit order
	Amount   float64

	StopPrice       float64
	LimitPriceUpper float64
	LimitPriceLower float64
	TriggerPrice    float64
	TargetAmount    float64
	ExecutedAmount  float64
	RemainingAmount float64
	Fee             float64
	Exchange        string
	InternalOrderID string
	ID              string
	AccountID       string
	ClientID        string
	ClientOrderID   string
	WalletAddress   string
	Offset          string
	Type            Type
	Side            Side
	Status          Status
	AssetType       asset.Item
	Date            time.Time
	LastUpdated     time.Time
	Pair            currency.Pair
	// Trades            []TradeHistory
	Account string
	// TotalAmountNotRequired defines the requirement of the amount to be
	// claimed on actual holdings. If not required a claim will be made on what
	// is actually available for the calling subsystem.
	TotalAmountNotRequired bool
}

// Validate checks the supplied data and returns whether or not it's valid
func (s *Submit) Validate(opt ...validate.Checker) error {
	if s == nil {
		return ErrSubmissionIsNil
	}

	if s.Pair.IsEmpty() {
		return ErrPairIsEmpty
	}

	if s.AssetType == "" {
		return ErrAssetNotSet
	}

	if s.Side != Buy &&
		s.Side != Sell &&
		s.Side != Bid &&
		s.Side != Ask {
		return ErrSideIsInvalid
	}

	if s.Type != Market && s.Type != Limit {
		return ErrTypeIsInvalid
	}

	if s.Amount <= 0 {
		return fmt.Errorf("submit validation error %w, suppled: %.8f",
			ErrAmountIsInvalid,
			s.Amount)
	}

	if s.Type == Limit && s.Price <= 0 {
		return ErrPriceMustBeSetIfLimitOrder
	}

	for _, o := range opt {
		err := o.Check()
		if err != nil {
			return err
		}
	}

	return nil
}
