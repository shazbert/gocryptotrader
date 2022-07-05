package twap

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// Config defines the base elements required to undertake the TWAP strategy.
type Config struct {
	Exchange exchange.IBotExchange
	Pair     currency.Pair
	Asset    asset.Item

	Start time.Time
	End   time.Time

	// StrategyInterval defines the heartbeat of the strategy
	StrategyInterval kline.Interval

	// SignalInterval defines the interval period for singal generation
	SignalInterval kline.Interval

	// SignalLookback defines the signal lookback period to construct an
	// adequate signal.
	SignalLookback float64

	// Amount if accumulating refers to quotation used to buy, if deaccum it
	// will refer to the base amount to sell
	Amount float64

	// MaxSlippage needed for protection in low liqudity environments.
	// WARNING: 0 value == 100% slippage
	MaxSlippage float64
	// Accumulation if you are buying or selling value
	Accumulation bool
	// AllowTradingPastEndTime if volume has not been met exceed end time.
	AllowTradingPastEndTime bool
}

// Check validates all parameter fields before undertaking specfic strategy
func (cfg *Config) Check() error {
	if cfg == nil {
		return errParamsAreNil
	}

	if cfg.Exchange == nil {
		return errExchangeIsNil
	}

	if cfg.Pair.IsEmpty() {
		return currency.ErrPairIsEmpty
	}

	if !cfg.Asset.IsValid() {
		return fmt.Errorf("'%v' %w", cfg.Asset, asset.ErrNotSupported)
	}

	err := common.StartEndTimeCheck(cfg.Start, cfg.End)
	if err != nil {
		return err
	}

	if cfg.StrategyInterval <= 0 {
		return fmt.Errorf("strategy interval %w", kline.ErrUnsetInterval)
	}

	if cfg.SignalInterval <= 0 {
		return fmt.Errorf("signal interval %w", kline.ErrUnsetInterval)
	}

	err = cfg.Exchange.GetBase().ValidateKline(cfg.Pair, cfg.Asset, cfg.SignalInterval)
	if err != nil {
		return fmt.Errorf("strategy interval %w", err)
	}

	if cfg.Amount <= 0 {
		return errInvalidVolume
	}

	if cfg.MaxSlippage < 0 || cfg.MaxSlippage > 100 {
		return fmt.Errorf("'%v' %w", cfg.MaxSlippage, errInvalidMaxSlippageValue)
	}
	return nil
}
