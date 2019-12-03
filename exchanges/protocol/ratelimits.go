package protocol

import "golang.org/x/time/rate"

// var limits map[string]map[string]RateLimits

type Rate rate.Limit

// func HelloRate() {
// 	r := rate.NewLimiter(1, 3)
// 	r
// }
