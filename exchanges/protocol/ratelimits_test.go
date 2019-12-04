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
