package otlp

import (
	"errors"

	"go.opentelemetry.io/otel/attribute"

	"github.com/georgepsarakis/errorcontext"
)

type Error struct {
	*errorcontext.BaseError[[]attribute.KeyValue]
}

func NewError(err error, context ...attribute.KeyValue) *Error {
	return &Error{
		BaseError: errorcontext.NewBaseError[[]attribute.KeyValue](err, context),
	}
}

func (e *Error) Context() []attribute.KeyValue {
	if e == nil {
		return nil
	}
	return e.BaseError.ContextFields()[:]
}

func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	var o *Error
	if errors.As(err, &o) {
		return o
	}
	return nil
}
