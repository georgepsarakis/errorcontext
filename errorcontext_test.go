package errorcontext

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testError struct {
	*BaseError[[]string]
}

func TestBaseError_IsZero(t *testing.T) {
	te := testError{}
	assert.True(t, te.IsZero())
}

func TestCollect(t *testing.T) {
	err := &testError{
		BaseError: &BaseError[[]string]{
			originalErr:   errors.New("original error"),
			contextFields: []string{"attr1", "attr2"},
		},
	}
	err2 := fmt.Errorf("something bad happened: %w", err)
	err3 := &testError{
		BaseError: &BaseError[[]string]{
			originalErr:   err2,
			contextFields: []string{"attr3"},
		},
	}
	var attrs []string
	for _, e := range Collect[*testError](err3) {
		attrs = append(attrs, e.ContextFields()...)
	}
	assert.Equal(t, []string{"attr3", "attr1", "attr2"}, attrs)
}
