package protocol

import "golang.org/x/time/rate"

// DefaultGlobalRate this defaults the rate to once every second
var DefaultGlobalRate = &GlobalRate{
	Auth:   rate.NewLimiter(1, 1),
	UnAuth: rate.NewLimiter(1, 1),
}

// Limiter interface to determine if we are a specific rate for a function or we have
// global auth and unauth values
type Limiter interface {
	IsGlobal() bool
	Execute(auth bool)
	Reserve(n int, auth bool) error
}

// GlobalRate is global rate limit variables
type GlobalRate struct {
	UnAuth *rate.Limiter
	Auth   *rate.Limiter
}

// SpecificRate defines a specific rate limiter for a designated function
type SpecificRate struct {
	Rate *rate.Limiter
}
