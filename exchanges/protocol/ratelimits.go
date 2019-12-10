package protocol

import (
	"errors"
	"time"

	"golang.org/x/time/rate"
)

// IsGlobal returns if this struct is a global value
func (g *GlobalRate) IsGlobal() bool {
	return true
}

// Execute is temporarily pauses execution in a routine to be inline with the
// rate limit system
func (g *GlobalRate) Execute(auth bool) {
	if auth {
		if g.Auth.Allow() {
			return
		}

		spot := g.Auth.Reserve()
		time.Sleep(spot.Delay())
		if !spot.OK() {
			panic("EEEEK")
		}
		return
	}

	if g.UnAuth.Allow() {
		return
	}

	spot := g.UnAuth.Reserve()
	time.Sleep(spot.Delay())
	if !spot.OK() {
		panic("EEEEK")
	}
	return
}

// Reserve allocates the amount of requests that will need to be sent and when
// rate limits are available will send it burst like; like a super saiyan.
func (g *GlobalRate) Reserve(n int, auth bool) error {
	if auth {
		if g.Auth.Burst() < n {
			return errors.New("reserve amount exceeded")
		}
		r := g.Auth.ReserveN(time.Now(), n)
		time.Sleep(r.Delay())
		return nil
	}
	if g.UnAuth.Burst() < n {
		return errors.New("reserve amount exceeded")
	}
	r := g.UnAuth.ReserveN(time.Now(), n)
	time.Sleep(r.Delay())
	return nil
}

func (g *GlobalRate) GetUnAuthLimit() rate.Limit {
	return g.UnAuth.Limit()
}

func (g *GlobalRate) GetAuthLimit() rate.Limit {
	return g.Auth.Limit()
}

func (g *GlobalRate) GetUnAuthBucket() int {
	return g.UnAuth.Burst()
}

func (g *GlobalRate) GetAuthBucket() int {
	return g.UnAuth.Burst()
}

// IsGlobal returns if this is a global variable
func (s *SpecificRate) IsGlobal() bool {
	return false
}

// Execute is temporarily pauses execution in a routine to be inline with the
// rate limit system
func (s *SpecificRate) Execute(_ bool) {
	if s.Rate.Allow() {
		return
	}

	spot := s.Rate.Reserve()
	time.Sleep(spot.Delay())
	if !spot.OK() {
		panic("EEEEK")
	}
	return
}

// Reserve allocates the amount of requests that will need to be sent and when
// rate limits are available will send it burst like; like a super saiyan.
func (s *SpecificRate) Reserve(n int, _ bool) error {
	if s.Rate.Burst() < n {
		return errors.New("reserve amount exceeded")
	}
	r := s.Rate.ReserveN(time.Now(), n)
	time.Sleep(r.Delay())
	return nil
}

func (s *SpecificRate) GetUnAuthLimit() rate.Limit {
	return s.Rate.Limit()
}

func (s *SpecificRate) GetAuthLimit() rate.Limit {
	return s.Rate.Limit()
}

func (s *SpecificRate) GetUnAuthBucket() int {
	return s.Rate.Burst()
}

func (s *SpecificRate) GetAuthBucket() int {
	return s.Rate.Burst()
}
