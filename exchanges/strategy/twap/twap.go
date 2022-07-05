package twap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/strategy"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	errParamsAreNil                   = errors.New("params are nil")
	errInvalidVolume                  = errors.New("invalid volume")
	errInvalidMaxSlippageValue        = errors.New("invalid max slippage percentage value, need to be between 0 and 100")
	errExchangeIsNil                  = errors.New("exchange is nil")
	errTWAPIsNil                      = errors.New("twap is nil")
	errNoBalanceFound                 = errors.New("no balance found")
	errVolumeToSellExceedsFreeBalance = errors.New("volume to sell exceeds free balance")
	errConfigurationIsNil             = errors.New("strategy configuration is nil")

	errExceedsFreeBalance = errors.New("amount exceeds current free balance")
)

// Strategy defines a TWAP strategy that handles the accumulation/de-accumulation
// of assets via a time weighted average price.
type Strategy struct {
	strategy.Base
	*Config
	holdings        map[currency.Code]*account.ProtectedBalance
	Reporter        chan Report
	Candles         kline.Item
	orderbook       *orderbook.Depth
	ExecutionLimits order.MinMaxLevel

	AmountPerAction float64
}

// GetTWAP returns a TWAP struct to manage TWAP allocation or deallocation of
// position.
func New(ctx context.Context, p *Config) (*Strategy, error) {
	if err := p.Check(); err != nil {
		return nil, err
	}

	// Gets tranche levels for liquidity options
	depth, err := orderbook.GetDepth(p.Exchange.GetName(), p.Pair, p.Asset)
	if err != nil {
		return nil, err
	}

	creds, err := p.Exchange.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}

	baseAmount, err := account.GetBalance(p.Exchange.GetName(),
		creds.SubAccount, p.Asset, p.Pair.Base)
	if err != nil {
		return nil, err
	}

	minMaxLevel, err := p.Exchange.GetOrderExecutionLimits(p.Asset, p.Pair)
	if err != nil {
		return nil, err
	}

	fmt.Println("Base amount", baseAmount.GetAvailableWithoutBorrow(), p.Pair.Base)

	quoteAmount, err := account.GetBalance(p.Exchange.GetName(),
		creds.SubAccount, p.Asset, p.Pair.Quote)
	if err != nil {
		return nil, err
	}

	fmt.Println("Quote amount", quoteAmount.GetAvailableWithoutBorrow(), p.Pair.Quote)

	if p.Accumulation {
		freeQuote := quoteAmount.GetAvailableWithoutBorrow()
		if p.Amount > freeQuote {
			return nil, fmt.Errorf("cannot sell quote %s amount %v to buy base %s %w of %v",
				p.Pair.Quote,
				p.Amount,
				p.Pair.Base,
				errExceedsFreeBalance,
				freeQuote)
		}
	} else {
		freeBase := baseAmount.GetAvailableWithoutBorrow()
		if p.Amount > freeBase {
			return nil, fmt.Errorf("cannot sell base %s amount %v to buy quote %s %w of %v",
				p.Pair.Base,
				p.Amount,
				p.Pair.Quote,
				errExceedsFreeBalance,
				freeBase)
		}
	}

	monAmounts := map[currency.Code]*account.ProtectedBalance{
		p.Pair.Base:  baseAmount,
		p.Pair.Quote: quoteAmount,
	}

	fmt.Printf("start requested %s, aligning to strategy interval %s\n", p.Start, p.StrategyInterval)

	fmt.Printf("end requested %s\n", p.End)

	fmt.Printf("start requested local %s\n", p.Start)

	fmt.Printf("Splitting amount %v across interval\n", p.Amount)

	expectedDuration := p.End.Sub(p.Start)

	fmt.Printf("expected strategy duration %s\n", expectedDuration)

	expectedActions := expectedDuration / p.StrategyInterval.Duration()

	// Drop mantissa as it is not needed after the last action we can use left
	// over amounts up till the amounts per action if we surpass end date for
	// whatever reason.
	wholeActions := int(expectedActions)

	fmt.Printf("expected strategy actions %d\n", wholeActions)

	amountPerAction := p.Amount / float64(wholeActions)

	fmt.Printf("amount per action %v\n", amountPerAction)

	if p.Accumulation {
		fmt.Println(minMaxLevel.MinNotional)
		if minMaxLevel.MinNotional > amountPerAction {
			return nil, fmt.Errorf("minimum quote %v exceeds amount per action %v",
				minMaxLevel.MinNotional, amountPerAction)
		}
	} else {
		if minMaxLevel.MinAmount > amountPerAction {
			return nil, fmt.Errorf("minimum base %v exceeds amount per action %v",
				minMaxLevel.MinAmount, amountPerAction)
		}
	}

	return &Strategy{
		Config:          p,
		Reporter:        make(chan Report),
		orderbook:       depth,
		holdings:        monAmounts,
		AmountPerAction: amountPerAction,
	}, nil
}

// Run inititates a TWAP allocation using the specified paramaters.
func (t *Strategy) Run(ctx context.Context) error {
	if t == nil {
		return errTWAPIsNil
	}

	if t.Config == nil {
		return errConfigurationIsNil
	}

	wait := time.Until(t.Start)
	timer := time.NewTimer(wait)

	for {
		select {
		case <-timer.C:
			// Reset timer here so we don't start drifting
			timer.Reset(t.StrategyInterval.Duration())
			signalEnd := time.Now().UTC().Truncate(t.SignalInterval.Duration())
			fmt.Printf("signal end time truncated to singal interval %v %s\n", signalEnd, t.SignalInterval)
			signalStart := signalEnd.Add(-t.SignalInterval.Duration() * time.Duration(t.SignalLookback))
			fmt.Printf("signal start time from lookback period %s lookback:%v\n", signalStart, t.SignalLookback)
			candles, err := t.Exchange.GetHistoricCandlesExtended(ctx,
				t.Pair,
				t.Asset,
				signalStart,
				signalEnd,
				t.SignalInterval)
			if err != nil {
				return err
			}

			fmt.Println(candles)

			fmt.Printf("sleeping for %s...", t.StrategyInterval.Duration())
		case <-t.Shutdown:
			if !timer.Stop() {
				<-timer.C
			}
			return nil
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			log.Errorf(log.Global, "twap strategy error: %s", ctx.Err())
			return ctx.Err()
		}
	}
}

// Report defines a TWAP action
type Report struct {
	Order    order.SubmitResponse
	TWAP     float64
	Slippage float64
	Error    error
	Finished bool
	Balance  map[currency.Code]float64
	Info     interface{}
}
