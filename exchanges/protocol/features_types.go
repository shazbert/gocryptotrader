package protocol

import "time"

var (
// // On infers functionality support and enabled
// On = func(s *GlobalRate) *Component { b := true; return &b }
// // Off infers functionality support and disabled
// Off = func() *bool { b := false; return &b }
)

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
