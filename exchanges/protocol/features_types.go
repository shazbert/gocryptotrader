package protocol

import "time"

var ()

// SupportedFeatures is a supported features list
var SupportedFeatures = []string{
	"TickerBatching",
	"AutoPairUpdates",
	"AccountBalance",
	"CryptoDeposit",
	"CryptoWithdrawal",
	"FiatWithdraw",
	"GetOrder",
	"GetOrders",
	"CancelOrders",
	"CancelOrder",
	"SubmitOrder",
	"SubmitOrders",
	"ModifyOrder",
	"DepositHistory",
	"WithdrawalHistory",
	"TradeFetching",
	"ExchangeTradeHistory",
	"UserTradeHistory",
	"TradeFee",
	"FiatDepositFee",
	"FiatWithdrawalFee",
	"CryptoDepositFee",
	"CryptoWithdrawalFee",
	"TickerFetching",
	"KlineFetching",
	"OrderbookFetching",
	"AccountInfo",
	"FiatDeposit",
	"DeadMansSwitch",
	"Subscribe",
	"Unsubscribe",
	"AuthenticatedEndpoints",
	"MessageCorrelation",
	"MessageSequenceNumbers",
	"Withdraw",
}

// Features stores the exchange supported protocol functionality
type Features struct {
	REST      *Components `json:"rest,omitempty"`
	Websocket *Components `json:"websocket,omitempty"`
	Fix       *Components `json:"fix,omitempty"`
}

// Permissions defines a set of allowable permissions
type Permissions uint32

// Component derives a singular potential supported function
type Component struct {
	Enabled bool
	Rate    Limiter `json:"-"`
	Auth    bool    `json:"-"`
}

// TradeHistoryCaveat defines a set of exchange params that will allow for a sync item
// to be generated to populate via rest the current trading tip and also
// populate the full historic trade information for a currency asset
type TradeHistoryCaveat struct {
	HistoricFetching bool
	HistoricalOffset time.Duration
	StartTime        time.Time
}
