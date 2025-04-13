package requestid

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/nakat-t/aws-lambda-go-middleware/middleware"
)

// CtxKey is the default key type used to store the request ID within the context.
type CtxKey struct{}

// Config is the configuration for the RequestID and ExtendedRequestID middleware.
type Config struct {
	ctxKey any
}

// Option is a function type to modify the RequestID and ExtendedRequestID configuration.
type Option func(*Config)

// WithCtxKey specifies the key of the request ID to be set in the context.
func WithCtxKey(ctxKey any) Option {
	return func(c *Config) {
		c.ctxKey = ctxKey
	}
}

// RequestID is middleware that extracts the request ID from the API Gateway request context
// and sets it in the Go context.Context.
// If the request ID does not exist, an empty string is set.
func RequestID(opts ...Option) middleware.MiddlewareFunc {
	// Default configuration
	config := Config{
		ctxKey: CtxKey{},
	}
	// Apply options
	for _, opt := range opts {
		opt(&config)
	}

	return func(next middleware.HandlerFunc) middleware.HandlerFunc {
		return func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			// Get request ID from APIGatewayProxyRequestContext
			reqID := request.RequestContext.RequestID

			// Set request ID in the new context
			ctxWithReqID := context.WithValue(ctx, config.ctxKey, reqID)

			// Call the next handler with the new context containing the request ID
			return next(ctxWithReqID, request)
		}
	}
}

// ExtendedRequestID is middleware that extracts the extended request ID from the API Gateway request context
// and sets it in the Go context.Context.
// If the extended request ID does not exist, an empty string is set.
func ExtendedRequestID(opts ...Option) middleware.MiddlewareFunc {
	// Default configuration
	config := Config{
		ctxKey: CtxKey{},
	}
	// Apply options
	for _, opt := range opts {
		opt(&config)
	}

	return func(next middleware.HandlerFunc) middleware.HandlerFunc {
		return func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			// Get extended request ID from APIGatewayProxyRequestContext
			reqID := request.RequestContext.ExtendedRequestID

			// Set extended request ID in the new context
			ctxWithReqID := context.WithValue(ctx, config.ctxKey, reqID)

			// Call the next handler with the new context containing the request ID
			return next(ctxWithReqID, request)
		}
	}
}
