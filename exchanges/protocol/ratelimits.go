package protocol

var limits map[string]map[string]RateLimits

// RateLimits defines exchange side rate limits for each individual protocol
type RateLimits struct {
	Authenticated   Limit
	Unauthenticated Limit
	count           int32
}

// Limit in duration
type Limit int64
