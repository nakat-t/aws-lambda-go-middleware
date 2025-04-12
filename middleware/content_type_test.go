package middleware

import (
	"context"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

// mockNextHandler はテスト用の最終ハンドラです。
// 呼び出された場合は常に 200 OK を返します。
var mockNextHandler = func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: "OK"}, nil
}

// createRequest はテスト用の APIGatewayProxyRequest を作成します。
func createRequest(contentType string) events.APIGatewayProxyRequest {
	headers := make(map[string]string)
	if contentType != "" {
		// http.CanonicalHeaderKey を使って正規化されたキーで設定
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
		expectNextCalled    bool // next ハンドラが呼ばれることを期待するかどうか
	}{
		{
			name:                "許可された Content-Type (完全一致)",
			allowedContentTypes: []string{"application/json"},
			requestContentType:  "application/json",
			expectedStatusCode:  http.StatusOK,
			expectNextCalled:    true,
		},
		{
			name:                "許可された Content-Type (パラメータ付き)",
			allowedContentTypes: []string{"application/json"},
			requestContentType:  "application/json; charset=utf-8",
			expectedStatusCode:  http.StatusOK,
			expectNextCalled:    true,
		},
		{
			name:                "許可された Content-Type (大文字小文字区別なし)",
			allowedContentTypes: []string{"application/json"},
			requestContentType:  "Application/JSON",
			expectedStatusCode:  http.StatusOK,
			expectNextCalled:    true,
		},
		{
			name:                "許可されていない Content-Type",
			allowedContentTypes: []string{"application/json"},
			requestContentType:  "text/xml",
			expectedStatusCode:  http.StatusUnsupportedMediaType,
			expectNextCalled:    false,
		},
		{
			name:                "Content-Type ヘッダーなし",
			allowedContentTypes: []string{"application/json"},
			requestContentType:  "", // ヘッダーなしを示す
			expectedStatusCode:  http.StatusUnsupportedMediaType,
			expectNextCalled:    false,
		},
		{
			name:                "無効な Content-Type ヘッダー",
			allowedContentTypes: []string{"application/json"},
			requestContentType:  "invalid-content-type",
			expectedStatusCode:  http.StatusUnsupportedMediaType,
			expectNextCalled:    false,
		},
		{
			name:                "許可リストが空の場合",
			allowedContentTypes: []string{}, // 空のリスト
			requestContentType:  "application/json",
			expectedStatusCode:  http.StatusUnsupportedMediaType,
			expectNextCalled:    false,
		},
		{
			name:                "複数の許可タイプ (許可されるケース)",
			allowedContentTypes: []string{"application/json", "application/xml"},
			requestContentType:  "application/xml",
			expectedStatusCode:  http.StatusOK,
			expectNextCalled:    true,
		},
		{
			name:                "複数の許可タイプ (許可されないケース)",
			allowedContentTypes: []string{"application/json", "application/xml"},
			requestContentType:  "text/plain",
			expectedStatusCode:  http.StatusUnsupportedMediaType,
			expectNextCalled:    false,
		},
		{
			name:                "許可リストに無効なタイプが含まれる場合 (無視される)",
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

			// next ハンドラが呼ばれたかを記録するモック
			mockHandler := func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
				nextCalled = true
				return mockNextHandler(ctx, request)
			}

			// テスト対象のミドルウェアを作成
			middleware := AllowContentType(tt.allowedContentTypes)
			handlerWithMiddleware := middleware(mockHandler)

			// リクエストを作成
			request := createRequest(tt.requestContentType)

			// ハンドラを実行
			response, err := handlerWithMiddleware(context.Background(), request)

			// アサーション
			assert.NoError(err)
			assert.Equal(tt.expectedStatusCode, response.StatusCode)
			assert.Equal(tt.expectNextCalled, nextCalled, "Next handler call expectation mismatch")

			if !tt.expectNextCalled {
				// エラー時のデフォルトボディを確認
				assert.Contains(response.Body, defaultUnsupportedMediaTypeBody)
				assert.Equal("text/plain; charset=utf-8", response.Headers["Content-Type"])
			} else {
				assert.Equal("OK", response.Body) // next ハンドラが返したボディ
			}
		})
	}
}

func TestAllowContentType_WithOptions(t *testing.T) {
	assert := assert.New(t)
	customErrorBody := "Invalid Content Type Provided"
	nextCalled := false

	// next ハンドラが呼ばれたかを記録するモック
	mockHandler := func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		nextCalled = true
		return mockNextHandler(ctx, request)
	}

	// オプション付きでミドルウェアを作成
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
	// カスタムエラーボディを確認
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

	// オプションなしでミドルウェアを作成 (デフォルトでは何も許可しない)
	middleware := AllowContentType([]string{})
	handlerWithMiddleware := middleware(mockHandler)

	request := createRequest("application/json")
	response, err := handlerWithMiddleware(context.Background(), request)

	assert.NoError(err)
	assert.Equal(http.StatusUnsupportedMediaType, response.StatusCode)
	assert.False(nextCalled)
	// デフォルトのエラーボディを確認
	assert.Equal(defaultUnsupportedMediaTypeBody, response.Body)
}
