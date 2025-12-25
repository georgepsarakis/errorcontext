package zap

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/georgepsarakis/errorcontext"
)

func TestError_Context(t *testing.T) {
	t.Parallel()

	core, observedLogs := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	err := errors.New("test error")
	zf := NewError(err, zap.String("tag1", "test1"))
	fields := append(zf.ContextFields(),
		zap.Int("attempt", 3),
		zap.Duration("backoff", time.Second))

	logger.Info("failed to fetch URL", fields...)

	logs := observedLogs.All()
	require.NotEmpty(t, logs)
	require.Len(t, logs, 1)
	r := logs[0]
	assert.Equal(t, []zap.Field{
		zap.String("tag1", "test1"),
		zap.Int("attempt", 3),
		zap.Duration("backoff", time.Second),
	}, r.Context)
	assert.Equal(t, "failed to fetch URL", r.Message)
}

func TestChainContext(t *testing.T) {
	t.Parallel()

	err := errors.New("test error")
	zf := NewError(err, zap.String("tag1", "test1"))
	zf2 := NewError(zf, zap.String("tag2", "test2"))

	assert.Equal(t,
		[]zap.Field{
			zap.String("tag2", "test2"),
			zap.String("tag1", "test1"),
		},
		AsChainContext(zf2))
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

	var pe *Error
	require.ErrorAs(t, err, &pe)
	assert.True(t, pe.IsPanic())

	fields := AsContext(err)
	require.NotEmpty(t, fields)
	require.Len(t, fields, 2)

	zfPanic := fields[0]

	assert.Equal(t, zfPanic.Key, "panic")
	assert.Equal(t, zfPanic.String, "panic: runtime error: invalid memory address or nil pointer dereference")
	zfStackTrace := fields[1]

	assert.Equal(t, zfStackTrace.Key, "stack")

	assert.NotEmpty(t, zfStackTrace.Interface)
	b, err := json.Marshal(zfStackTrace.Interface)
	require.NoError(t, err)

	var stackLines []string
	require.NoError(t, json.Unmarshal(b, &stackLines))
	assert.Contains(t,
		stackLines[2],
		"errorcontext/backend/zap/zap_test.go:73",
		fmt.Sprintf("%s", strings.Join(stackLines, "\n")))
}
