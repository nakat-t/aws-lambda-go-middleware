package validate

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/nakat-t/aws-lambda-go-middleware/middleware"
	"github.com/stretchr/testify/assert"
)

// Sample struct for testing
type TestUser struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=0,lte=130"`
}

// Custom context key for testing
type TestCtxKey string

// Helper function to mock the next handler
func mockHandler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "success",
	}, nil
}

// Mock handler to retrieve data from context on success
func mockHandlerWithContext(key any) middleware.HandlerFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		data, ok := ctx.Value(key).(TestUser)
		if !ok {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       "failed to get context value",
			}, nil
		}
		responseBody, _ := json.Marshal(data)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Body:       string(responseBody),
		}, nil
	}
}

func TestValidate_ValidRequest(t *testing.T) {
	// Create valid user data
	validUser := TestUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}
	jsonData, _ := json.Marshal(validUser)

	// Create a request
	req := events.APIGatewayProxyRequest{
		Body: string(jsonData),
	}

	// Create Validate middleware with mock handler and context key
	handler := Validate[TestUser]()(mockHandlerWithContext(CtxKey{}))

	// Execute middleware
	resp, err := handler(context.Background(), req)

	// Assertion
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Unmarshal the response body and compare it with the original data
	var responseUser TestUser
	err = json.Unmarshal([]byte(resp.Body), &responseUser)
	assert.NoError(t, err)
	assert.Equal(t, validUser, responseUser)
}

func TestValidate_InvalidRequest(t *testing.T) {
	// Invalid user data (missing required field)
	invalidUser := TestUser{
		Name: "John Doe",
		// Email is missing
		Age: 30,
	}
	jsonData, _ := json.Marshal(invalidUser)

	// Create a request
	req := events.APIGatewayProxyRequest{
		Body: string(jsonData),
	}

	// Create Validate middleware
	handler := Validate[TestUser]()(mockHandler)

	// Execute middleware
	resp, err := handler(context.Background(), req)

	// Assertion
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, defaultErrorBody, resp.Body)
	assert.Equal(t, defaultErrorContentType, resp.Headers["Content-Type"])
}

func TestValidate_InvalidJSON(t *testing.T) {
	// Invalid JSON format
	invalidJSON := `{"name": "John Doe", "email": "john@example.com", "age": 30,}` // Extra comma

	// Create a request
	req := events.APIGatewayProxyRequest{
		Body: invalidJSON,
	}

	// Create Validate middleware
	handler := Validate[TestUser]()(mockHandler)

	// Execute middleware
	resp, err := handler(context.Background(), req)

	// Assertion
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestValidate_EmptyBody(t *testing.T) {
	// Empty request body
	req := events.APIGatewayProxyRequest{
		Body: "",
	}

	// Create Validate middleware
	handler := Validate[TestUser]()(mockHandler)

	// Execute middleware
	resp, err := handler(context.Background(), req)

	// Assertion
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestValidate_WithCustomCtxKey(t *testing.T) {
	// Create valid user data
	validUser := TestUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}
	jsonData, _ := json.Marshal(validUser)

	// Create a request
	req := events.APIGatewayProxyRequest{
		Body: string(jsonData),
	}

	// Use a custom context key
	customKey := TestCtxKey("user")

	// Create Validate middleware with custom key
	handler := Validate[TestUser](WithCtxKey(customKey))(mockHandlerWithContext(customKey))

	// Execute middleware
	resp, err := handler(context.Background(), req)

	// Assertion
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Unmarshal the response body and compare it with the original data
	var responseUser TestUser
	err = json.Unmarshal([]byte(resp.Body), &responseUser)
	assert.NoError(t, err)
	assert.Equal(t, validUser, responseUser)
}

func TestValidate_WithCustomResponse(t *testing.T) {
	// Invalid user data
	invalidUser := TestUser{
		Name: "John Doe",
		// Email is missing
		Age: 30,
	}
	jsonData, _ := json.Marshal(invalidUser)

	// Create a request
	req := events.APIGatewayProxyRequest{
		Body: string(jsonData),
	}

	// Set a custom error response
	customContentType := "application/json"
	customBody := `{"error": "Validation failed"}`

	// Create Validate middleware with custom response
	handler := Validate[TestUser](WithResponse(customContentType, customBody))(mockHandler)

	// Execute middleware
	resp, err := handler(context.Background(), req)

	// Assertion
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, customBody, resp.Body)
	assert.Equal(t, customContentType, resp.Headers["Content-Type"])
}

func TestValidate_InvalidAge(t *testing.T) {
	// User with invalid age value (out of range)
	invalidUser := TestUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   150, // Greater than 130
	}
	jsonData, _ := json.Marshal(invalidUser)

	// Create a request
	req := events.APIGatewayProxyRequest{
		Body: string(jsonData),
	}

	// Create Validate middleware
	handler := Validate[TestUser]()(mockHandler)

	// Execute middleware
	resp, err := handler(context.Background(), req)

	// Assertion
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestValidate_InvalidEmail(t *testing.T) {
	// User with invalid email address
	invalidUser := TestUser{
		Name:  "John Doe",
		Email: "not-an-email", // Invalid email address
		Age:   30,
	}
	jsonData, _ := json.Marshal(invalidUser)

	// Create a request
	req := events.APIGatewayProxyRequest{
		Body: string(jsonData),
	}

	// Create Validate middleware
	handler := Validate[TestUser]()(mockHandler)

	// Execute middleware
	resp, err := handler(context.Background(), req)

	// Assertion
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
