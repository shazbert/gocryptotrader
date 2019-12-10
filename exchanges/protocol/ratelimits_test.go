package protocol

import (
	"fmt"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

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

type Priority int

var (
	High   = Priority(0)
	Medium = Priority(1)
	Low    = Priority(2)

	HighPriorityAuth   = func() (Priority, bool) { return High, true }
	MediumPriorityAuth = func() (Priority, bool) { return Medium, true }
	LowPriorityAuth    = func() (Priority, bool) { return Low, true }

	HighPriorityUnAuth   = func() (Priority, bool) { return High, false }   //4
	MediumPriorityUnAuth = func() (Priority, bool) { return Medium, false } //3
	LowPriorityUnAuth    = func() (Priority, bool) { return Low, false }    //2

	MinRefreshForLane = time.Second * 10
)

func TestThroughPut(t *testing.T) {
	// // Auth Rate - 50 requests / min
	// authrate := rate.NewLimiter(rate.Every(time.Minute), 50)

	// UnAuth Rate - 50 requests / min
	fmt.Println("time Minute:", time.Minute)
	rLimit := rate.Every(time.Minute)
	fmt.Println("rate limit", rLimit)
	unAuthrate := rate.NewLimiter(rLimit, 50)

	// derive functionality
	var features = []func() (Priority, bool){
		HighPriorityUnAuth,
		HighPriorityUnAuth,
		LowPriorityUnAuth,
		LowPriorityUnAuth,
	}

	tp, err := Hello(features, unAuthrate)
	if err != nil {
		t.Error("things are bad")
	}

	fmt.Println(tp)
}

// ThroughPut
type ThroughPut struct {
	Low    time.Duration
	Medium time.Duration
	High   time.Duration
	Window rate.Limit
}

// Hello world
func Hello(functions []func() (Priority, bool), r *rate.Limiter) (ThroughPut, error) {
	// bucket := r.Burst()
	// limit := r.Limit()

	// var high, medium, low int
	// for i := range functions {
	// 	p, _ := functions[i]()
	// 	switch p {
	// 	case High:
	// 		high++
	// 	case Medium:
	// 		medium++
	// 	case Low:
	// 		low++
	// 	}
	// }

	// if high+medium+low <= 0 {
	// 	return ThroughPut{}, errors.New("this is stuffed")
	// }

	// var LowItems time.Duration
	// if low > 0 {
	// 	LowItems = low * time.Second * 10
	// }

	// if high > 0 {

	// }
	return ThroughPut{
		High: time.Second * 12,
	}, nil
}
