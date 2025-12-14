package errorcontext

import (
	"errors"
)

type BaseError[T any] struct {
	error
	originalErr   error
	contextFields T
}

func NewBaseError[T any](originalErr error, initialContext T) *BaseError[T] {
	return &BaseError[T]{
		originalErr:   originalErr,
		contextFields: initialContext,
	}
}

func (e *BaseError[T]) Error() string {
	return e.originalErr.Error()
}

// Unwrap allows the original error to be resolved by errors.Is & errors.As.
func (e *BaseError[T]) Unwrap() error {
	return e.originalErr
}

func (e *BaseError[T]) IsZero() bool {
	if e == nil {
		return true
	}
	return e.originalErr == nil
}

func (e *BaseError[T]) ContextFields() T {
	return e.contextFields
}

// SetContextFields is a setter that replaces the attached error context.
func (e *BaseError[T]) SetContextFields(f T) {
	e.contextFields = f
}

// Collect finds aggregates all errors that match the given target type,
// within the error chain of err. The resulting slice contains target error instances
// in reverse order.
func Collect[T error](err error) []T {
	if err == nil {
		return nil
	}
	var found []T
	currentErr := err
	for currentErr != nil {
		var target T
		if errors.As(currentErr, &target) {
			var last T
			if len(found) > 0 {
				last = found[len(found)-1]
			}
			if !errors.Is(target, last) {
				found = append(found, target)
			}
		}
		currentErr = errors.Unwrap(currentErr)
	}
	return found
}
