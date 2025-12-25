package zap

import (
	"errors"

	"go.uber.org/zap"

	"github.com/georgepsarakis/errorcontext"
)

type Error struct {
	*errorcontext.BaseError[[]zap.Field]
}

func NewError(err error, context ...zap.Field) *Error {
	return &Error{
		BaseError: errorcontext.NewBaseError[[]zap.Field](err, context),
	}
}

func (e *Error) Context() []zap.Field {
	return e.ContextFields()
}

func (e *Error) AddContextFields(f ...zap.Field) {
	e.SetContextFields(append(e.ContextFields(), f...))
}

func (e *Error) MarkAsPanic() *Error {
	_ = e.BaseError.MarkAsPanic()
	return e
}

func AsContext(err error) []zap.Field {
	if err == nil {
		return nil
	}
	var z *Error
	if errors.As(err, &z) {
		return z.ContextFields()
	}
	return nil
}

func AsChainContext(err error) []zap.Field {
	if err == nil {
		return nil
	}
	var z []zap.Field
	for _, e := range errorcontext.Collect[*Error](err) {
		z = append(z, e.Context()...)
	}
	return z
}

func FromPanic(p errorcontext.Panic) *Error {
	return NewError(
		errors.New(p.Message),
		zap.String(errorcontext.FieldNamePanicMessage, p.Message),
		zap.Strings(errorcontext.FieldNamePanicStackTrace, p.Stack),
	).MarkAsPanic()
}
