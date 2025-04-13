package contenttype

import (
	"context"
	"mime"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/nakat-t/aws-lambda-go-middleware/middleware"
)

const (
	// defaultErrorBody is the default response body when Content-Type is not allowed.
	defaultErrorBody = "Unsupported Media Type"

	// defaultErrorContentType is the default Content-Type for error responses.
	defaultErrorContentType = "text/plain; charset=utf-8"
)

// AllowContentTypeConfig is the configuration for the AllowContentType middleware.
type AllowContentTypeConfig struct {
	allowedTypes     []string
	errorBody        string
	errorContentType string
}

// AllowContentTypeOption is a function type to modify the AllowContentType configuration.
type AllowContentTypeOption func(*AllowContentTypeConfig)

// WithResponse sets the response Content-Type header and response body for error cases.
func WithResponse(contentType string, body string) AllowContentTypeOption {
	return func(c *AllowContentTypeConfig) {
		c.errorContentType = contentType
		c.errorBody = body
	}
}

// AllowContentType creates middleware that validates if the request's Content-Type header is included in the specified list.
//
// If the Content-Type header does not exist or has a media type not in the list,
// it returns a response with status code 415 (Unsupported Media Type) and "Unsupported Media Type" body by default.
// The response body can be customized with the WithResponse option.
//
// If the contentTypes list is empty, all Content-Types will be rejected.
// Content-Type comparison is done only on the media type part, parameters (e.g., charset=utf-8) are ignored.
// Comparison is case-insensitive.
//
// Examples:
// AllowContentType([]string{"application/json"}) allows "application/json" and "application/json; charset=utf-8".
// AllowContentType([]string{"application/json", "application/xml"}) allows both JSON and XML.
func AllowContentType(contentTypes []string, opts ...AllowContentTypeOption) middleware.MiddlewareFunc {
	// Default configuration
	config := AllowContentTypeConfig{
		allowedTypes:     contentTypes,
		errorBody:        defaultErrorBody,
		errorContentType: defaultErrorContentType,
	}
	// Apply options
	for _, opt := range opts {
		opt(&config)
	}

	// Convert allowed Content-Types to lowercase and store in a map
	allowedMap := make(map[string]struct{}, len(config.allowedTypes))
	for _, ct := range config.allowedTypes {
		mediaType, _, err := mime.ParseMediaType(strings.ToLower(ct))
		if err == nil {
			allowedMap[mediaType] = struct{}{}
		}
	}

	// Prepare error response
	errorResponse := events.APIGatewayProxyResponse{
		StatusCode: http.StatusUnsupportedMediaType,
		Body:       config.errorBody,
		Headers:    map[string]string{"Content-Type": config.errorContentType},
	}

	return func(next middleware.HandlerFunc) middleware.HandlerFunc {
		return func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			contentTypeHeader := request.Headers[http.CanonicalHeaderKey("Content-Type")]

			if contentTypeHeader == "" {
				// One could consider allowing requests without Content-Type (like GET), but
				// chi's AllowContentType also rejects requests without headers, so we follow that approach.
				return errorResponse, nil
			}

			mediaType, _, err := mime.ParseMediaType(strings.ToLower(contentTypeHeader))
			if err != nil {
				// Also reject if parsing fails
				return errorResponse, nil
			}

			if _, ok := allowedMap[mediaType]; !ok {
				// If more detailed error messages are needed, they can be set here
				// errorResponse.Body = fmt.Sprintf("%s: Content-Type '%s' not allowed. Allowed: %v", config.ErrorBody, mediaType, config.AllowedTypes)
				return errorResponse, nil
			}

			return next(ctx, request)
		}
	}
}
