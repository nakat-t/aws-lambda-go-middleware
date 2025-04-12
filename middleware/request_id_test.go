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
			name:           "リクエストIDが存在する場合",
			inputRequestID: "test-request-id-123",
			expectedReqID:  "test-request-id-123",
		},
		{
			name:           "リクエストIDが存在しない場合 (空文字列)",
			inputRequestID: "",
			expectedReqID:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// アサーション用
			assert := assert.New(t)

			// テスト対象のリクエストを作成
			request := events.APIGatewayProxyRequest{
				RequestContext: events.APIGatewayProxyRequestContext{
					RequestID: tt.inputRequestID,
				},
			}

			// モックの最終ハンドラ
			// このハンドラ内で GetReqID を呼び出し、期待通りの値が取得できるか確認
			mockHandler := func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
				// コンテキストからリクエストIDを取得
				actualReqID := GetReqID(ctx)
				// 期待されるリクエストIDと一致するかアサート
				assert.Equal(tt.expectedReqID, actualReqID, "GetReqID should return the correct request ID")

				// 元のリクエストオブジェクトが変更されていないことを確認 (念のため)
				assert.Equal(request, req, "Request object should not be modified")

				// ダミーのレスポンスを返す
				return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
			}

			// RequestID ミドルウェアを適用
			handlerWithMiddleware := RequestID(mockHandler)

			// ミドルウェアが適用されたハンドラを実行
			response, err := handlerWithMiddleware(context.Background(), request)

			// エラーが発生しないこと、ステータスコードが OK であることを確認
			assert.NoError(err, "Handler should not return an error")
			assert.Equal(http.StatusOK, response.StatusCode, "Status code should be OK")
		})
	}
}

func TestGetReqID_ContextWithoutID(t *testing.T) {
	// アサーション用
	assert := assert.New(t)

	// RequestID ミドルウェアによって設定されていない空のコンテキスト
	ctx := context.Background()

	// リクエストIDが設定されていないコンテキストから GetReqID を呼び出す
	reqID := GetReqID(ctx)

	// 空文字列が返されることを期待
	assert.Empty(reqID, "GetReqID should return an empty string for context without request ID")
}
