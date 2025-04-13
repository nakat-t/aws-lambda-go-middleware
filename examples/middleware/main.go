package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/nakat-t/aws-lambda-go-middleware/middleware"
	"github.com/nakat-t/aws-lambda-go-middleware/middleware/contenttype"
	"github.com/nakat-t/aws-lambda-go-middleware/middleware/requestid"
)

// mainHandler is a simple handler that retrieves the request ID and includes it in the response body.
func mainHandler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Get the request ID set by the RequestID middleware
	reqID := ctx.Value(requestid.CtxKey{}).(string)

	log.Printf("Handler received request. RequestID: %s", reqID)

	// Create response body
	responseBody := map[string]string{
		"message":   "Request processed successfully",
		"requestID": reqID,
	}
	jsonBody, _ := json.Marshal(responseBody)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(jsonBody),
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}

func main() {
	m1 := requestid.RequestID()
	m2 := contenttype.AllowContentType([]string{"application/json"}, contenttype.WithResponse("application/json", "{\"error\":\"Only application/json is allowed\"}"))

	// Apply the chain to the final handler
	wrappedHandler := middleware.Use(mainHandler, m1, m2)

	// --- Running sample requests ---
	log.Println("--- Running Sample Request 1 (Allowed Content-Type) ---")
	sampleRequestAllowed := events.APIGatewayProxyRequest{
		HTTPMethod: "POST",
		Path:       "/test",
		Headers: map[string]string{
			"Content-Type": "application/json; charset=utf-8", // Allowed type
		},
		Body: `{"data": "sample"}`,
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: "sample-req-id-1", // Sample request ID
		},
	}

	// Execute the handler
	responseAllowed, errAllowed := wrappedHandler(context.Background(), sampleRequestAllowed)
	if errAllowed != nil {
		log.Printf("Error from handler (Allowed): %v", errAllowed)
	} else {
		log.Printf("Response (Allowed): StatusCode=%d, Body=%s", responseAllowed.StatusCode, responseAllowed.Body)
	}

	fmt.Println() // Separator line

	log.Println("--- Running Sample Request 2 (Disallowed Content-Type) ---")
	sampleRequestDisallowed := events.APIGatewayProxyRequest{
		HTTPMethod: "POST",
		Path:       "/test",
		Headers: map[string]string{
			"Content-Type": "text/plain", // Disallowed type
		},
		Body: "plain text data",
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: "sample-req-id-2",
		},
	}

	// Execute the handler
	responseDisallowed, errDisallowed := wrappedHandler(context.Background(), sampleRequestDisallowed)
	if errDisallowed != nil {
		// AllowContentType doesn't return an error but handles it in the response, so this typically doesn't happen
		log.Printf("Error from handler (Disallowed): %v", errDisallowed)
	} else {
		log.Printf("Response (Disallowed): StatusCode=%d, Body=%s", responseDisallowed.StatusCode, responseDisallowed.Body)
	}

	fmt.Println()

	log.Println("--- Running Sample Request 3 (Missing Content-Type) ---")
	sampleRequestMissing := events.APIGatewayProxyRequest{
		HTTPMethod: "POST", // POST, so Content-Type is expected
		Path:       "/test",
		Headers:    map[string]string{}, // No Content-Type
		Body:       `{"data": "sample"}`,
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: "sample-req-id-3",
		},
	}

	// Execute the handler
	responseMissing, errMissing := wrappedHandler(context.Background(), sampleRequestMissing)
	if errMissing != nil {
		log.Printf("Error from handler (Missing): %v", errMissing)
	} else {
		log.Printf("Response (Missing): StatusCode=%d, Body=%s", responseMissing.StatusCode, responseMissing.Body)
	}
}
