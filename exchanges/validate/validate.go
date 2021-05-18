package validate

import (
	"errors"
	"fmt"
)

// Checker defines validation check functionality
type Checker interface {
	Check() error
}

// Check defines a validation check function to close over individual validation
// check methods
type Check func() error

// Check initiates the Check functionality
func (v Check) Check() error {
	return v()
}

var ErrIsNil = errors.New("instance is nil cannot validate")

type Validator struct{}

func (v *Validator) Validate(exchName string, opt ...Checker) error {
	if v == nil {
		return ErrIsNil
	}
	for x := range opt {
		err := opt[x].Check()
		if err != nil {
			return fmt.Errorf("%s validation error: %w", exchName, err)
		}
	}
	return nil
}
