package middleware

import (
	"context"
	"errors"

	"github.com/aws/aws-lambda-go/events"
)

// HandlerFunc represents the type of AWS Lambda APIGatewayProxy event handler.
// This is the ultimate target function of the middleware chain.
type HandlerFunc func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

// MiddlewareFunc represents the type of middleware that wraps a HandlerFunc and returns a new HandlerFunc.
// Middleware is used for request preprocessing, response postprocessing, or error handling.
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// Chain is a structure for building a middleware chain and applying it to a final handler.
// Middleware is executed in the order they are added (the first added is the outermost).
type Chain struct {
	middlewares []MiddlewareFunc
}

// NewChain creates a new middleware chain.
// The middleware passed as arguments will form the initial chain.
func NewChain(middlewares ...MiddlewareFunc) Chain {
	// Create a copy of the slice to prevent changes to the original slice
	newMiddlewares := make([]MiddlewareFunc, len(middlewares))
	copy(newMiddlewares, middlewares)
	return Chain{middlewares: newMiddlewares}
}

// Then adds a new middleware to the end of the existing chain.
// This method returns a new Chain instance, and the original Chain is not modified.
func (c Chain) Then(mw MiddlewareFunc) Chain {
	newMiddlewares := make([]MiddlewareFunc, len(c.middlewares)+1)
	copy(newMiddlewares, c.middlewares)
	newMiddlewares[len(c.middlewares)] = mw
	return Chain{middlewares: newMiddlewares}
}

// HandlerFunc applies the final HandlerFunc to the end of the middleware chain,
// and returns a HandlerFunc with all middleware applied.
// Middleware is executed in the order they were applied (the first added is the outermost).
// If the final handler is nil, a default handler that returns an error is used.
func (c Chain) HandlerFunc(final HandlerFunc) HandlerFunc {
	if final == nil {
		// Default handler
		final = func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			return events.APIGatewayProxyResponse{}, errors.New("no handler provided")
		}
	}

	// Apply in reverse order of the slice to make the first added middleware the outermost
	// Example: NewChain(m1, m2).Then(m3).HandlerFunc(h) executes in the order m1 -> m2 -> m3 -> h -> m3 -> m2 -> m1
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		final = c.middlewares[i](final)
	}
	return final
}

// Use is a helper function to apply multiple middleware to a single HandlerFunc.
// This is convenient when you want to apply middleware directly without using the Chain structure.
// Middleware is applied in reverse order of the arguments, so the execution order is the same as the argument order.
// Example: Use(h, m1, m2, m3) executes in the order m1 -> m2 -> m3 -> h -> m3 -> m2 -> m1
func Use(h HandlerFunc, middlewares ...MiddlewareFunc) HandlerFunc {
	return NewChain(middlewares...).HandlerFunc(h)
}
