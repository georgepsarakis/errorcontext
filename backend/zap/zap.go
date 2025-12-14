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
	return e.BaseError.ContextFields()
}

func (e *Error) AddContextFields(f ...zap.Field) {
	e.BaseError.SetContextFields(append(e.BaseError.ContextFields(), f...))
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
