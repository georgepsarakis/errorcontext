# errorcontext

Panic Handlers & Contextual Error Types for [zerolog](https://github.com/rs/zerolog), [zap](https://github.com/uber-go/zap)
Loggers & [OpenTelemetry Metrics](https://github.com/open-telemetry/opentelemetry-go).

## Why use this package?

- ✅ Cleaner Logs: No more hunting for IDs; your context is embedded in the error.

- ✅ Standardized Recovery: A consistent way to turn panics into trackable errors.

- ✅ Backend Agnostic: Works seamlessly with the logging libraries you already use.

Check out the documentation and start making your Go errors more meaningful!

🔗 Docs: https://pkg.go.dev/github.com/georgepsarakis/errorcontext

📦 GitHub: https://github.com/georgepsarakis/errorcontext

## Motivation

### Errors are more than plain messages

Error handling should ideally occur as high in the stack as possible.
A function that generates errors and must provide error context will likely:
1. Require coupling with the logging subsystem.
2. Require the addition of parameters which may be out-of-scope for the current function and only utilized as log record context (could be addressed with logger cloning see [zap#Logger.With](https://pkg.go.dev/go.uber.org/zap#Logger.With)).
3. Need to make a decision on whether to log the error context, with a potential overhead of redundant log records requiring correlation and increased log volumes.

However, this is often orthogonal to providing a comprehensive context within which the error was generated.
For example, a database error such as a unique constraint error often requires the HTTP Request ID known at the service layer and
other identifiers usually known at the query layer, to be included in the log record.

### Panic Handling

Panics are exceptional errors that potentially stop goroutine and eventually program execution. The process of handling a panic is
called _recovery_ and can only be performed within a deferred function and within the stack of its origin goroutine.
Unrecovered panics can cause downtime if they cause abrupt program shutdown. They can also go undetected or become difficult to inspect, as the stack trace is printed in an unstructured manner in stderr.

This package introduces a convenient wrapper that converts panics thrown by a function to error values.

## Examples

### zap

```go
package main

import (
	"errors"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/georgepsarakis/errorcontext"
	zaperrorcontext "github.com/georgepsarakis/errorcontext/backend/zap"
)

var ErrProcessingFailure = errors.New("processing failure")

func main() {
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
}

func tryWithError() error {
	return zaperrorcontext.NewError(
		ErrProcessingFailure,
		zap.String("path", "/a/b"),
		zap.Bool("enabled", true),
	)
}
```

### `Recoverer`

Panics are exceptional errors that signify undefined behavior and further execution may need to be stopped.
A common case is `nil` pointer dereference where an underlying value access is attempted while the pointer doesn't yet point to a value.
`Recoverer` provides a structured way of handling panics and converting them to error values.
Both the panic message and the stack trace are retained as error context.

In this example, `zap`-specific error types are used, but any error e.g. one constructed by `fmt.Errorf` can be used. See also `DefaultErrorGenerator`.

```go
package main


import (
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/georgepsarakis/errorcontext"
	zaperrorcontext "github.com/georgepsarakis/errorcontext/backend/zap"
)

func main() {
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
}
```

Will output (stack lines omitted for brevity):

```json
{"level":"warn","ts":1766772153.207416,"caller":"errorcontext/examples_test.go:63",
  "msg":"something failed",
  "error_context":{
    "panic":"panic: something bad happened",
    "stack":[".../runtime/panic.go:783 +0x120", "..."], "is_panic":true
  }, "error":"panic: something bad happened"}
```