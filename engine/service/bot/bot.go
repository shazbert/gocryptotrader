package bot

import (
	"errors"
	"fmt"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/asset"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	log "github.com/thrasher-/gocryptotrader/logger"
)

var _ = ticker.Ticker{}
var _ = orderbook.Base{}

// New starts a new swarm instance
func New() Swarm {
	fmt.Println("THE SWARM LIVES MWAHAHAHAHA")

	return Swarm{}
}

// Swarm defines the full instances of preconfigured bots
type Swarm []Instance

// Shutdown shuts down all instances
func (s *Swarm) Shutdown() []error {
	if s == nil {
		return []error{errors.New("s is nil mate")}
	}
	return nil
}

// Kill shuts down a singular bot instance
func (s Swarm) Kill(p uuid.UUID) error {
	if s == nil {
		return errors.New("s is nil mate!!! lolol")
	}
	for i := range s {
		err := s[i].Shutdown()
		if err != nil {
			return err
		}
		// pop instance from the slice here

	}
	return nil
}

// StartInstance registers a new bot instance into the engine swarm
func (s *Swarm) StartInstance(c Config, framework Trader) error {
	newInstance := Instance{}

	fmt.Println("MEOW CATS!!!!!!!!!!")

	newInstance.FullScreen = c.ExchangeCurrencies

	if framework == nil {
		newdummy := &Dummy{}
		newInstance.Logic = newdummy
	} else {
		newInstance.Logic = framework
	}

	go newInstance.Monitor()

	*s = append(*s, newInstance)
	return nil
}

// Instance defines a pre configured automatic trading bot
type Instance struct {
	Profit       float64
	Currencies   []currency.Code
	Exchanges    []string
	FullScreen   map[string]currency.Pairs
	Logic        Trader
	EventManager *EventManager
	shutdown     chan struct{}
}

// EventManager yay
type EventManager struct{}

// SendEvent sends event bro
func (e *EventManager) SendEvent(d *Decision) {
	fmt.Println("Event sent")
}

// Shutdown shuts down all routines and breaks all links to data
func (i *Instance) Shutdown() error {
	i = nil
	return nil
}

// Monitor starts a monitoring service for the instance acting on changes in
// levels in trading data ie ticker, orderbook data
func (i *Instance) Monitor() {
	fmt.Println("BOT HAS START WOOT!")
	tick := time.NewTicker(time.Second)

	for {
		select {
		case <-tick.C:
			fmt.Println("BOT MEOW")
			for exch := range i.FullScreen {
				for a := range i.FullScreen[exch] {
					price, err := ticker.GetTicker(exch, i.FullScreen[exch][a], asset.Spot)
					if err != nil {
						log.Error(log.Global, err)
					} else {
						d, err := i.Logic.CheckTicker(price)
						if err != nil {
							log.Error(log.Global, err)
						} else {
							i.EventManager.SendEvent(d)
						}
					}

					fmt.Println("BOT:", price)

					ob, err := orderbook.Get(exch, i.FullScreen[exch][a], asset.Spot)
					if err != nil {
						log.Error(log.Global, err)
					} else {
						d, err := i.Logic.CheckOrderbook(ob)
						if err != nil {
							log.Error(log.Global, err)
						} else {
							i.EventManager.SendEvent(d)
						}
					}

					fmt.Println("BOT ORDERBOOK")
				}
			}
			fmt.Println("BOT DONE")
		case <-i.shutdown:
			return
		}
	}
}

// Dummy implements the trader interface
type Dummy struct{}

func (d *Dummy) Setup(cfg interface{}) error                        { return nil }
func (d *Dummy) Start() error                                       { return nil }
func (d *Dummy) CheckTicker(t ticker.Price) (*Decision, error)      { return &Decision{}, nil }
func (d *Dummy) CheckOrderbook(t orderbook.Base) (*Decision, error) { return &Decision{}, nil }
func (d *Dummy) CheckRawTrade(tradeData string, interval int) (*Decision, error) {
	return &Decision{}, nil
}
