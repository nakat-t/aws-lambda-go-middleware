package middleware

import (
	"context"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

// mockNextHandler is a final handler for testing.
// It always returns 200 OK when called.
var mockNextHandler = func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: "OK"}, nil
}

// createRequest creates an APIGatewayProxyRequest for testing.
func createRequest(contentType string) events.APIGatewayProxyRequest {
	headers := make(map[string]string)
	if contentType != "" {
		// Set with normalized key using http.CanonicalHeaderKey
		headers[http.CanonicalHeaderKey("Content-Type")] = contentType
	}
	return events.APIGatewayProxyRequest{
		Headers: headers,
	}
}

func TestAllowContentType(t *testing.T) {
	tests := []struct {
		name                string
		allowedContentTypes []string
		requestContentType  string
		expectedStatusCode  int
		expectNextCalled    bool // Whether we expect the next handler to be called
	}{
		{
			name:                "Allowed Content-Type (exact match)",
			allowedContentTypes: []string{"application/json"},
			requestContentType:  "application/json",
			expectedStatusCode:  http.StatusOK,
			expectNextCalled:    true,
		},
		{
			name:                "Allowed Content-Type (with parameters)",
			allowedContentTypes: []string{"application/json"},
			requestContentType:  "application/json; charset=utf-8",
			expectedStatusCode:  http.StatusOK,
			expectNextCalled:    true,
		},
		{
			name:                "Allowed Content-Type (case insensitive)",
			allowedContentTypes: []string{"application/json"},
			requestContentType:  "Application/JSON",
			expectedStatusCode:  http.StatusOK,
			expectNextCalled:    true,
		},
		{
			name:                "Disallowed Content-Type",
			allowedContentTypes: []string{"application/json"},
			requestContentType:  "text/xml",
			expectedStatusCode:  http.StatusUnsupportedMediaType,
			expectNextCalled:    false,
		},
		{
			name:                "Missing Content-Type header",
			allowedContentTypes: []string{"application/json"},
			requestContentType:  "", // Indicates no header
			expectedStatusCode:  http.StatusUnsupportedMediaType,
			expectNextCalled:    false,
		},
		{
			name:                "Invalid Content-Type header",
			allowedContentTypes: []string{"application/json"},
			requestContentType:  "invalid-content-type",
			expectedStatusCode:  http.StatusUnsupportedMediaType,
			expectNextCalled:    false,
		},
		{
			name:                "Empty allowlist",
			allowedContentTypes: []string{}, // Empty list
			requestContentType:  "application/json",
			expectedStatusCode:  http.StatusUnsupportedMediaType,
			expectNextCalled:    false,
		},
		{
			name:                "Multiple allowed types (allowed case)",
			allowedContentTypes: []string{"application/json", "application/xml"},
			requestContentType:  "application/xml",
			expectedStatusCode:  http.StatusOK,
			expectNextCalled:    true,
		},
		{
			name:                "Multiple allowed types (disallowed case)",
			allowedContentTypes: []string{"application/json", "application/xml"},
			requestContentType:  "text/plain",
			expectedStatusCode:  http.StatusUnsupportedMediaType,
			expectNextCalled:    false,
		},
		{
			name:                "Allowlist contains invalid type (ignored)",
			allowedContentTypes: []string{"application/json", "invalid-type"},
			requestContentType:  "application/json",
			expectedStatusCode:  http.StatusOK,
			expectNextCalled:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			nextCalled := false

			// Mock to record if the next handler was called
			mockHandler := func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
				nextCalled = true
				return mockNextHandler(ctx, request)
			}

			// Create the middleware under test
			middleware := AllowContentType(tt.allowedContentTypes)
			handlerWithMiddleware := middleware(mockHandler)

			// Create the request
			request := createRequest(tt.requestContentType)

			// Execute the handler
			response, err := handlerWithMiddleware(context.Background(), request)

			// Assertions
			assert.NoError(err)
			assert.Equal(tt.expectedStatusCode, response.StatusCode)
			assert.Equal(tt.expectNextCalled, nextCalled, "Next handler call expectation mismatch")

			if !tt.expectNextCalled {
				// Check default body for error case
				assert.Contains(response.Body, defaultUnsupportedMediaTypeBody)
				assert.Equal("text/plain; charset=utf-8", response.Headers["Content-Type"])
			} else {
				assert.Equal("OK", response.Body) // Body returned by next handler
			}
		})
	}
}

func TestAllowContentType_WithOptions(t *testing.T) {
	assert := assert.New(t)
	customErrorBody := "Invalid Content Type Provided"
	nextCalled := false

	// Mock to record if the next handler was called
	mockHandler := func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		nextCalled = true
		return mockNextHandler(ctx, request)
	}

	// Create middleware with options
	middleware := AllowContentType(
		[]string{"application/vnd.api+json"},
		WithResponseBody(customErrorBody),
	)
	handlerWithMiddleware := middleware(mockHandler)

	// --- Test Case 1: Allowed Content Type ---
	requestAllowed := createRequest("application/vnd.api+json")
	responseAllowed, errAllowed := handlerWithMiddleware(context.Background(), requestAllowed)

	assert.NoError(errAllowed)
	assert.Equal(http.StatusOK, responseAllowed.StatusCode)
	assert.True(nextCalled, "Next handler should be called for allowed type")
	assert.Equal("OK", responseAllowed.Body)

	// --- Test Case 2: Disallowed Content Type ---
	nextCalled = false                                     // Reset flag
	requestDisallowed := createRequest("application/json") // Not in the allowed list for this test
	responseDisallowed, errDisallowed := handlerWithMiddleware(context.Background(), requestDisallowed)

	assert.NoError(errDisallowed) // Middleware handles the error, returns response
	assert.Equal(http.StatusUnsupportedMediaType, responseDisallowed.StatusCode)
	assert.False(nextCalled, "Next handler should not be called for disallowed type")
	// Check custom error body
	assert.Equal(customErrorBody, responseDisallowed.Body)
	assert.Equal("text/plain; charset=utf-8", responseDisallowed.Headers["Content-Type"])

	// --- Test Case 3: Missing Content Type ---
	nextCalled = false                  // Reset flag
	requestMissing := createRequest("") // No Content-Type header
	responseMissing, errMissing := handlerWithMiddleware(context.Background(), requestMissing)

	assert.NoError(errMissing)
	assert.Equal(http.StatusUnsupportedMediaType, responseMissing.StatusCode)
	assert.False(nextCalled, "Next handler should not be called for missing header")
	assert.Equal(customErrorBody, responseMissing.Body) // Custom error body should be used
}

func TestAllowContentType_DefaultOptions(t *testing.T) {
	assert := assert.New(t)
	nextCalled := false

	mockHandler := func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		nextCalled = true
		return mockNextHandler(ctx, request)
	}

	// Create middleware without options (by default, nothing is allowed)
	middleware := AllowContentType([]string{})
	handlerWithMiddleware := middleware(mockHandler)

	request := createRequest("application/json")
	response, err := handlerWithMiddleware(context.Background(), request)

	assert.NoError(err)
	assert.Equal(http.StatusUnsupportedMediaType, response.StatusCode)
	assert.False(nextCalled)
	// Check default error body
	assert.Equal(defaultUnsupportedMediaTypeBody, response.Body)
}
