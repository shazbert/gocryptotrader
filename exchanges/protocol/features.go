package protocol

import (
	"errors"
	"time"
)

var (
	// On infers functionality support and enabled
	On = func() *bool { b := true; return &b }()
	// Off infers functionality support and disabled
	Off = func() *bool { b := false; return &b }()
)

// Features stores the exchange supported protocol functionality
type Features struct {
	REST      *Components `json:"rest,omitempty"`
	Websocket *Components `json:"websocket,omitempty"`
	Fix       *Components `json:"fix,omitempty"`
}

// Permissions defines a set of allowable permissions
type Permissions uint32

// Components hold all variables for an exchange protocol functionality
// (e.g REST or Websocket)
type Components struct {
	// Enabled always used and viewable in config
	Enabled bool `json:"enabled"`

	// nil == unsupported for this protocol scheme
	TickerBatching         *bool       `json:"tickerBatching,omitempty"`
	AutoPairUpdates        *bool       `json:"autoPairUpdates,omitempty"`
	AccountBalance         *bool       `json:"accountBalance,omitempty"`
	CryptoDeposit          *bool       `json:"cryptoDeposit,omitempty"`
	CryptoWithdrawal       *bool       `json:"cryptoWithdrawal,omitempty"`
	FiatWithdraw           *bool       `json:"fiatWithdraw,omitempty"`
	GetOrder               *bool       `json:"getOrder,omitempty"`
	GetOrders              *bool       `json:"getOrders,omitempty"`
	CancelOrders           *bool       `json:"cancelOrders,omitempty"`
	CancelOrder            *bool       `json:"cancelOrder,omitempty"`
	SubmitOrder            *bool       `json:"submitOrder,omitempty"`
	SubmitOrders           *bool       `json:"submitOrders,omitempty"`
	ModifyOrder            *bool       `json:"modifyOrder,omitempty"`
	DepositHistory         *bool       `json:"depositHistory,omitempty"`
	WithdrawalHistory      *bool       `json:"withdrawalHistory,omitempty"`
	TradeFetching          *bool       `json:"tradeFetching,omitempty"`
	ExchangeTradeHistory   *bool       `json:"exchangeTradeHistory,omitempty"`
	UserTradeHistory       *bool       `json:"userTradeHistory,omitempty"`
	TradeFee               *bool       `json:"tradeFee,omitempty"`
	FiatDepositFee         *bool       `json:"fiatDepositFee,omitempty"`
	FiatWithdrawalFee      *bool       `json:"fiatWithdrawalFee,omitempty"`
	CryptoDepositFee       *bool       `json:"cryptoDepositFee,omitempty"`
	CryptoWithdrawalFee    *bool       `json:"cryptoWithdrawalFee,omitempty"`
	TickerFetching         *bool       `json:"tickerFetching,omitempty"`
	KlineFetching          *bool       `json:"klineFetching,omitempty"`
	OrderbookFetching      *bool       `json:"orderbookFetching,omitempty"`
	AccountInfo            *bool       `json:"accountInfo,omitempty"`
	FiatDeposit            *bool       `json:"fiatDeposit,omitempty"`
	DeadMansSwitch         *bool       `json:"deadMansSwitch,omitempty"`
	Subscribe              *bool       `json:"subscribe,omitempty"`
	Unsubscribe            *bool       `json:"unsubscribe,omitempty"`
	AuthenticatedEndpoints *bool       `json:"authenticatedEndpoints,omitempty"`
	MessageCorrelation     *bool       `json:"messageCorrelation,omitempty"`
	MessageSequenceNumbers *bool       `json:"messageSequenceNumbers,omitempty"`
	Withdraw               *uint32     `json:"-"`
	Limits                 *RateLimits `json:"-"`
}

// ProtocolSupported checks to see if the protocol is supported
func (c *Components) ProtocolSupported() bool {
	return c != nil
}

// TickerBatchingSupported checks if ticker batching functionality is supported
func (c *Components) TickerBatchingSupported() bool {
	return c != nil && c.TickerBatching != nil
}

// SubscribeEnabled checks if subscription functionality is enabled
func (c *Components) SubscribeEnabled() bool {
	return c.Subscribe != nil && *c.Subscribe
}

// UnsubscribeEnabled checks if unsubscribe functionality is enabled
func (c *Components) UnsubscribeEnabled() bool {
	return c.Subscribe != nil && *c.Subscribe
}

// AutoPairUpdatesEnabled checks if auto pair updating functionality is enabled
func (c *Components) AutoPairUpdatesEnabled() bool {
	return c.Subscribe != nil && *c.Subscribe
}

// Update takes in a secondary functionality list to ensure default full list is
// primed
func (c *Components) Update(p *Components) error {
	if p == nil {
		return errors.New("fucked")
	}
	if p.TickerBatching != nil {
		if c.TickerBatching == nil {
			return errors.New("default TickerBatching support unset")
		}
		c.TickerBatching = p.TickerBatching
	}
	if p.AutoPairUpdates != nil {
		if c.AutoPairUpdates == nil {
			return errors.New("default AutoPairUpdates support unset")
		}
		c.AutoPairUpdates = p.AutoPairUpdates
	}
	if p.AccountBalance != nil {
		if c.AccountBalance == nil {
			return errors.New("default AccountBalance support unset")
		}
		c.AccountBalance = p.AccountBalance
	}
	if p.CryptoDeposit != nil {
		if c.CryptoDeposit == nil {
			return errors.New("default CryptoDeposit support unset")
		}
		c.CryptoDeposit = p.CryptoDeposit
	}
	if p.CryptoWithdrawal != nil {
		if c.CryptoWithdrawal == nil {
			return errors.New("default CryptoWithdrawal support unset")
		}
		c.CryptoWithdrawal = p.CryptoWithdrawal
	}
	if p.FiatWithdraw != nil {
		if c.FiatWithdraw == nil {
			return errors.New("default FiatWithdraw support unset")
		}
		c.FiatWithdraw = p.FiatWithdraw
	}
	if p.GetOrder != nil {
		if c.GetOrder == nil {
			return errors.New("default GetOrder support unset")
		}
		c.GetOrder = p.GetOrder
	}
	if p.GetOrders != nil {
		if c.GetOrders == nil {
			return errors.New("default GetOrders support unset")
		}
		c.GetOrders = p.GetOrders
	}
	if p.CancelOrders != nil {
		if c.CancelOrders == nil {
			return errors.New("default CancelOrders support unset")
		}
		c.CancelOrders = p.CancelOrders
	}
	if p.CancelOrder != nil {
		if c.CancelOrder == nil {
			return errors.New("default CancelOrder support unset")
		}
		c.CancelOrder = p.CancelOrder
	}
	if p.SubmitOrder != nil {
		if c.SubmitOrder == nil {
			return errors.New("default SubmitOrder support unset")
		}
		c.SubmitOrder = p.SubmitOrder
	}
	if p.SubmitOrders != nil {
		if c.SubmitOrders == nil {
			return errors.New("default SubmitOrders support unset")
		}
		c.SubmitOrders = p.SubmitOrders
	}
	if p.ModifyOrder != nil {
		if c.ModifyOrder == nil {
			return errors.New("default ModifyOrder support unset")
		}
		c.ModifyOrder = p.ModifyOrder
	}
	if p.DepositHistory != nil {
		if c.DepositHistory == nil {
			return errors.New("default DepositHistory support unset")
		}
		c.DepositHistory = p.DepositHistory
	}
	if p.WithdrawalHistory != nil {
		if c.WithdrawalHistory == nil {
			return errors.New("default WithdrawalHistory support unset")
		}
		c.WithdrawalHistory = p.WithdrawalHistory
	}
	if p.TradeFetching != nil {
		if c.TradeFetching == nil {
			return errors.New("default TradeFetching support unset")
		}
		c.TradeFetching = p.TradeFetching
	}
	if p.ExchangeTradeHistory != nil {
		if c.ExchangeTradeHistory == nil {
			return errors.New("default ExchangeTradeHistory support unset")
		}
		c.ExchangeTradeHistory = p.ExchangeTradeHistory
	}
	if p.UserTradeHistory != nil {
		if c.UserTradeHistory == nil {
			return errors.New("default UserTradeHistory support unset")
		}
		c.UserTradeHistory = p.UserTradeHistory
	}
	if p.TradeFee != nil {
		if c.TradeFee == nil {
			return errors.New("default TradeFee support unset")
		}
		c.TradeFee = p.TradeFee
	}
	if p.FiatDepositFee != nil {
		if c.FiatDepositFee == nil {
			return errors.New("default FiatDepositFee support unset")
		}
		c.FiatDepositFee = p.FiatDepositFee
	}
	if p.FiatWithdrawalFee != nil {
		if c.FiatWithdrawalFee == nil {
			return errors.New("default FiatWithdrawalFee support unset")
		}
		c.FiatWithdrawalFee = p.FiatWithdrawalFee
	}
	if p.CryptoDepositFee != nil {
		if c.CryptoDepositFee == nil {
			return errors.New("default CryptoDepositFee support unset")
		}
		c.CryptoDepositFee = p.CryptoDepositFee
	}
	if p.CryptoWithdrawalFee != nil {
		if c.CryptoWithdrawalFee == nil {
			return errors.New("default CryptoWithdrawalFee support unset")
		}
		c.CryptoWithdrawalFee = p.CryptoWithdrawalFee
	}
	if p.TickerFetching != nil {
		if c.TickerFetching == nil {
			return errors.New("default TickerFetching support unset")
		}
		c.TickerFetching = p.TickerFetching
	}
	if p.KlineFetching != nil {
		if c.KlineFetching == nil {
			return errors.New("default KlineFetching support unset")
		}
		c.KlineFetching = p.KlineFetching
	}
	if p.OrderbookFetching != nil {
		if c.OrderbookFetching == nil {
			return errors.New("default OrderbookFetching support unset")
		}
		c.OrderbookFetching = p.OrderbookFetching
	}
	if p.AccountInfo != nil {
		if c.AccountInfo == nil {
			return errors.New("default AccountInfo support unset")
		}
		c.AccountInfo = p.AccountInfo
	}
	if p.FiatDeposit != nil {
		if c.FiatDeposit == nil {
			return errors.New("default FiatDeposit support unset")
		}
		c.FiatDeposit = p.FiatDeposit
	}
	if p.DeadMansSwitch != nil {
		if c.DeadMansSwitch == nil {
			return errors.New("default DeadMansSwitch support unset")
		}
		c.DeadMansSwitch = p.DeadMansSwitch
	}
	if p.Subscribe != nil {
		if c.Subscribe == nil {
			return errors.New("default Subscribe support unset")
		}
		c.Subscribe = p.Subscribe
	}
	if p.Unsubscribe != nil {
		if c.Unsubscribe == nil {
			return errors.New("default Unsubscribe support unset")
		}
		c.Unsubscribe = p.Unsubscribe
	}
	if p.AuthenticatedEndpoints != nil {
		if c.AuthenticatedEndpoints == nil {
			return errors.New("default AuthenticatedEndpoints support unset")
		}
		c.AuthenticatedEndpoints = p.AuthenticatedEndpoints
	}
	if p.MessageCorrelation != nil {
		if c.MessageCorrelation == nil {
			return errors.New("default MessageCorrelation support unset")
		}
		c.MessageCorrelation = p.MessageCorrelation
	}
	if p.MessageSequenceNumbers != nil {
		if c.MessageSequenceNumbers == nil {
			return errors.New("default MessageSequenceNumbers support unset")
		}
		c.MessageSequenceNumbers = p.MessageSequenceNumbers
	}
	if p.Withdraw != nil {
		if c.Withdraw == nil {
			return errors.New("default Withdraw support unset")
		}
		c.Withdraw = p.Withdraw
	}

	return nil
}

// TradeHistoryCaveat defines a set of exchange params that will allow for a sync item
// to be generated to populate via rest the current trading tip and also
// populate the full historic trade information for a currency asset
type TradeHistoryCaveat struct {
	HistoricFetching bool
	HistoricalOffset time.Duration
	StartTime        time.Time
}
