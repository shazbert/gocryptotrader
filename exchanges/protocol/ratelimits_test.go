package protocol

import (
	"fmt"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

type GlobalRate struct {
	UnAuth *rate.Limiter
	Auth   *rate.Limiter
}

// Limiter interface and stuff
type Limiter interface {
	Execute()
}

type SpecificUnAuthRate struct {
	Global       *GlobalRate
	SpecificRate *rate.Limiter
}

func (s *SpecificUnAuthRate) Execute() {
	if s.Global != nil {
		if s.Global.UnAuth.Allow() {
			fmt.Println("youre allowed mate")
			return
		}

		spot := s.Global.UnAuth.Reserve()
		time.Sleep(spot.Delay())
		if !spot.OK() {
			panic("EEEEK")
		}
		return
	}

	if s.SpecificRate.Allow() {
		fmt.Println("youre totally allowed mate")
		return
	}

	spot := s.SpecificRate.Reserve()
	time.Sleep(spot.Delay())
	if !spot.OK() {
		panic("EEEEK")
	}
	return
}

type SpecificAuthRate struct {
	Global       *GlobalRate
	SpecificRate *rate.Limiter
}

func (s *SpecificAuthRate) Execute() {
	if s.Global != nil {
		if s.Global.Auth.Allow() {
			fmt.Println("youre allowed mate")
			return
		}

		spot := s.Global.Auth.Reserve()
		time.Sleep(spot.Delay())
		if !spot.OK() {
			panic("EEEEK")
		}
		return
	}

	if s.SpecificRate.Allow() {
		fmt.Println("youre totally allowed mate")
		return
	}

	spot := s.SpecificRate.Reserve()
	time.Sleep(spot.Delay())
	if !spot.OK() {
		panic("EEEEK")
	}
	return
}

func TestLimit(t *testing.T) {
	duration := rate.Every(time.Second)
	r := rate.NewLimiter(duration, 3)
	burstSize := r.Burst()
	fmt.Println("BURST SIZE:", burstSize)
	reservationBurst := r.ReserveN(time.Now(), burstSize+3)
	if reservationBurst == nil {
		fmt.Println("YAY MEOW")
	}
	t.Fatalf("%d", reservationBurst.Delay())
	time.Sleep(reservationBurst.Delay())
	if !reservationBurst.OK() {
		t.Error("this should not occur")
	}
	if r.Allow() {
		t.Error("Should not be able to")
	}
	if r.Allow() {
		t.Error("Should not be able to")
	}
	if r.Allow() {
		t.Error("Should not be able to")
	}
	if !r.Allow() {
		spot := r.Reserve()
		delay := spot.Delay()
		fmt.Println("Rate limit works with delay:", delay.String())
		time.Sleep(delay)
		if !spot.OK() {
			t.Error("OH NOES")
		}
	}
}
