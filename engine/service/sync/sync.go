package sync

import (
	"fmt"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
)

// Syncer interface defines engine service synchronization functionality for
// things that require regular updating
type Syncer interface {
	Switch()
	GetState()
	Shutdown()
	Start()
}

// Manager defines a sync manager
type Manager struct {
	Syncers  []*Sync
	cont     chan struct{}
	divert   chan struct{}
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// GetManager returns a sync manager
func GetManager() *Manager {
	m := new(Manager)
	m.cont = make(chan struct{})
	m.divert = make(chan struct{})
	m.shutdown = make(chan struct{})
	go m.Moderate()
	return m
}

// Divert diverts traffic
func (m *Manager) Divert() {
	select {
	case m.divert <- struct{}{}:
	case <-m.divert:
	}
}

// Moderate moderates split stream diversion
func (m *Manager) Moderate() {
	fmt.Println("moderator")
	m.wg.Add(1)
	for {
		select {
		case <-m.divert:
			select {
			case m.divert <- struct{}{}:
			case <-m.shutdown:
				m.wg.Done()
				return
			}
		case <-m.shutdown:
			m.wg.Done()
			return
		default:
			select {
			case m.cont <- struct{}{}:
			case <-m.shutdown:
				m.wg.Done()
				return
			}
		}
	}
}

func (m *Manager) NewSyncGroup(service uuid.UUID, fn []func() error) error {
	var newwg sync.WaitGroup
	for i := range fn {
		err := m.NewSyncer(&newwg, fn[i])
		if err != nil {
			return err
		}
	}

	newwg.Wait()
	return nil
}

// NewSyncer makes a new syncer type
func (m *Manager) NewSyncer(wg *sync.WaitGroup, fn func() error) error {
	s := &Sync{
		cont:     m.cont,
		initial:  wg,
		wg:       &m.wg,
		shutdown: m.shutdown,
	}

	m.Syncers = append(m.Syncers, s)
	s.Synchronise(5*time.Second, fn)
	return nil
}

// Shutdown shuts everything down
func (m *Manager) Shutdown() error {
	close(m.shutdown)
	m.wg.Wait()
	m.Syncers = nil
	return nil
}

// Sync bla bla bla
type Sync struct {
	cont     <-chan struct{}
	initial  *sync.WaitGroup
	wg       *sync.WaitGroup
	group    *sync.WaitGroup
	shutdown chan struct{}
}

// Synchronise play and stuff
func (s *Sync) Synchronise(wait time.Duration, fn func() error) {
	s.initial.Add(1)
	var wg sync.WaitGroup
	var meow bool
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		s.wg.Add(1)
		t := time.NewTimer(0)
		wg.Done()
		for {
			select {
			case <-t.C:
				select {
				case <-s.cont:
				case <-s.shutdown:
					t.Stop()
					s.wg.Done()
					return
				}
				err := fn() // runs singular functionality
				if err != nil {
					fmt.Println(err)
				}
				if !meow {
					s.initial.Done()
					meow = true
				}
				t.Reset(wait)

			case <-s.shutdown:
				t.Stop()
				s.wg.Done()
				return
			}
		}
	}(&wg)
	wg.Wait()
}
