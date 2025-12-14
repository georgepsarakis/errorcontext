package examples

import (
	"errors"

	"go.uber.org/zap"

	zaperrorcontext "github.com/georgepsarakis/errorcontext/backend/zap"
)

var ErrProcessingFailure = errors.New("processing failure")

func ExampleAsError() {
	cfg := zap.NewProductionConfig()
	zapLogger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	defer zapLogger.Sync()
	err = tryWithError()
	if errors.Is(err, ErrProcessingFailure) {
		zapLogger.Warn("something failed",
			zap.Dict("error_context", zaperrorcontext.AsError(err)...),
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
