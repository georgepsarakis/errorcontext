package zerolog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rs/zerolog"

	"github.com/georgepsarakis/errorcontext"
)

func init() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
}

func newLogger(t *testing.T) (zerolog.Logger, *bytes.Buffer) {
	t.Helper()
	now := time.Date(2025, time.January, 2, 11, 22, 33, 0, time.UTC)
	zerolog.TimestampFunc = func() time.Time { return now }

	output := bytes.NewBuffer(nil)
	return zerolog.New(output).With().Timestamp().Logger(), output
}

func TestError_Context(t *testing.T) {
	lg, output := newLogger(t)
	err := errors.New("something went really wrong")

	ze := NewError(
		err,
		zerolog.Dict().Str("a", "b"))

	lg.Error().Dict("context", ze.ContextFields()).Send()
	assert.JSONEq(t,
		`{
  "level": "error",
  "context": {
    "a": "b",
    "stack": [
      {
        "func": "TestError_Context",
        "line": "35",
        "source": "zerolog_test.go"
      },
      {
        "func": "tRunner",
        "line": "1934",
        "source": "testing.go"
      },
      {
        "func": "goexit",
        "line": "1268",
        "source": "asm_arm64.s"
      }
    ],
    "error": "something went really wrong"
  },
  "time": "2025-01-02T11:22:33Z"
}`, output.String())
}

func TestChainContext(t *testing.T) {
	lg, output := newLogger(t)

	baseErr := errors.New("something went really wrong")
	ze1 := NewError(baseErr, zerolog.Dict().Str("a", "b"))
	ze2 := NewError(ze1, zerolog.Dict().Str("c", "d"))

	ev := lg.Info()
	for i, c := range AsChainContext(ze2) {
		ev.Dict(fmt.Sprintf("err%d", i+1), c.Context)
	}
	ev.Send()
	assert.JSONEq(t, `
		{
		  "level": "info",
		  "err1": {
			"c": "d",
			"stack": [
			  {
				"func": "TestChainContext",
				"line": "73",
				"source": "zerolog_test.go"
			  },
			  {
				"func": "tRunner",
				"line": "1934",
				"source": "testing.go"
			  },
			  {
				"func": "goexit",
				"line": "1268",
				"source": "asm_arm64.s"
			  }
			],
			"error": "something went really wrong"
		  },
		  "err2": {
			"a": "b",
			"stack": [
			  {
				"func": "TestChainContext",
				"line": "73",
				"source": "zerolog_test.go"
			  },
			  {
				"func": "tRunner",
				"line": "1934",
				"source": "testing.go"
			  },
			  {
				"func": "goexit",
				"line": "1268",
				"source": "asm_arm64.s"
			  }
			],
			"error": "something went really wrong"
		  },
		  "time": "2025-01-02T11:22:33Z"
		}
	`, output.String())
}

func TestPanicHandler(t *testing.T) {
	var err error
	recoverer := errorcontext.Recoverer[*Error]{
		NewErrorFunc: func(p errorcontext.Panic) *Error {
			return FromPanic(p)
		},
	}
	require.NotPanics(t, func() {
		type temp struct {
			fieldA string
		}
		var tmp *temp
		err = recoverer.Wrap(func() error {
			tmp.fieldA = "panic"
			return nil
		})
	})

	var ze *Error
	require.ErrorAs(t, err, &ze)

	lg, output := newLogger(t)
	lg.Error().Dict("context", ze.ContextFields()).Send()

	var c map[string]any

	require.NoError(t, json.Unmarshal(output.Bytes(), &c))

	var ctx map[string]any

	require.IsType(t, ctx, c)

	ctx = c["context"].(map[string]any)

	require.IsType(t, ctx["stack"], []any{})

	stack := ctx["stack"].([]any)

	assert.IsType(t, stack[0], "")
	assert.Contains(t, stack[0].(string), "panic")
	assert.Contains(t, stack[2].(string), "errorcontext/backend/zerolog/zerolog_test.go:145")

	msg := ctx["panic"]

	assert.Equal(t, msg, "panic: runtime error: invalid memory address or nil pointer dereference")
}
