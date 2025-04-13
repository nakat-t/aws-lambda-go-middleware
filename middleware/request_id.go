package middleware

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
)

// reqIDKey is the type for the key used to store the request ID in the context.
// Using a private type prevents key collisions with other packages.
type reqIDKey struct{}

// RequestID is middleware that extracts the request ID from the API Gateway request context
// and sets it in the Go context.Context.
// If the request ID does not exist, an empty string is set.
// Subsequent handlers and middleware can use the GetReqID function to retrieve the request ID from the context.
func RequestID() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			// Get request ID from APIGatewayProxyRequestContext
			reqID := request.RequestContext.RequestID

			// Set request ID in the new context
			// Using context.WithValue to associate reqID with the key reqIDKey{}
			ctxWithReqID := context.WithValue(ctx, reqIDKey{}, reqID)

			// Call the next handler with the new context containing the request ID
			return next(ctxWithReqID, request)
		}
	}
}

// GetReqID retrieves the request ID that was set by the RequestID middleware from the context.Context.
// If the request ID does not exist in the context, an empty string is returned.
func GetReqID(ctx context.Context) string {
	// Use context.Value to get the value associated with the key reqIDKey{}
	if reqID, ok := ctx.Value(reqIDKey{}).(string); ok {
		return reqID
	}
	// Return an empty string if the key doesn't exist or the type is not string
	return ""
}
