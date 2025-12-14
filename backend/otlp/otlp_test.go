package otlp

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestError_Context(t *testing.T) {
	err := NewError(errors.New("database error"),
		attribute.Bool("attr1", true),
		attribute.String("attr2", "test"))

	assert.Equal(t,
		[]attribute.KeyValue{
			attribute.Bool("attr1", true),
			attribute.String("attr2", "test"),
		},
		AsContext(err))
}
