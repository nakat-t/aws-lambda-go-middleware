package requestid

import (
	"context"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestRequestID(t *testing.T) {
	tests := []struct {
		name           string
		inputRequestID string
		expectedReqID  string
	}{
		{
			name:           "When request ID exists",
			inputRequestID: "test-request-id-123",
			expectedReqID:  "test-request-id-123",
		},
		{
			name:           "When request ID does not exist (empty string)",
			inputRequestID: "",
			expectedReqID:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For assertions
			assert := assert.New(t)

			// Create request for testing
			request := events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					RequestID: tt.inputRequestID,
				},
			}

			// Mock final handler
			// Call ctx.Value inside this handler and verify that the expected value can be retrieved
			mockHandler := func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
				// Get request ID from context
				actualReqID := ctx.Value(CtxKey{})
				// Assert that it matches the expected request ID
				assert.Equal(tt.expectedReqID, actualReqID, "ctx.Value should return the correct request ID")

				// Verify that the original request object is not modified (just in case)
				assert.Equal(request, req, "Request object should not be modified")

				// Return a dummy response
				return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
			}

			// Apply RequestID middleware
			handlerWithMiddleware := RequestID()(mockHandler)

			// Execute the handler with middleware applied
			response, err := handlerWithMiddleware(context.Background(), request)

			// Verify no error occurs and status code is OK
			assert.NoError(err, "Handler should not return an error")
			assert.Equal(http.StatusOK, response.StatusCode, "Status code should be OK")
		})
	}
}

func TestExtendedRequestID(t *testing.T) {
	tests := []struct {
		name                  string
		inputExtendedReqID    string
		expectedExtendedReqID string
	}{
		{
			name:                  "When extended request ID exists",
			inputExtendedReqID:    "extended-req-id-abc-123",
			expectedExtendedReqID: "extended-req-id-abc-123",
		},
		{
			name:                  "When extended request ID does not exist",
			inputExtendedReqID:    "",
			expectedExtendedReqID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For assertions
			assert := assert.New(t)

			// Create request for testing
			request := events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					ExtendedRequestID: tt.inputExtendedReqID,
				},
			}

			// Mock final handler
			mockHandler := func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
				// Get extended request ID from context
				actualExtendedReqID := ctx.Value(CtxKey{})
				// Assert that it matches the expected extended request ID
				assert.Equal(tt.expectedExtendedReqID, actualExtendedReqID, "ctx.Value should return the correct extended request ID")

				// Return a dummy response
				return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
			}

			// Apply ExtendedRequestID middleware
			handlerWithMiddleware := ExtendedRequestID()(mockHandler)

			// Execute the handler with middleware applied
			response, err := handlerWithMiddleware(context.Background(), request)

			// Verify no error occurs and status code is OK
			assert.NoError(err, "Handler should not return an error")
			assert.Equal(http.StatusOK, response.StatusCode, "Status code should be OK")
		})
	}
}

func TestRequestIDWithCustomCtxKey(t *testing.T) {
	// Custom context key type
	type customCtxKey struct{}

	// For assertions
	assert := assert.New(t)

	// Test data
	testReqID := "custom-key-request-id"
	request := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: testReqID,
		},
	}

	// Create custom key
	customKey := customCtxKey{}

	// Mock final handler
	mockHandler := func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		// Get request ID using custom key
		actualReqID := ctx.Value(customKey)
		// Verify it's the expected value
		assert.Equal(testReqID, actualReqID, "ctx.Value with custom key should return the correct request ID")

		// Verify the default key doesn't have a value (since we used a custom key)
		defaultKeyValue := ctx.Value(CtxKey{})
		assert.Nil(defaultKeyValue, "Default context key should not contain a value when custom key is used")

		return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
	}

	// Apply RequestID middleware with custom key option
	handlerWithMiddleware := RequestID(WithCtxKey(customKey))(mockHandler)

	// Execute the handler with middleware applied
	response, err := handlerWithMiddleware(context.Background(), request)

	// Verify no error occurs and status code is OK
	assert.NoError(err, "Handler should not return an error")
	assert.Equal(http.StatusOK, response.StatusCode, "Status code should be OK")
}

func TestExtendedRequestIDWithCustomCtxKey(t *testing.T) {
	// Custom context key type
	type customExtendedReqIDKey struct{}

	// For assertions
	assert := assert.New(t)

	// Test data
	testExtendedReqID := "custom-key-extended-request-id"
	request := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			ExtendedRequestID: testExtendedReqID,
		},
	}

	// Create custom key
	customKey := customExtendedReqIDKey{}

	// Mock final handler
	mockHandler := func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		// Get extended request ID using custom key
		actualExtendedReqID := ctx.Value(customKey)
		// Verify it's the expected value
		assert.Equal(testExtendedReqID, actualExtendedReqID, "ctx.Value with custom key should return the correct extended request ID")

		return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
	}

	// Apply ExtendedRequestID middleware with custom key option
	handlerWithMiddleware := ExtendedRequestID(WithCtxKey(customKey))(mockHandler)

	// Execute the handler with middleware applied
	response, err := handlerWithMiddleware(context.Background(), request)

	// Verify no error occurs and status code is OK
	assert.NoError(err, "Handler should not return an error")
	assert.Equal(http.StatusOK, response.StatusCode, "Status code should be OK")
}

func TestRequestIDWithMultipleOptions(t *testing.T) {
	// This test verifies that multiple options can be applied and the last one takes precedence
	// Custom context key types
	type firstCtxKey struct{}
	type secondCtxKey struct{}

	// For assertions
	assert := assert.New(t)

	// Test data
	testReqID := "multiple-options-req-id"
	request := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: testReqID,
		},
	}

	// Create custom keys
	firstKey := firstCtxKey{}
	secondKey := secondCtxKey{}

	// Mock final handler
	mockHandler := func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		// The second key should take precedence
		firstKeyValue := ctx.Value(firstKey)
		secondKeyValue := ctx.Value(secondKey)

		assert.Nil(firstKeyValue, "First context key should not contain a value because it was overridden")
		assert.Equal(testReqID, secondKeyValue, "Second context key should contain the request ID")

		return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
	}

	// Apply RequestID middleware with multiple custom key options
	// The last option should take precedence
	handlerWithMiddleware := RequestID(
		WithCtxKey(firstKey),
		WithCtxKey(secondKey),
	)(mockHandler)

	// Execute the handler with middleware applied
	response, err := handlerWithMiddleware(context.Background(), request)

	// Verify no error occurs and status code is OK
	assert.NoError(err, "Handler should not return an error")
	assert.Equal(http.StatusOK, response.StatusCode, "Status code should be OK")
}

func TestGetReqID_ContextWithoutID(t *testing.T) {
	// For assertions
	assert := assert.New(t)

	// Empty context not set by RequestID middleware
	ctx := context.Background()

	// Call ctx.Value on a context without a request ID set
	reqID := ctx.Value(CtxKey{})

	// Expect an empty string to be returned
	assert.Empty(reqID, "ctx.Value should return an empty string for context without request ID")
}
