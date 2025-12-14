package zerolog

import (
	"errors"

	"github.com/rs/zerolog"

	"github.com/georgepsarakis/errorcontext"
)

type Error struct {
	*errorcontext.BaseError[*zerolog.Event]
}

func NewError(err error, dict *zerolog.Event) *Error {
	if dict == nil {
		dict = zerolog.Dict()
	}
	dict = dict.Stack().Err(err)
	b := errorcontext.NewBaseError[*zerolog.Event](
		err,
		dict,
	)
	return &Error{
		BaseError: b,
	}
}

func (e *Error) Context() *zerolog.Event {
	if e == nil {
		return zerolog.Dict()
	}
	return e.BaseError.ContextFields()
}

func (e *Error) AddContextFields(f map[string]any) {
	e.BaseError.SetContextFields(e.BaseError.ContextFields().Fields(f))
}

func AsContext(err error) *zerolog.Event {
	if err == nil {
		return nil
	}
	var z *Error
	if errors.As(err, &z) {
		return z.ContextFields()
	}
	return nil
}

type ErrorEventPair struct {
	Err     error
	Context *zerolog.Event
}

func AsChainContext(err error) []ErrorEventPair {
	if err == nil {
		return nil
	}
	var z []ErrorEventPair
	for _, e := range errorcontext.Collect[*Error](err) {
		z = append(z,
			ErrorEventPair{
				Err:     e,
				Context: e.Context(),
			})
	}
	return z
}
