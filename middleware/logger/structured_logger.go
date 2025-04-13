package logger

import (
	"context"
	"log/slog"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/nakat-t/aws-lambda-go-middleware/middleware"
)

// Config represents the configuration for the StructuredLogger middleware.
type Config struct {
	logger                      *slog.Logger
	isRequestBodyLoggingEnable  bool
	isResponseBodyLoggingEnable bool
}

// Option is a function type to modify the StructuredLogger configuration.
type Option func(*Config)

// WithLogger sets a custom logger for the StructuredLogger middleware.
func WithLogger(logger *slog.Logger) Option {
	return func(c *Config) {
		c.logger = logger
	}
}

// WithRequestBodyLogging enables or disables request body logging in the middleware.
// By default, logging is disabled.
func WithRequestBodyLogging(enable bool) Option {
	return func(c *Config) {
		c.isRequestBodyLoggingEnable = enable
	}
}

// WithResponseBodyLogging enables or disables response body logging in the middleware.
// By default, logging is disabled.
func WithResponseBodyLogging(enable bool) Option {
	return func(c *Config) {
		c.isResponseBodyLoggingEnable = enable
	}
}

// StructuredLogger creates middleware that logs request and response information using structured logging.
//
// By default, it uses slog.Default() as the logger. A custom logger can be specified using the WithLogger option.
//
// The middleware logs:
//   - Before handler execution: Request information (excluding Body, but including Body size)
//   - After handler execution: Response information (excluding Body, but including Body size),
//     error if any, and execution duration
//
// Example:
//
//	// Use with default logger
//	handler := middleware.Use(myHandler, logger.StructuredLogger())
//
//	// Use with custom logger
//	customLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	handler := middleware.Use(myHandler, logger.StructuredLogger(WithLogger(customLogger)))
func StructuredLogger(opts ...Option) middleware.MiddlewareFunc {
	// Default configuration
	config := &Config{
		logger:                      slog.Default(),
		isRequestBodyLoggingEnable:  false,
		isResponseBodyLoggingEnable: false,
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	return func(next middleware.HandlerFunc) middleware.HandlerFunc {
		return func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			start := time.Now()

			// Log request information
			logRequest(ctx, config, &request)

			// Execute the handler
			response, err := next(ctx, request)

			// Calculate execution duration
			duration := time.Since(start)

			// Log response information, error if any, and execution duration
			logResponse(ctx, config, &response, err, duration)

			return response, err
		}
	}
}

// logRequest logs request information in a structured format.
func logRequest(ctx context.Context, config *Config, request *events.APIGatewayProxyRequest) {
	// Create a copy of the request with Body field cleared to avoid logging sensitive data
	reqCopy := *request
	bodySize := len(reqCopy.Body)
	if !config.isRequestBodyLoggingEnable {
		reqCopy.Body = "(omitted)"
	}

	config.logger.LogAttrs(ctx, slog.LevelInfo, "request received",
		slog.Any("request", reqCopy),
		slog.Int("bodySize", bodySize),
	)
}

// logResponse logs response information, error, and execution duration in a structured format.
func logResponse(
	ctx context.Context,
	config *Config,
	response *events.APIGatewayProxyResponse,
	err error,
	duration time.Duration,
) {
	// Create a copy of the response with Body field cleared to avoid logging sensitive data
	respCopy := *response
	bodySize := len(respCopy.Body)
	if !config.isResponseBodyLoggingEnable {
		respCopy.Body = "(omitted)"
	}

	attrs := []slog.Attr{
		slog.Any("response", respCopy),
		slog.Int("bodySize", bodySize),
		slog.Duration("duration", duration),
	}

	level := slog.LevelInfo
	message := "request processed successfully"

	if err != nil {
		level = slog.LevelError
		message = "request processing failed"
		attrs = append(attrs, slog.Any("error", err))
	}

	config.logger.LogAttrs(ctx, level, message, attrs...)
}
