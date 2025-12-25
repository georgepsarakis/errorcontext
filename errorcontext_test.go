package errorcontext

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestRecoverer_Wrap(t *testing.T) {
	type args struct {
		fn                    func() error
		panicValueTransformer func(r any) (string, error)
	}
	type testCase[T error] struct {
		name           string
		args           args
		wantErrMessage string
		wantStackTrace string
	}
	newErrorFunc := func(p Panic) error {
		return errors.New(p.Message + "\n" + strings.Join(p.Stack, "\n"))
	}
	tests := []testCase[error]{
		{
			name: "should recover any panic",
			args: args{
				fn: func() error {
					type tmp struct {
						field1 string
					}
					var v *tmp
					// Cause panic due to nil pointer dereference
					v.field1 = "something bad happened"
					return nil
				},
			},
			wantErrMessage: "panic: runtime error: invalid memory address or nil pointer",
			wantStackTrace: "errorcontext/errorcontext_test.go:68",
		},
		{
			name: "should format custom panic values",
			args: args{
				fn: func() error {
					panic(map[string]string{
						"field1": "undefined behavior",
						"field2": "panic with map",
					})
					return nil //nolint:govet
				},
				panicValueTransformer: func(r any) (string, error) {
					switch v := r.(type) {
					case map[string]string:
						b, err := json.Marshal(v)
						return string(b), err
					}
					return "", nil
				},
			},
			wantErrMessage: `panic: {"field1":"undefined behavior","field2":"panic with map"}`,
			wantStackTrace: "errorcontext/errorcontext_test.go:79",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRecoverer(newErrorFunc)
			r.PanicValueTransform = tt.args.panicValueTransformer

			var err error
			require.NotPanics(t, func() {
				err = r.Wrap(tt.args.fn)
			})
			assert.ErrorContains(t, err, tt.wantErrMessage)
			assert.ErrorContains(t, err, tt.wantStackTrace)
		})
	}
}
