# errorcontext

Contextual Error Types for [zerolog](https://github.com/rs/zerolog), [zap](https://github.com/uber-go/zap) 
Loggers & [OpenTelemetry Metrics](https://github.com/open-telemetry/opentelemetry-go).

## Examples

### zap

```go
package main

import (
	"errors"

	"go.uber.org/zap"

	zaperrorcontext "github.com/georgepsarakis/errorcontext/backend/zap"
)

var ErrProcessingFailure = errors.New("processing failure")

func main() {
	cfg := zap.NewProductionConfig()
	zapLogger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	defer zapLogger.Sync()

	err = process()
	if errors.Is(err, ErrProcessingFailure) {
		zapLogger.Warn("something failed",
			zap.Dict("error_context", zaperrorcontext.AsContext(err)...),
			zap.Error(err))
	}
}

func process() error {
	return zaperrorcontext.NewError(
		ErrProcessingFailure,
		zap.String("path", "/a/b"),
		zap.Bool("enabled", true),
	)
}
```
