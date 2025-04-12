package middleware

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
			// Call GetReqID inside this handler and verify that the expected value can be retrieved
			mockHandler := func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
				// Get request ID from context
				actualReqID := GetReqID(ctx)
				// Assert that it matches the expected request ID
				assert.Equal(tt.expectedReqID, actualReqID, "GetReqID should return the correct request ID")

				// Verify that the original request object is not modified (just in case)
				assert.Equal(request, req, "Request object should not be modified")

				// Return a dummy response
				return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
			}

			// Apply RequestID middleware
			handlerWithMiddleware := RequestID(mockHandler)

			// Execute the handler with middleware applied
			response, err := handlerWithMiddleware(context.Background(), request)

			// Verify no error occurs and status code is OK
			assert.NoError(err, "Handler should not return an error")
			assert.Equal(http.StatusOK, response.StatusCode, "Status code should be OK")
		})
	}
}

func TestGetReqID_ContextWithoutID(t *testing.T) {
	// For assertions
	assert := assert.New(t)

	// Empty context not set by RequestID middleware
	ctx := context.Background()

	// Call GetReqID on a context without a request ID set
	reqID := GetReqID(ctx)

	// Expect an empty string to be returned
	assert.Empty(reqID, "GetReqID should return an empty string for context without request ID")
}
