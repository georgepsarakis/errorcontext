package errorcontext

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
)

type BaseError[T any] struct {
	originalErr   error
	contextFields T
	isPanic       bool
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

func (e *BaseError[T]) MarkAsPanic() *BaseError[T] {
	e.isPanic = true
	return e
}

func (e *BaseError[T]) IsPanic() bool {
	if e == nil {
		return false
	}
	return e.isPanic
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

const FieldNamePanicStackTrace = "stack"
const FieldNamePanicMessage = "panic"

type Panic struct {
	Message string
	Stack   []string
}

type ErrorGenerator[T error] func(p Panic) T

func DefaultErrorGenerator(p Panic) error {
	return fmt.Errorf("%s\n%s", p.Message, strings.Join(p.Stack, "\n"))
}

var _ ErrorGenerator[error] = DefaultErrorGenerator

type Recoverer[T error] struct {
	// newErrorFunc converts a Panic value to a type with the `error` interface.
	// This is a required parameter.
	newErrorFunc ErrorGenerator[T]
	// PanicValueTransform if set will try to format arbitrary panic value types,
	// such as a struct or a map.
	PanicValueTransform func(r any) (string, error)
	// SkippedStackTraceLines sets the number of stack trace lines to be skipped.
	// In-library stack trace lines may be considered irrelevant or noise and
	// thus can be optionally skipped. By default, no lines are skipped.
	SkippedStackTraceLines uint
}

func NewRecoverer[T error](newError ErrorGenerator[T]) Recoverer[T] {
	if newError == nil {
		panic(ErrNewErrorFuncNotSet)
	}
	return Recoverer[T]{
		newErrorFunc: newError,
	}
}

var ErrNewErrorFuncNotSet = errors.New("error generator function is not set")

// Wrap allows recovery from panics for the given function.
// Panics are translated and propagated as errors that can be handled accordingly.
// Note: unrecovered panics can cause an abnormal program exit.
func (r Recoverer[T]) Wrap(fn func() error) (err error) {
	if r.newErrorFunc == nil {
		return fn()
	}
	defer func() {
		if rv := recover(); rv != nil {
			err = r.newErrorFunc(r.Format(rv))
		}
	}()
	err = fn()
	return err
}

// WrapFunc is a convenience wrapper that returns a decorated function,
// ensuring that panics are converted to error values.
//
// A common use case is to pass the function directly to errgroup.Submit:
//
//	grp := errgroup.Group{}
//	recoverer := errorcontext.NewRecoverer[error](errorcontext.DefaultErrorGenerator)
//	grp.Go(recoverer.WrapFunc(func() error {
//		panic("something bad happened")
//	}))
func (r Recoverer[T]) WrapFunc(fn func() error) func() error {
	return func() error {
		return r.Wrap(fn)
	}
}

// Format transforms an arbitrary value thrown by panic to an error message
// along with providing the current goroutine stack trace for the panic root cause.
// If PanicValueTransform is non-nil, an attempt to format the recovered value is performed.
// If the formatter function returns an error, a fallback approach is used and the failure
// error message is appended to the standard message template.
// Note: this method is intended to be public in order to facilitate testing.
func (r Recoverer[T]) Format(rv any) Panic {
	var baseMessage string
	switch v := rv.(type) {
	case error, string:
		baseMessage = fmt.Sprintf("%s: %s", FieldNamePanicMessage, v)
	default:
		if r.PanicValueTransform != nil {
			formatted, err := r.PanicValueTransform(rv)
			if err != nil {
				baseMessage = fmt.Sprintf("%s: %v\nfailed to transform: %s", FieldNamePanicMessage, v, err.Error())
			} else {
				baseMessage = fmt.Sprintf("%s: %s", FieldNamePanicMessage, formatted)
			}
		}
		if baseMessage == "" {
			baseMessage = fmt.Sprintf("%s: %v", FieldNamePanicMessage, v)
		}
	}
	debugStack := debug.Stack()
	stackLines := make([]string, 0, bytes.Count(debugStack, []byte{'\n'}))
	scanner := bufio.NewScanner(bytes.NewReader(debugStack))
	var lineNumber uint
	for scanner.Scan() {
		lineNumber++
		if lineNumber <= r.SkippedStackTraceLines {
			continue
		}
		stackLines = append(stackLines, scanner.Text())
	}
	return Panic{
		Message: baseMessage,
		Stack:   stackLines,
	}
}
