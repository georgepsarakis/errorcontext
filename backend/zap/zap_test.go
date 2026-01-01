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

func TestError_ContextFields(t *testing.T) {
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

	assert.Nil(t, AsChainContext(nil))
}

func TestPanicHandler(t *testing.T) {
	var err error
	recoverer := errorcontext.NewRecoverer[*Error](FromPanic)
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
	require.Len(t, fields, 3)

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
		stackLines[10],
		"errorcontext/backend/zap/zap_test.go",
		strings.Join(stackLines, "\n"))
}

func TestAsContext(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want []zap.Field
	}{
		{
			name: "nil",
			args: args{
				err: nil,
			},
			want: nil,
		},
		{
			name: "wrapped errorcontext/zap.Error found",
			args: args{
				err: fmt.Errorf("wrapped error: %w",
					NewError(errors.New("test error"), zap.Bool("is_test", true))),
			},
			want: []zap.Field{zap.Bool("is_test", true)},
		},
		{
			name: "errorcontext/zap.Error not found",
			args: args{
				err: fmt.Errorf("wrapped error: %w", errors.New("test error")),
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tt.want, AsContext(tt.args.err), "AsContext(%v)", tt.args.err)
		})
	}
}
