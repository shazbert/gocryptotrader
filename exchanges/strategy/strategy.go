package strategy

import (
	"context"
	"errors"
	"sync"

	"github.com/gofrs/uuid"
)

var errBaseNotFound = errors.New("strategy base not found")

// Requirement defines baseline functionality for strategy implementation to
// GCT.
type Requirement interface {
	GetBase() (*Base, error)
	GetContext() (context.Context, error)
	IsRunning() (bool, error)
	Stop() error
	Report() error
	Run() error
}

// StrategyManager defines management processes
type StrategyManager struct {
	Strategies []Requirement
	m          sync.Mutex
}

// Register
func (sm *StrategyManager) Register(obj Requirement) (uuid.UUID, error) {
	sm.m.Lock()
	defer sm.m.Unlock()
	return uuid.Nil, nil
}

// Run runs the applicable strategy
func (sm *StrategyManager) Run(id uuid.UUID) error {
	return nil
}

func (sm *StrategyManager) Stop(id uuid.UUID) error {
	return nil
}

// GetState returns a reportable history of actions; pnl, errors, etc.
func (sm *StrategyManager) IsRunning(id uuid.UUID) (chan interface{}, error) {
	return nil, nil
}

// Base defines the base strategy application for quick implementation and usage.
type Base struct {
	Context  context.Context
	Verbose  bool
	Shutdown chan struct{}
	WG       sync.WaitGroup
}

// GetBase returns the strategy base
func (b *Base) GetBase() (*Base, error) {
	if b == nil {
		return nil, errBaseNotFound
	}
	return b, nil
}

// Signals defines different signal options that a strategy relies on to execute
// functionality. (withdrawal, disaster recovery, )
type Signals interface {
	Wait() <-chan interface{}
	Input(interface{}) error
}

type Report struct {
}

type Reporter chan Report
