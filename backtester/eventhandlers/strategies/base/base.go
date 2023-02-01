package base

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
)

var (
	errSimultaneousProcessingNotSupported = errors.New("simultaneous processing not supported by strategy")
	errSimultaneousProcessingAlreadySet   = errors.New("simultaneous processing already set")
	errStrategyNameUnset                  = errors.New("strategy name unset")
	errStrategyDescriptionUnset           = errors.New("strategy description unset")
)

// Strategy is base implementation of the Handler interface
type Strategy struct {
	useSimultaneousProcessing        bool
	CanSupportSimultaneousProcessing bool
	Name                             string
	Description                      string
}

// GetBaseData returns the non-interface version of the Handler
func (s *Strategy) GetBaseData(d data.Handler) (*signal.Signal, error) {
	if s == nil {
		return nil, fmt.Errorf("%w strategy", gctcommon.ErrNilPointer)
	}
	if d == nil {
		return nil, fmt.Errorf("%w incoming data.Handler", gctcommon.ErrNilPointer)
	}
	latest, err := d.Latest()
	if err != nil {
		return nil, err
	}
	if latest == nil {
		return nil, common.ErrNilEvent
	}
	return &signal.Signal{
		Base:       latest.GetBase(),
		ClosePrice: latest.GetClosePrice(),
		HighPrice:  latest.GetHighPrice(),
		OpenPrice:  latest.GetOpenPrice(),
		LowPrice:   latest.GetLowPrice(),
	}, nil
}

// UsingSimultaneousProcessing returns whether multiple currencies can be
// assessed in one go.
func (s *Strategy) UsingSimultaneousProcessing() (bool, error) {
	if s == nil {
		return false, fmt.Errorf("%w strategy", gctcommon.ErrNilPointer)
	}
	return s.CanSupportSimultaneousProcessing && s.useSimultaneousProcessing, nil
}

// SetSimultaneousProcessing sets whether multiple currencies can be assessed in
// one go.
func (s *Strategy) SetSimultaneousProcessing(b bool) error {
	if s == nil {
		return fmt.Errorf("%w strategy", gctcommon.ErrNilPointer)
	}

	if !s.CanSupportSimultaneousProcessing && b {
		return errSimultaneousProcessingNotSupported
	}

	if s.useSimultaneousProcessing == b {
		return errSimultaneousProcessingAlreadySet
	}

	s.useSimultaneousProcessing = b
	return nil
}

// CloseAllPositions sends a closing signal to supported strategies, allowing
// them to sell off any positions held default use-case is for when a user
// closes the application when running a live strategy.
func (s *Strategy) CloseAllPositions([]holdings.Holding, []data.Event) ([]signal.Event, error) {
	return nil, gctcommon.ErrFunctionNotSupported
}

// GetName returns the name of the strategy
func (s *Strategy) GetName() (string, error) {
	if s == nil {
		return "", fmt.Errorf("%w strategy", gctcommon.ErrNilPointer)
	}
	if s.Name == "" {
		return "", errStrategyNameUnset
	}
	return s.Name, nil
}

// GetDescription provides a clear and comprehensive overview of the strategy,
// including the definition of terms and a clear explanation of its purpose.
func (s *Strategy) GetDescription() (string, error) {
	if s == nil {
		return "", fmt.Errorf("%w strategy", gctcommon.ErrNilPointer)
	}
	if s.Description == "" {
		return "", errStrategyDescriptionUnset
	}
	return s.Description, nil
}

// SupportsSimultaneousProcessing highlights whether the strategy can handle
// multiple currency calculations.
func (s *Strategy) SupportsSimultaneousProcessing() (bool, error) {
	if s == nil {
		return false, fmt.Errorf("%w strategy", gctcommon.ErrNilPointer)
	}
	return s.CanSupportSimultaneousProcessing, nil
}
