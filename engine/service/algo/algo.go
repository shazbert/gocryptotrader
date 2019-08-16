package algo

import (
	uuid "github.com/satori/go.uuid"
)

// Base critical variables for algorithm production
type Base struct {
	Version string
	Name    string
	ID      uuid.UUID
}

// Trader defines
type Trader interface{}
