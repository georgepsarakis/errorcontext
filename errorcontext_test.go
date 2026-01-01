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
	t.Parallel()

	te := testError{}
	assert.True(t, te.IsZero())

	var err BaseError[error]
	assert.True(t, err.IsZero())
}

func TestCollect(t *testing.T) {
	t.Parallel()

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
	assert.Nil(t, Collect[error](nil))
}

func TestRecoverer_Wrap(t *testing.T) {
	t.Parallel()

	type args struct {
		fn func() error
	}
	type testCase[T error] struct {
		name           string
		args           args
		recoverer      Recoverer[error]
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
			recoverer: func() Recoverer[error] {
				r := NewRecoverer(newErrorFunc)
				r.PanicValueTransform = nil
				return r
			}(),
			wantErrMessage: "panic: runtime error: invalid memory address or nil pointer",
			wantStackTrace: "errorcontext/errorcontext_test.go",
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
			},
			recoverer: func() Recoverer[error] {
				r := NewRecoverer(newErrorFunc)
				r.PanicValueTransform = func(r any) (string, error) {
					switch v := r.(type) {
					case map[string]string:
						b, err := json.Marshal(v)
						return string(b), err
					}
					return "", nil
				}
				return r
			}(),
			wantErrMessage: `panic: {"field1":"undefined behavior","field2":"panic with map"}`,
			wantStackTrace: "errorcontext/errorcontext_test.go",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.recoverer

			var err error
			require.NotPanics(t, func() {
				err = r.Wrap(tt.args.fn)
			})
			assert.ErrorContains(t, err, tt.wantErrMessage)
			assert.ErrorContains(t, err, tt.wantStackTrace)
		})
	}
}

func TestRecoverer_Format(t *testing.T) {
	t.Parallel()

	type args struct {
		rv any
	}
	type testCase[T error] struct {
		name string
		r    Recoverer[T]
		args args
		want Panic
	}
	tests := []testCase[error]{
		{
			name: "should format arbitrary panic value types with the custom formatter",
			r: func() Recoverer[error] {
				r := NewRecoverer(DefaultErrorGenerator)
				r.PanicValueTransform = func(r any) (string, error) {
					return fmt.Sprintf("formatted: %v", r), nil
				}
				return r
			}(),
			args: args{
				rv: []string{"test1", "test2"},
			},
			want: Panic{
				Message: "panic: formatted: [test1 test2]",
			},
		},
		{
			name: "should fall back to format arbitrary panic value types",
			r:    NewRecoverer(DefaultErrorGenerator),
			args: args{
				rv: []string{"test1", "test2"},
			},
			want: Panic{
				Message: "panic: [test1 test2]",
			},
		},
		{
			name: "on error should fall back to generic formatting panic values",
			r: func() Recoverer[error] {
				r := NewRecoverer(DefaultErrorGenerator)
				r.PanicValueTransform = func(r any) (string, error) {
					return "", errors.New("formatter failed")
				}
				return r
			}(),
			args: args{
				rv: []string{"test1", "test2"},
			},
			want: Panic{
				Message: "panic: [test1 test2]\nfailed to transform: formatter failed",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.r.Format(tt.args.rv)
			assert.Equal(t, tt.want.Message, f.Message)
			assert.NotEmpty(t, f.Stack)
		})
	}
}

func TestRecoverer_WrapFunc(t *testing.T) {
	t.Parallel()

	type args struct {
		fn func() error
	}
	type testCase[T error] struct {
		name      string
		r         Recoverer[T]
		args      args
		want      assert.ErrorAssertionFunc
		wantPanic assert.PanicAssertionFunc
	}
	tests := []testCase[error]{
		{
			name: "should recover any panic",
			r:    NewRecoverer(DefaultErrorGenerator),
			args: args{
				fn: func() error {
					panic("something bad happened")
					return nil //nolint:govet
				},
			},
			want: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantPanic != nil {
				tt.wantPanic(t, func() { _ = tt.r.WrapFunc(tt.args.fn)() })
			} else {
				tt.want(t, tt.r.WrapFunc(tt.args.fn)())
			}
		})
	}
}

func TestDefaultErrorGenerator(t *testing.T) {
	t.Parallel()

	type args struct {
		p Panic
	}
	tests := []struct {
		name           string
		args           args
		wantErr        assert.ErrorAssertionFunc
		wantErrMessage string
	}{
		{
			name: "converts a Panic instance to error",
			args: args{
				p: Panic{
					Message: "panic: something bad happened",
					Stack:   []string{"line1", "line2"},
				},
			},
			wantErr:        assert.Error,
			wantErrMessage: "panic: something bad happened\nline1\nline2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DefaultErrorGenerator(tt.args.p)
			tt.wantErr(t, err)
			if err != nil {
				assert.Equal(t, tt.wantErrMessage, err.Error())
			}
		})
	}
}

func TestBaseError_MarkAsPanic(t *testing.T) {
	t.Parallel()

	t.Run("should mark as panic", func(t *testing.T) {
		t.Parallel()

		err := BaseError[error]{}
		assert.False(t, err.IsPanic())
		assert.True(t, err.MarkAsPanic().IsPanic())
	})

	t.Run("should not fail on nil receiver", func(t *testing.T) {
		t.Parallel()

		var err BaseError[error]
		assert.False(t, err.IsPanic())
	})
}

func TestNewRecoverer(t *testing.T) {
	t.Parallel()

	assert.PanicsWithError(t, ErrNewErrorFuncNotSet.Error(), func() {
		NewRecoverer[error](nil)
	})
}
