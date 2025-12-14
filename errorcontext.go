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

func (e *BaseError[T]) SetContextFields(f T) {
	e.contextFields = f
}

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
