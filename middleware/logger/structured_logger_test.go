package logger

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/nakat-t/aws-lambda-go-middleware/middleware"
)

// testHandler is a simple handler for testing
type testHandler struct {
	resp events.APIGatewayProxyResponse
	err  error
}

func (h testHandler) Handle(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return h.resp, h.err
}

// testLogHandler is a custom slog.Handler that captures log records for testing
type testLogHandler struct {
	records []map[string]interface{}
}

func (h *testLogHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *testLogHandler) Handle(ctx context.Context, r slog.Record) error {
	m := make(map[string]interface{})
	m["level"] = r.Level.String()
	m["message"] = r.Message

	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value.Any()
		return true
	})

	h.records = append(h.records, m)
	return nil
}

func (h *testLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *testLogHandler) WithGroup(name string) slog.Handler {
	return h
}

func TestStructuredLogger_DefaultLogger(t *testing.T) {
	// Setup test handler and middleware with default logger (we can't easily test the default logger output)
	handler := testHandler{
		resp: events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Body:       "OK",
		},
		err: nil,
	}

	// The middleware function to test
	mw := StructuredLogger()

	// Apply middleware to handler
	wrappedHandler := mw(handler.Handle)

	// Execute handler
	req := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/test",
		Body:       "test body",
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: "test-request-id",
		},
	}

	resp, err := wrappedHandler(context.Background(), req)

	// Verify response passes through correctly
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if resp.Body != "OK" {
		t.Errorf("Expected body %q, got %q", "OK", resp.Body)
	}
}

func TestStructuredLogger_CustomLogger(t *testing.T) {
	// Setup test handler
	handler := testHandler{
		resp: events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Body:       "response body",
			Headers:    map[string]string{"Content-Type": "text/plain"},
		},
		err: nil,
	}

	// Setup custom logger with test handler
	logHandler := &testLogHandler{records: []map[string]interface{}{}}
	logger := slog.New(logHandler)

	// The middleware function to test with custom logger
	mw := StructuredLogger(WithLogger(logger))

	// Apply middleware to handler
	wrappedHandler := mw(handler.Handle)

	// Execute handler
	req := events.APIGatewayProxyRequest{
		HTTPMethod: "POST",
		Path:       "/test",
		Body:       "test request body",
		Headers:    map[string]string{"Content-Type": "application/json"},
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: "test-request-id",
		},
	}

	_, _ = wrappedHandler(context.Background(), req)

	// Verify logs
	if len(logHandler.records) != 2 {
		t.Fatalf("Expected 2 log records, got %d", len(logHandler.records))
	}

	// Check request log
	reqLog := logHandler.records[0]
	if reqLog["level"] != "INFO" {
		t.Errorf("Expected request log level INFO, got %s", reqLog["level"])
	}
	if reqLog["message"] != "request received" {
		t.Errorf("Expected request log message 'request received', got %q", reqLog["message"])
	}
	if reqBodySize, ok := reqLog["bodySize"].(int64); !ok || int(reqBodySize) != len(req.Body) {
		t.Errorf("Expected request bodySize %d, got %v (type: %T)", len(req.Body), reqLog["bodySize"], reqLog["bodySize"])
	}

	// Check response log
	respLog := logHandler.records[1]
	if respLog["level"] != "INFO" {
		t.Errorf("Expected response log level INFO, got %s", respLog["level"])
	}
	if respLog["message"] != "request processed successfully" {
		t.Errorf("Expected response log message 'request processed successfully', got %q", respLog["message"])
	}
	if respBodySize, ok := respLog["bodySize"].(int64); !ok || int(respBodySize) != len(handler.resp.Body) {
		t.Errorf("Expected response bodySize %d, got %v (type: %T)", len(handler.resp.Body), respLog["bodySize"], respLog["bodySize"])
	}
	if respLog["duration"] == nil {
		t.Errorf("Expected duration to be set, got nil")
	}
}

func TestStructuredLogger_WithError(t *testing.T) {
	// Setup test handler with error
	testError := fmt.Errorf("test error")
	handler := testHandler{
		resp: events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
		},
		err: testError,
	}

	// Setup custom logger with test handler
	logHandler := &testLogHandler{records: []map[string]interface{}{}}
	logger := slog.New(logHandler)

	// The middleware function to test
	mw := StructuredLogger(WithLogger(logger))

	// Apply middleware to handler
	wrappedHandler := mw(handler.Handle)

	// Execute handler
	req := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/error",
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: "error-request-id",
		},
	}

	_, err := wrappedHandler(context.Background(), req)

	// Verify error is passed through
	if err != testError {
		t.Errorf("Expected error %v, got %v", testError, err)
	}

	// Verify logs
	if len(logHandler.records) != 2 {
		t.Fatalf("Expected 2 log records, got %d", len(logHandler.records))
	}

	// Check response log with error
	respLog := logHandler.records[1]
	if respLog["level"] != "ERROR" {
		t.Errorf("Expected response log level ERROR, got %s", respLog["level"])
	}
	if respLog["message"] != "request processing failed" {
		t.Errorf("Expected response log message 'request processing failed', got %q", respLog["message"])
	}

	// Check error is logged
	loggedErr, ok := respLog["error"].(error)
	if !ok {
		t.Errorf("Expected error to be logged as error type")
	} else if loggedErr.Error() != testError.Error() {
		t.Errorf("Expected logged error %q, got %q", testError.Error(), loggedErr.Error())
	}
}

func TestStructuredLogger_Integration(t *testing.T) {
	// Integration test with middleware.Use function
	logHandler := &testLogHandler{records: []map[string]interface{}{}}
	logger := slog.New(logHandler)

	// Create a test handler
	handlerFunc := func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		// Simulate processing time
		time.Sleep(1 * time.Millisecond)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Body:       "Hello, " + req.Body,
		}, nil
	}

	// Use the middleware
	wrappedHandler := middleware.Use(handlerFunc, StructuredLogger(WithLogger(logger)))

	// Execute handler
	req := events.APIGatewayProxyRequest{
		HTTPMethod: "PUT",
		Path:       "/users/123",
		Body:       "World",
		Headers:    map[string]string{"Content-Type": "text/plain"},
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: "integration-test-id",
		},
	}

	resp, err := wrappedHandler(context.Background(), req)

	// Verify response
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if resp.Body != "Hello, World" {
		t.Errorf("Expected body %q, got %q", "Hello, World", resp.Body)
	}

	// Verify logs
	if len(logHandler.records) != 2 {
		t.Fatalf("Expected 2 log records, got %d", len(logHandler.records))
	}

	// Check duration is reasonable
	duration, ok := logHandler.records[1]["duration"].(time.Duration)
	if !ok {
		t.Errorf("Expected duration to be of type time.Duration")
	} else if duration < 1*time.Millisecond {
		t.Errorf("Expected duration to be at least 1ms, got %v", duration)
	}
}
