package stream

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// Websocket functionality list and state consts
const (
	// WebsocketNotEnabled alerts of a disabled websocket
	WebsocketNotEnabled                = "exchange_websocket_not_enabled"
	WebsocketNotAuthenticatedUsingRest = "%v - Websocket not authenticated, using REST\n"
	Ping                               = "ping"
	Pong                               = "pong"
	UnhandledMessage                   = " - Unhandled websocket message: "
)

// Websocket defines a return type for websocket connections via the interface
// wrapper for routine processing
type Websocket struct {
	canUseAuthenticatedEndpoints bool
	enabled                      bool
	Init                         bool
	connected                    bool
	connecting                   bool
	verbose                      bool
	connectionMonitorRunning     bool
	trafficMonitorRunning        bool
	dataMonitorRunning           bool
	trafficTimeout               time.Duration
	connectionMonitorDelay       time.Duration
	proxyAddr                    string
	defaultURL                   string
	defaultURLAuth               string
	runningURL                   string
	runningURLAuth               string
	exchangeName                 string
	m                            sync.Mutex
	connectionMutex              sync.RWMutex
	connector                    func() error

	subscriptionMutex sync.Mutex
	subscriptions     map[Connection]ChannelSubscription
	Subscribe         chan []ChannelSubscription
	Unsubscribe       chan []ChannelSubscription

	// Subscriber function for package defined websocket subscriber
	// functionality
	Subscriber func([]ChannelSubscription) error
	// Unsubscriber function for packaged defined websocket unsubscriber
	// functionality
	Unsubscriber func([]ChannelSubscription) error
	// GenerateSubs function for package defined websocket generate
	// subscriptions functionality
	GenerateSubs func() ([]ChannelSubscription, error)

	DataHandler chan interface{}
	ToRoutine   chan interface{}

	Match *Match

	// shutdown synchronises shutdown event across routines
	ShutdownC chan struct{}
	Wg        *sync.WaitGroup

	// Orderbook is a local buffer of orderbooks
	Orderbook buffer.Orderbook

	// Trade is a notifier of occurring trades
	Trade trade.Trade

	// Fills is a notifier of occurring fills
	Fills fill.Fills

	// trafficAlert monitors if there is a halt in traffic throughput
	TrafficAlert chan struct{}
	// ReadMessageErrors will received all errors from ws.ReadMessage() and
	// verify if its a disconnection
	ReadMessageErrors chan error
	features          *protocol.Features

	// Standard stream connection
	UnauthState ConnState
	Conn        ConnectionPool
	// Authenticated stream connection
	AuthState ConnState
	AuthConn  ConnectionPool

	// Latency reporter
	ExchangeLevelReporter Reporter

	// MaxSubScriptionsPerConnection defines the maximum number of
	// subscriptions per connection that is allowed by the exchange.
	MaxSubscriptionsPerConnection int
}

// ConnectionPool is a collection of websocket connections
type ConnectionPool []Connection

// Shutdown closes all websocket connections
func (c ConnectionPool) Shutdown() error {
	for i := range c {
		err := c[i].Shutdown()
		if err != nil {
			return err
		}
	}
	return nil
}

// SetURL sets the URL for all connections
func (c ConnectionPool) SetURL(url string) {
	for i := range c {
		c[i].SetURL(url)
	}
}

// SetProxy sets the proxy for all connections
func (c ConnectionPool) SetProxy(url string) {
	for i := range c {
		c[i].SetProxy(url)
	}
}

// SendPayloadUnsubscribe sends a payload to unsubscribe from a channel
// it will then release the subscription from the connection and return
func (w *Websocket) SendPayloadUnsubscribe(payload any, isAuth bool, channels ...ChannelSubscription) error {
	// Match the channels to the connection
	m := make(map[Connection][]ChannelSubscription)
	for i := range channels {
		for key, val := range w.subscriptions {
			if val.Equal(&channels[i]) {
				m[key] = append(m[key], channels[i])
				break
			}
		}
	}
	return nil
}

// SendPayloadSubscribe sends a payload to subscribe to a channel and adds the
// subscription to the connection.
func (w *Websocket) SendPayloadSubscribe(payload any, isAuth bool, channels ...ChannelSubscription) error {
	return nil
}

// WebsocketSetup defines variables for setting up a websocket connection
type WebsocketSetup struct {
	ExchangeConfig *config.Exchange
	DefaultURL     string
	RunningURL     string
	RunningURLAuth string
	// Connector is a function that connects individual websocket connections
	// to the exchange. This can be used for exchanges that have multiple
	// websocket connections.
	Connector             func(conn Connection, isAuth bool) error
	Subscriber            func(conn Connection, subs []ChannelSubscription) error
	Unsubscriber          func([]ChannelSubscription) error
	GenerateSubscriptions func() ([]ChannelSubscription, error)
	Features              *protocol.Features

	// Local orderbook buffer config values
	OrderbookBufferConfig buffer.Config

	TradeFeed bool

	// Fill data config values
	FillsFeed bool

	// MaxWebsocketSubscriptionsPerConnection defines the maximum number of
	// subscriptions per connection that is allowed by the exchange.
	MaxWebsocketSubscriptionsPerConnection int
}

// WebsocketConnection contains all the data needed to send a message to a WS
// connection
type WebsocketConnection struct {
	Verbose   bool
	connected int32

	// Gorilla websocket does not allow more than one goroutine to utilise
	// writes methods
	writeControl sync.Mutex

	RateLimit    int64
	ExchangeName string
	URL          string
	ProxyURL     string
	Wg           *sync.WaitGroup
	Connection   *websocket.Conn
	ShutdownC    chan struct{}

	Match             *Match
	ResponseMaxLimit  time.Duration
	Traffic           chan struct{}
	readMessageErrors chan error

	Reporter Reporter
}

// ConnState defines the connection state
type ConnState struct {
	URL              string
	ResponseMaxLimit time.Duration
	RateLimit        int64
	Reporter         Reporter
}
