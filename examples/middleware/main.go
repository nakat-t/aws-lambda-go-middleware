package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	mw "github.com/nakat-t/aws-lambda-go-middleware/middleware"
)

// mainHandler はリクエスト ID を取得し、それをボディに含めて返すシンプルなハンドラです。
func mainHandler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// RequestID ミドルウェアによって設定されたリクエスト ID を取得
	reqID := mw.GetReqID(ctx)

	log.Printf("Handler received request. RequestID: %s", reqID)

	// レスポンスボディを作成
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
	m1 := mw.RequestID
	m2 := mw.AllowContentType([]string{"application/json"}, mw.WithResponseBody("Only application/json is allowed"))

	// チェーンを最終ハンドラに適用
	wrappedHandler := mw.Use(mainHandler, m1, m2)

	// --- サンプルリクエストの実行 ---
	log.Println("--- Running Sample Request 1 (Allowed Content-Type) ---")
	sampleRequestAllowed := events.APIGatewayProxyRequest{
		HTTPMethod: "POST",
		Path:       "/test",
		Headers: map[string]string{
			"Content-Type": "application/json; charset=utf-8", // 許可されるタイプ
		},
		Body: `{"data": "sample"}`,
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: "sample-req-id-1", // サンプルのリクエストID
		},
	}

	// ハンドラを実行
	responseAllowed, errAllowed := wrappedHandler(context.Background(), sampleRequestAllowed)
	if errAllowed != nil {
		log.Printf("Error from handler (Allowed): %v", errAllowed)
	} else {
		log.Printf("Response (Allowed): StatusCode=%d, Body=%s", responseAllowed.StatusCode, responseAllowed.Body)
	}

	fmt.Println() // 区切り線

	log.Println("--- Running Sample Request 2 (Disallowed Content-Type) ---")
	sampleRequestDisallowed := events.APIGatewayProxyRequest{
		HTTPMethod: "POST",
		Path:       "/test",
		Headers: map[string]string{
			"Content-Type": "text/plain", // 許可されていないタイプ
		},
		Body: "plain text data",
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: "sample-req-id-2",
		},
	}

	// ハンドラを実行
	responseDisallowed, errDisallowed := wrappedHandler(context.Background(), sampleRequestDisallowed)
	if errDisallowed != nil {
		// AllowContentType はエラーを返さず、レスポンスで処理するため、通常ここには来ない
		log.Printf("Error from handler (Disallowed): %v", errDisallowed)
	} else {
		log.Printf("Response (Disallowed): StatusCode=%d, Body=%s", responseDisallowed.StatusCode, responseDisallowed.Body)
	}

	fmt.Println()

	log.Println("--- Running Sample Request 3 (Missing Content-Type) ---")
	sampleRequestMissing := events.APIGatewayProxyRequest{
		HTTPMethod: "POST", // POST なので Content-Type が期待される
		Path:       "/test",
		Headers:    map[string]string{}, // Content-Type なし
		Body:       `{"data": "sample"}`,
		RequestContext: events.APIGatewayProxyRequestContext{
			RequestID: "sample-req-id-3",
		},
	}

	// ハンドラを実行
	responseMissing, errMissing := wrappedHandler(context.Background(), sampleRequestMissing)
	if errMissing != nil {
		log.Printf("Error from handler (Missing): %v", errMissing)
	} else {
		log.Printf("Response (Missing): StatusCode=%d, Body=%s", responseMissing.StatusCode, responseMissing.Body)
	}
}
