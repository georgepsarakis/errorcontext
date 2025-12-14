package zap

import (
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
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
		ChainContext(zf2))
}
