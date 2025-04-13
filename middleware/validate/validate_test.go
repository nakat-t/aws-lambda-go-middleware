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
	Name  string `json:"name" xml:"name" validate:"required"`
	Email string `json:"email" xml:"email" validate:"required,email"`
	Age   int    `json:"age" xml:"age" validate:"gte=0,lte=130"`
}

// TestUserWithUnmarshaler implements the RequestUnmarshaler interface
type TestUserWithUnmarshaler struct {
	Name     string `validate:"required"`
	Email    string `validate:"required,email"`
	Age      int    `validate:"gte=0,lte=130"`
	FromJSON bool   // Added to track which unmarshal method was used
}

// UnmarshalFromRequest implements the RequestUnmarshaler interface
func (u *TestUserWithUnmarshaler) UnmarshalFromRequest(data []byte) error {
	// For test purposes, manually set the values
	u.Name = "John Custom"
	u.Email = "john.custom@example.com"
	u.Age = 35
	u.FromJSON = false
	return nil
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

func TestValidate_XMLRequest(t *testing.T) {
	// Valid user data in XML format
	xmlData := `<TestUser><name>John Doe</name><email>john@example.com</email><age>30</age></TestUser>`

	// Create a request
	req := events.APIGatewayProxyRequest{
		Body: xmlData,
	}

	// Create Validate middleware with handler that returns validated data
	handler := Validate[TestUser]()(mockHandlerWithContext(CtxKey{}))

	// Execute middleware
	resp, err := handler(context.Background(), req)

	// Assertion
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Unmarshal the response body to check data
	var responseUser TestUser
	err = json.Unmarshal([]byte(resp.Body), &responseUser)
	assert.NoError(t, err)
	assert.Equal(t, "John Doe", responseUser.Name)
	assert.Equal(t, "john@example.com", responseUser.Email)
	assert.Equal(t, 30, responseUser.Age)
}

func TestValidate_WithCustomUnmarshaler(t *testing.T) {
	// The actual content doesn't matter since we use our custom unmarshaler
	req := events.APIGatewayProxyRequest{
		Body: `{"dummy": "data"}`,
	}

	// Create a handler that returns the validated data
	handlerWithUnmarshaler := func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		data, ok := ctx.Value(CtxKey{}).(TestUserWithUnmarshaler)
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

	// Create Validate middleware
	handler := Validate[TestUserWithUnmarshaler]()(handlerWithUnmarshaler)

	// Execute middleware
	resp, err := handler(context.Background(), req)

	// Assertion
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Unmarshal the response body and check
	var responseUser TestUserWithUnmarshaler
	err = json.Unmarshal([]byte(resp.Body), &responseUser)
	assert.NoError(t, err)
	assert.Equal(t, "John Custom", responseUser.Name)
	assert.Equal(t, "john.custom@example.com", responseUser.Email)
	assert.Equal(t, 35, responseUser.Age)
	assert.False(t, responseUser.FromJSON) // Confirm it used the custom unmarshaler
}

func TestValidate_InvalidXML(t *testing.T) {
	// Invalid XML format
	invalidXML := `<TestUser><name>John Doe<name><email>john@example.com</email><age>30</age></TestUser>` // Missing closing tag

	// Create a request
	req := events.APIGatewayProxyRequest{
		Body: invalidXML,
	}

	// Create Validate middleware
	handler := Validate[TestUser]()(mockHandler)

	// Execute middleware
	resp, err := handler(context.Background(), req)

	// Assertion
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDetermineContentType(t *testing.T) {
	testCases := []struct {
		name     string
		body     string
		expected string
	}{
		{"JSON Object", `{"name": "John"}`, "json"},
		{"JSON Array", `[1, 2, 3]`, "json"},
		{"XML Document", `<root><item>value</item></root>`, "xml"},
		{"Leading Whitespace JSON", `   {"name": "John"}`, "json"},
		{"Leading Whitespace XML", `   <root>value</root>`, "xml"},
		{"Unknown Format", `Hello, world!`, "unknown"},
		{"Empty String", ``, "unknown"},
		{"Only Whitespace", `   `, "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := determineContentType(tc.body)
			assert.Equal(t, tc.expected, result)
		})
	}
}
