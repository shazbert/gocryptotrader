package request

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"golang.org/x/time/rate"
)

// Rate limiting errors.
var (
	ErrRateLimiterAlreadyDisabled = errors.New("rate limiter already disabled")
	ErrRateLimiterAlreadyEnabled  = errors.New("rate limiter already enabled")
	ErrDelayNotAllowed            = errors.New("delay not allowed")

	errInvalidWeight = errors.New("weight must be equal-or-greater than 1")
)

// RateLimitNotRequired is a no-op rate limiter.
var RateLimitNotRequired *RateLimiterWithWeight

// Const here define individual functionality sub types for rate limiting.
const (
	Unset EndpointLimit = iota
	Auth
	UnAuth
)

// EndpointLimit defines individual endpoint rate limits.
type EndpointLimit uint16

// Weight defines the number of reservations to be used. This is a generalised weight for rate limiting.
// e.g. n weight = n request. i.e. 50 Weight = 50 requests.
type Weight uint8

// RateLimitDefinitions is a map of endpoint limits to rate limiters.
type RateLimitDefinitions map[any]*RateLimiterWithWeight

// RateLimiterWithWeight is a rate limiter coupled with a weight which refers to the number or weighting of the request.
// This is used to define the rate limit for a specific endpoint.
type RateLimiterWithWeight struct {
	limiter *rate.Limiter
	weight  Weight
	m       sync.Mutex
}

// RateLimitWithWeightOverride requests a limiter with an optional request-specific weight.
type RateLimitWithWeightOverride struct {
	Limiter        *RateLimiterWithWeight
	WeightOverride Weight
}

type rateLimitReservation struct {
	limiter      *RateLimiterWithWeight
	reservations []*rate.Reservation
}

// NewRateLimit creates a new RateLimit based of time interval and how many actions allowed and breaks it down to an
// actions-per-second basis -- Burst rate is kept as one as this is not supported for out-bound requests.
func NewRateLimit(interval time.Duration, actions int) *rate.Limiter {
	if actions <= 0 || interval <= 0 {
		// Returns an un-restricted rate limiter
		return rate.NewLimiter(rate.Inf, 1)
	}

	i := 1 / interval.Seconds()
	rps := i * float64(actions)
	return rate.NewLimiter(rate.Limit(rps), 1)
}

// NewRateLimitWithWeight creates a new RateLimit based of time interval and how many actions allowed. This also has a
// weight count which refers to the number or weighting of the request. This is used to define the rate limit for a
// specific endpoint.
func NewRateLimitWithWeight(interval time.Duration, actions int, weight Weight) *RateLimiterWithWeight {
	return GetRateLimiterWithWeight(NewRateLimit(interval, actions), weight)
}

// NewWeightedRateLimitByDuration creates a new RateLimit based of time interval. This equates to 1 action per interval.
// The weight is set to 1.
func NewWeightedRateLimitByDuration(interval time.Duration) *RateLimiterWithWeight {
	return NewRateLimitWithWeight(interval, 1, 1)
}

// GetRateLimiterWithWeight couples a rate limiter with a weight count into an accepted defined rate limiter with weight
// struct.
func GetRateLimiterWithWeight(l *rate.Limiter, weight Weight) *RateLimiterWithWeight {
	return &RateLimiterWithWeight{limiter: l, weight: weight}
}

// NewBasicRateLimit returns an object that implements the limiter interface for basic rate limit.
func NewBasicRateLimit(interval time.Duration, actions int, weight Weight) RateLimitDefinitions {
	rl := NewRateLimitWithWeight(interval, actions, weight)
	return RateLimitDefinitions{Unset: rl, Auth: rl, UnAuth: rl}
}

// RateLimit throttles a request based on weight, delaying the request.
// Errors if no delay is permitted via the context and a delay is required.
func (r *RateLimiterWithWeight) RateLimit(ctx context.Context, endpointWeightOverride Weight, additionalRateLimits ...RateLimitWithWeightOverride) error {
	if err := common.NilGuard(r); err != nil {
		return err
	}

	tn := time.Now()
	reserved := make([]rateLimitReservation, 0, len(additionalRateLimits)+1)
	cancelReservations := func() {
		for i := len(reserved) - 1; i >= 0; i-- {
			reservation := reserved[i]
			reservation.limiter.m.Lock()
			for j := len(reservation.reservations) - 1; j >= 0; j-- {
				reservation.reservations[j].CancelAt(tn)
			}
			reservation.limiter.m.Unlock()
		}
	}

	rateLimits := append([]RateLimitWithWeightOverride{{Limiter: r, WeightOverride: endpointWeightOverride}}, additionalRateLimits...)
	var finalDelay time.Duration
	for _, rateLimit := range rateLimits {
		if err := common.NilGuard(rateLimit.Limiter); err != nil {
			cancelReservations()
			return err
		}
		weight := rateLimit.WeightOverride
		rateLimit.Limiter.m.Lock()
		if weight == 0 {
			weight = rateLimit.Limiter.weight
		}
		if weight == 0 {
			rateLimit.Limiter.m.Unlock()
			cancelReservations()
			return errInvalidWeight
		}
		reservations := make([]*rate.Reservation, 0, weight)
		for range weight {
			// Reserving one token at a time avoids requiring burst capacity.
			reservations = append(reservations, rateLimit.Limiter.limiter.ReserveN(tn, 1))
		}
		delay := reservations[len(reservations)-1].DelayFrom(tn)
		rateLimit.Limiter.m.Unlock()
		reserved = append(reserved, rateLimitReservation{limiter: rateLimit.Limiter, reservations: reservations})
		if finalDelay < delay {
			finalDelay = delay
		}
	}

	if hasDelayNotAllowed(ctx) {
		if finalDelay > 0 {
			cancelReservations()
			return ErrDelayNotAllowed
		}
		return nil
	}

	if dl, ok := ctx.Deadline(); ok && dl.Before(tn.Add(finalDelay)) {
		cancelReservations()
		return fmt.Errorf("rate limit delay of %s will exceed deadline: %w", finalDelay, context.DeadlineExceeded)
	}

	if finalDelay == 0 {
		return nil
	}

	select {
	case <-ctx.Done():
		cancelReservations()
		return ctx.Err()
	case <-time.After(finalDelay):
		return nil
	}
}
