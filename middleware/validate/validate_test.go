package validate

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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

// UnmarshalRequest implements the RequestUnmarshaler interface
func (u *TestUserWithUnmarshaler) UnmarshalRequest(data []byte) error {
	// For test purposes, manually set the values
	u.Name = "John Custom"
	u.Email = "john.custom@example.com"
	u.Age = 35
	u.FromJSON = false
	return nil
}

// TestUserWithValidator implements the Validator interface
type TestUserWithValidator struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Age         int    `json:"age"`
	IsValidated bool   `json:"-"`          // For test purposes, to track if Validate was called
	ShouldFail  bool   `json:"shouldFail"` // Used to test validation failure
}

// Validate implements the Validator interface
func (u *TestUserWithValidator) Validate() error {
	u.IsValidated = true

	// Basic validation logic
	if u.Name == "" {
		return errors.New("name is required")
	}

	if u.Email == "" {
		return errors.New("email is required")
	}

	if u.Age < 0 || u.Age > 130 {
		return errors.New("age must be between 0 and 130")
	}

	// Used for testing validation failure
	if u.ShouldFail {
		return errors.New("validation intentionally failed")
	}

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

// Generic mock handler to retrieve any type from context
func mockHandlerWithGenericContext[T any](key any) middleware.HandlerFunc {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		data, ok := ctx.Value(key).(T)
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

func TestValidate_WithCustomValidator_Success(t *testing.T) {
	// Valid user data with custom validator
	validUser := TestUserWithValidator{
		Name:       "Jane Doe",
		Email:      "jane@example.com",
		Age:        28,
		ShouldFail: false,
	}
	jsonData, _ := json.Marshal(validUser)

	// Create a request
	req := events.APIGatewayProxyRequest{
		Body: string(jsonData),
	}

	// Create a special handler that directly checks the IsValidated flag
	handlerWithValidator := func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		data, ok := ctx.Value(CtxKey{}).(TestUserWithValidator)
		if !ok {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       "failed to get context value",
			}, nil
		}

		// Direct assertion on IsValidated flag
		if !data.IsValidated {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       "validator was not called",
			}, nil
		}

		// Return success with the data for other assertions
		responseBody, _ := json.Marshal(data)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Body:       string(responseBody),
		}, nil
	}

	// Create Validate middleware
	handler := Validate[TestUserWithValidator]()(handlerWithValidator)

	// Execute middleware
	resp, err := handler(context.Background(), req)

	// Assertion
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Unmarshal the response body and check
	var responseUser TestUserWithValidator
	err = json.Unmarshal([]byte(resp.Body), &responseUser)
	assert.NoError(t, err)
	assert.Equal(t, validUser.Name, responseUser.Name)
	assert.Equal(t, validUser.Email, responseUser.Email)
	assert.Equal(t, validUser.Age, responseUser.Age)
	// No need to check IsValidated here, as it was checked in the handler
}

func TestValidate_WithCustomValidator_Failure(t *testing.T) {
	// User data that will fail custom validation
	invalidUser := TestUserWithValidator{
		Name:       "Jane Doe",
		Email:      "jane@example.com",
		Age:        28,
		ShouldFail: true, // This will cause the Validate() method to return an error
	}
	jsonData, _ := json.Marshal(invalidUser)

	// Create a request
	req := events.APIGatewayProxyRequest{
		Body: string(jsonData),
	}

	// Create Validate middleware
	handler := Validate[TestUserWithValidator]()(mockHandler)

	// Execute middleware
	resp, err := handler(context.Background(), req)

	// Assertion
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, defaultErrorBody, resp.Body)
	assert.Equal(t, defaultErrorContentType, resp.Headers["Content-Type"])
}

func TestValidate_WithCustomValidator_MissingFields(t *testing.T) {
	// User with missing required fields
	invalidUser := TestUserWithValidator{
		Name:  "", // Missing name
		Email: "jane@example.com",
		Age:   28,
	}
	jsonData, _ := json.Marshal(invalidUser)

	// Create a request
	req := events.APIGatewayProxyRequest{
		Body: string(jsonData),
	}

	// Create Validate middleware
	handler := Validate[TestUserWithValidator]()(mockHandler)

	// Execute middleware
	resp, err := handler(context.Background(), req)

	// Assertion
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestValidate_WithCustomValidator_InvalidAge(t *testing.T) {
	// User with invalid age
	invalidUser := TestUserWithValidator{
		Name:  "Jane Doe",
		Email: "jane@example.com",
		Age:   150, // Greater than 130
	}
	jsonData, _ := json.Marshal(invalidUser)

	// Create a request
	req := events.APIGatewayProxyRequest{
		Body: string(jsonData),
	}

	// Create Validate middleware
	handler := Validate[TestUserWithValidator]()(mockHandler)

	// Execute middleware
	resp, err := handler(context.Background(), req)

	// Assertion
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestUserWithCustomAll implements both interfaces explicitly
type TestUserWithCustomAll struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Age         int    `json:"age"`
	FromJSON    bool   // Track unmarshaler usage
	IsValidated bool   // Track validator usage
}

// UnmarshalRequest implements the RequestUnmarshaler interface
func (u *TestUserWithCustomAll) UnmarshalRequest(data []byte) error {
	// For test purposes, manually set the values
	u.Name = "Custom Name"
	u.Email = "custom@example.com"
	u.Age = 42
	u.FromJSON = false
	return nil
}

// Validate implements the Validator interface
func (u *TestUserWithCustomAll) Validate() error {
	u.IsValidated = true
	return nil
}

func TestValidate_WithBothCustomInterfaces(t *testing.T) {
	// Create a mock handler that returns the validated data
	handlerWithBoth := func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		data, ok := ctx.Value(CtxKey{}).(TestUserWithCustomAll)
		if !ok {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       "failed to get context value",
			}, nil
		}

		// Check that both interfaces were used
		if !data.IsValidated {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       "validator not called",
			}, nil
		}

		if data.FromJSON {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       "unmarshaler method not used",
			}, nil
		}

		responseBody, _ := json.Marshal(data)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Body:       string(responseBody),
		}, nil
	}

	// Create a request - content doesn't matter since we have a custom unmarshaler
	req := events.APIGatewayProxyRequest{
		Body: `{"name": "Jane Both", "email": "jane.both@example.com", "age": 32}`,
	}

	// Create Validate middleware
	handler := Validate[TestUserWithCustomAll]()(handlerWithBoth)

	// Execute middleware
	resp, err := handler(context.Background(), req)

	// Assertion
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify the data through the response
	var responseUser TestUserWithCustomAll
	err = json.Unmarshal([]byte(resp.Body), &responseUser)
	assert.NoError(t, err)
	assert.Equal(t, "Custom Name", responseUser.Name)
	assert.Equal(t, "custom@example.com", responseUser.Email)
	assert.Equal(t, 42, responseUser.Age)
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

func TestValidate_Base64EncodedJSONRequest(t *testing.T) {
	// Create valid user data
	validUser := TestUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}
	jsonData, _ := json.Marshal(validUser)

	// Base64 encode the JSON data
	base64Data := base64.StdEncoding.EncodeToString(jsonData)

	// Create a request with Base64 encoded body
	req := events.APIGatewayProxyRequest{
		Body:            base64Data,
		IsBase64Encoded: true,
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

func TestValidate_Base64EncodedXMLRequest(t *testing.T) {
	// Valid user data in XML format
	xmlData := []byte(`<TestUser><name>John Doe</name><email>john@example.com</email><age>30</age></TestUser>`)

	// Base64 encode the XML data
	base64Data := base64.StdEncoding.EncodeToString(xmlData)

	// Create a request with Base64 encoded body
	req := events.APIGatewayProxyRequest{
		Body:            base64Data,
		IsBase64Encoded: true,
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

func TestValidate_InvalidBase64EncodedRequest(t *testing.T) {
	// Invalid Base64 string
	invalidBase64 := "This is not a valid base64 string!!!"

	// Create a request with invalid Base64 encoded body
	req := events.APIGatewayProxyRequest{
		Body:            invalidBase64,
		IsBase64Encoded: true,
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
