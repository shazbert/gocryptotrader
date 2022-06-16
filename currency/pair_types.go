package currency

import "errors"

// ErrPairIsEmpty defines an error when a pair is empty
var ErrPairIsEmpty = errors.New("pair is empty")

// Pair holds currency pair information
type Pair struct {
	Delimiter string `json:"delimiter,omitempty"`
	Base      Code   `json:"base,omitempty"`
	Quote     Code   `json:"quote,omitempty"`
}

// Pairs defines a list of pairs
type Pairs []Pair
