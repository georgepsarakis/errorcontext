package errorcontext_test

import (
	"errors"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/georgepsarakis/errorcontext"
	zaperrorcontext "github.com/georgepsarakis/errorcontext/backend/zap"
)

var ErrProcessingFailure = errors.New("processing failure")

func ExampleAsContext() {
	cfg := zap.NewProductionConfig()
	zapLogger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = zapLogger.Sync()
	}()
	err = tryWithError()
	if errors.Is(err, ErrProcessingFailure) {
		zapLogger.Warn("something failed",
			zap.Dict("error_context", zaperrorcontext.AsContext(err)...),
			zap.Error(err))
	}
	//Output:
}

func tryWithError() error {
	return zaperrorcontext.NewError(
		ErrProcessingFailure,
		zap.String("path", "/a/b"),
		zap.Bool("enabled", true),
	)
}

func ExampleRecoverer() {
	cfg := zap.NewProductionConfig()
	zapLogger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	grp := errgroup.Group{}
	recoverer := errorcontext.Recoverer[*zaperrorcontext.Error]{
		NewErrorFunc: func(p errorcontext.Panic) *zaperrorcontext.Error {
			return zaperrorcontext.FromPanic(p)
		},
	}
	grp.Go(recoverer.WrapFunc(func() error {
		panic("something bad happened")
	}))

	grp.Go(func() error {
		fmt.Println("Hello World")
		return nil
	})
	if err := grp.Wait(); err != nil {
		zapLogger.Warn("something failed",
			zap.Dict("error_context", zaperrorcontext.AsContext(err)...),
			zap.Error(err))
	}
	//Output: Hello World
}
