package validate

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"unicode"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator/v10"
	"github.com/nakat-t/aws-lambda-go-middleware/middleware"
)

const (
	// defaultErrorBody is the default response body when validation fails
	defaultErrorBody = "Bad Request: Validation Failed"

	// defaultErrorContentType is the default Content-Type when validation fails
	defaultErrorContentType = "text/plain; charset=utf-8"
)

// RequestUnmarshaler is an interface that allows custom unmarshaling from request body
type RequestUnmarshaler interface {
	UnmarshalRequest([]byte) error
}

// Validator is an interface that allows custom validation logic
type Validator interface {
	Validate() error
}

// CtxKey is the default key type for the validated request value stored in the context
type CtxKey struct{}

// Config is the configuration for the Validate middleware
type Config struct {
	ctxKey           any
	errorBody        string
	errorContentType string
}

// Option is a function type that modifies the Validate middleware settings
type Option func(*Config)

// WithCtxKey specifies the key for the validated request value to be set in the context
func WithCtxKey(ctxKey any) Option {
	return func(c *Config) {
		c.ctxKey = ctxKey
	}
}

// WithResponse customizes the Content-Type header and body of the response when a validation error occurs
func WithResponse(contentType string, body string) Option {
	return func(c *Config) {
		c.errorContentType = contentType
		c.errorBody = body
	}
}

// determineContentType examines the first non-whitespace character of the request body
// to determine whether it's JSON or XML.
// Returns "json" for JSON content, "xml" for XML content, or "unknown" if neither.
func determineContentType(body string) string {
	// Skip any leading whitespace
	// Check the first non-whitespace character
	for _, r := range body {
		if !unicode.IsSpace(r) {
			switch r {
			case '{', '[': // JSON typically starts with { or [
				return "json"
			case '<': // XML typically starts with <
				return "xml"
			default:
				return "unknown"
			}
		}
	}
	return "unknown"
}

// Validate creates a middleware that validates the request body as the specified type T
//
// The middleware performs the following processes:
// 1. If type T implements RequestUnmarshaler interface, it uses UnmarshalFromRequest method
// 2. Otherwise, it automatically detects if the request body is JSON or XML based on the first non-whitespace character:
//   - '{' or '[' for JSON (unmarshals using json.Unmarshal)
//   - '<' for XML (unmarshals using xml.Unmarshal)
//   - Other characters default to JSON
//
// 3. Performs validation of type T using validator/v10 (tags must be set)
// 4. Returns a 400 Bad Request error if validation fails
// 5. If validation succeeds, sets the value of type T in the context
//
// The key to set in the context defaults to CtxKey{}, but can be changed with the WithCtxKey option
// The response in case of an error can be customized with the WithResponse option
//
// Examples:
// ```
//
//	type User struct {
//	    Name  string `json:"name" validate:"required"`
//	    Email string `json:"email" validate:"required,email"`
//	    Age   int    `json:"age" validate:"gte=0,lte=130"`
//	}
//
// // Validates the request body as User type and sets the validated User object in the context
// Validate[User]()
//
// // Use a custom context key
// type UserKey string
// Validate[User](WithCtxKey(UserKey("user")))
//
// // Set a custom error response
// Validate[User](WithResponse("application/json", `{"error": "Validation failed"}`))
// ```
func Validate[T any](opts ...Option) middleware.MiddlewareFunc {
	// Default settings
	config := Config{
		ctxKey:           CtxKey{},
		errorBody:        defaultErrorBody,
		errorContentType: defaultErrorContentType,
	}
	// Apply options
	for _, opt := range opts {
		opt(&config)
	}

	// Prepare the response when a validation error occurs
	errorResponse := events.APIGatewayProxyResponse{
		StatusCode: http.StatusBadRequest,
		Body:       config.errorBody,
		Headers:    map[string]string{"Content-Type": config.errorContentType},
	}

	// Create a validator
	validate := validator.New(validator.WithRequiredStructEnabled())

	return func(next middleware.HandlerFunc) middleware.HandlerFunc {
		return func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			// There is an option to skip validation if the request body is empty,
			// but here, even if it is empty, it is treated as a validation error (because necessary validation is performed according to type T)
			if request.Body == "" {
				return errorResponse, nil
			}

			var data T
			var requestBody []byte

			// Handle base64 encoded body if needed
			if request.IsBase64Encoded {
				decodedBody, err := base64.StdEncoding.DecodeString(request.Body)
				if err != nil {
					return errorResponse, nil
				}
				requestBody = decodedBody
			} else {
				requestBody = []byte(request.Body)
			}

			// Check if type T implements RequestUnmarshaler interface
			var requestUnmarshaler RequestUnmarshaler
			dataPtr := any(&data)
			if value, ok := dataPtr.(RequestUnmarshaler); ok {
				requestUnmarshaler = value
				// Use the custom unmarshaler
				if err := requestUnmarshaler.UnmarshalRequest(requestBody); err != nil {
					return errorResponse, nil
				}
			} else {
				// Determine the content type from the first non-whitespace character
				contentType := determineContentType(string(requestBody))

				// Unmarshal the request body based on the content type
				switch contentType {
				case "json":
					if err := json.Unmarshal(requestBody, &data); err != nil {
						return errorResponse, nil
					}
				case "xml":
					if err := xml.Unmarshal(requestBody, &data); err != nil {
						return errorResponse, nil
					}
				default:
					// Default to JSON if content type cannot be determined
					if err := json.Unmarshal(requestBody, &data); err != nil {
						return errorResponse, nil
					}
				}
			}

			// Check if type T implements Validator interface
			var validator Validator
			dataPtr = any(&data)
			if value, ok := dataPtr.(Validator); ok {
				validator = value
				// Use the custom validator
				if err := validator.Validate(); err != nil {
					return errorResponse, nil
				}
			} else {
				// Execute validation
				if err := validate.Struct(data); err != nil {
					return errorResponse, nil
				}
			}

			// If validation succeeds, set the data in the context
			ctxWithData := context.WithValue(ctx, config.ctxKey, data)

			// Call the next handler with the new context containing the data
			return next(ctxWithData, request)
		}
	}
}
