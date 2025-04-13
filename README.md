# AWS Lambda Go Middleware

[![Go Reference](https://pkg.go.dev/badge/github.com/nakat-t/aws-lambda-go-middleware.svg)](https://pkg.go.dev/github.com/nakat-t/aws-lambda-go-middleware)
<!-- Add other badges like build status, code coverage, license etc. if applicable -->

`aws-lambda-go-middleware` is a library that provides `net/http` style middleware functionality for AWS Lambda Go handlers (handling `events.APIGatewayProxyRequest`). It allows you to modularize request preprocessing, response postprocessing, error handling, etc., and apply them to handlers as reusable components.

## Installation

```bash
go get github.com/nakat-t/aws-lambda-go-middleware/middleware
```

## Core Concepts

### `HandlerFunc`

A function type that represents the signature of an AWS Lambda API Gateway Proxy integration handler. This is the ultimate target of the middleware chain.

```go
type HandlerFunc func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)
```

### `MiddlewareFunc`

A function type that takes a `HandlerFunc` and returns a new `HandlerFunc`. Functions that implement this become middleware.

```go
type MiddlewareFunc func(next HandlerFunc) HandlerFunc
```

### `Use`

A function to apply middleware to a `HandlerFunc`.

```go
// Apply middleware m1, m2, m3 to handler h
wrappedHandler := middleware.Use(h, m1, m2, m3)
// Execution order: m1 -> m2 -> m3 -> h -> m3 -> m2 -> m1
```

## Usage

```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/nakat-t/aws-lambda-go-middleware/middleware"
	"github.com/nakat-t/aws-lambda-go-middleware/middleware/requestid"
	"github.com/nakat-t/aws-lambda-go-middleware/middleware/contenttype"
)

// Actual business logic
func myHandler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	reqID := ctx.Value(requestid.CtxKey{}).(string)
	log.Printf("Processing request: %s", reqID)
	// ... business logic ...
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{
			"Content-Type": "application/json",
		},
		Body: request.Body,
	}, nil
}

func main() {
    // Add request ID to context
	m1 := requestid.RequestID()
	// Allow only application/json
	m2 := contenttype.AllowContentType([]string{"application/json"})

	// Apply chain to handler
	wrappedHandler := middleware.Use(myHandler, m1, m2)

	// Start Lambda
	lambda.Start(wrappedHandler)
}

```

## Provided Middleware

### `AllowContentType`

Validates that the request's `Content-Type` header is included in the specified allowlist.

**Signature:**

```go
func AllowContentType(contentTypes []string, opts ...Option) middleware.MiddlewareFunc
```

**Options:**

```go
// Customize the response Content-Type header and body returned when Content-Type is not allowed.
func WithResponse(contentType string, body string) Option
```

**Comparison Rules:**

*   Only compares the media type part (e.g., `application/json` matches `application/json; charset=utf-8`).
*   Comparison is case-insensitive.
*   Returns `415 Unsupported Media Type` if the `Content-Type` header does not exist or is not in the allowlist.

## Sample Code

For a runnable sample code that includes examples of using `RequestID` and `AllowContentType`, refer to the `examples/middleware` directory in the repository.

```bash
# Run from the repository root directory
go run examples/middleware/main.go
```

### `RequestID`, `ExtendedRequestID`

Extract the request ID (`RequestContext.RequestID`) or the extended request ID (`RequestContext.ExtendedRequestID`) from the API Gateway request context and set it to `context.Context`. Subsequent middleware and handlers can retrieve this ID using the `CtxKey` key.

**Signature:**

```go
func RequestID(opts ...Option) middleware.MiddlewareFunc
```

**Options:**

```go
// WithCtxKey specifies the key of the request ID to be set in the context.
func WithCtxKey(ctxKey any) Option
```

### `StructuredLogger`

Creates middleware that logs request and response information using structured logging with `log/slog`.
Requests are automatically recorded before the handler is executed, and responses, execution times, and errors are automatically recorded after the handler is executed.

**Signature:**

```go
func StructuredLogger(opts ...Option) middleware.MiddlewareFunc
```

**Options:**

```go
// WithLogger sets a custom logger for the StructuredLogger middleware.
func WithLogger(logger *slog.Logger) Option

// WithRequestBodyLogging enables or disables request body logging in the middleware.
// By default, logging is disabled.
func WithRequestBodyLogging(enable bool) Option

// WithResponseBodyLogging enables or disables response body logging in the middleware.
// By default, logging is disabled.
func WithResponseBodyLogging(enable bool) Option
```

### `Validate`

This is middleware that validates the request body using the `github.com/go-playground/validator/v10` package. It unmarshals the request body into a variable of type T, performs validation, and if the validation passes, sets it to the context. If there is an error, it returns 400 Bad Request.

**Signature:**

```go
func Validate[T any](opts ...Option) middleware.MiddlewareFunc
```

**Options:**

```go
// WithCtxKey specifies the key of the unmarshaled request value to be set in the context.
func WithCtxKey(ctxKey any) Option

// Customize the response Content-Type header and body returned when validation error.
func WithResponse(contentType string, body string) Option
```

## License

This project is released under the license defined in the [LICENSE](LICENSE) file.
