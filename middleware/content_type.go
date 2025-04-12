package middleware

import (
	"context"
	"mime"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

const (
	// defaultUnsupportedMediaTypeBody は Content-Type が許可されていない場合のデフォルトのレスポンスボディです。
	defaultUnsupportedMediaTypeBody = "Unsupported Media Type"
)

// AllowContentTypeConfig は AllowContentType ミドルウェアの設定です。
type AllowContentTypeConfig struct {
	AllowedTypes []string
	ErrorBody    string
}

// AllowContentTypeOption は AllowContentType の設定を変更するための関数型です。
type AllowContentTypeOption func(*AllowContentTypeConfig)

// WithContentTypeErrorBody はエラー時のレスポンスボディを設定します。
func WithResponseBody(body string) AllowContentTypeOption {
	return func(c *AllowContentTypeConfig) {
		c.ErrorBody = body
	}
}

// AllowContentType はリクエストの Content-Type ヘッダーが指定されたリストに含まれているか検証するミドルウェアを作成します。
//
// Content-Type ヘッダーが存在しない場合、またはリストに含まれていないメディアタイプの場合、
// デフォルトではステータスコード 415 (Unsupported Media Type) と "Unsupported Media Type" というボディを持つレスポンスを返します。
// レスポンスボディは WithResponseBody オプションでカスタマイズ可能です。
//
// contentTypes リストが空の場合、すべての Content-Type を拒否します。
// Content-Type の比較は、メディアタイプ部分のみで行われ、パラメータ（例: charset=utf-8）は無視されます。
// 比較は大文字小文字を区別しません。
//
// 例:
// AllowContentType([]string{"application/json"}) は "application/json" および "application/json; charset=utf-8" を許可します。
// AllowContentType([]string{"application/json", "application/xml"}) は JSON と XML の両方を許可します。
func AllowContentType(contentTypes []string, opts ...AllowContentTypeOption) MiddlewareFunc {
	// デフォルト設定
	config := AllowContentTypeConfig{
		AllowedTypes: contentTypes,
		ErrorBody:    defaultUnsupportedMediaTypeBody,
	}
	// オプションを適用
	for _, opt := range opts {
		opt(&config)
	}

	// 許可する Content-Type を小文字に変換し、マップに格納
	allowedMap := make(map[string]struct{}, len(config.AllowedTypes))
	for _, ct := range config.AllowedTypes {
		mediaType, _, err := mime.ParseMediaType(strings.ToLower(ct))
		if err == nil {
			allowedMap[mediaType] = struct{}{}
		}
	}

	// エラーレスポンスを準備
	errorResponse := events.APIGatewayProxyResponse{
		StatusCode: http.StatusUnsupportedMediaType,
		Body:       config.ErrorBody,
		Headers:    map[string]string{"Content-Type": "text/plain; charset=utf-8"},
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			contentTypeHeader := request.Headers[http.CanonicalHeaderKey("Content-Type")]

			if contentTypeHeader == "" {
				// Content-Type が必須でない場合（GETなど）は許可する、という仕様も考えられるが、
				// chi の AllowContentType はヘッダーがない場合も拒否するため、それに合わせる。
				return errorResponse, nil
			}

			mediaType, _, err := mime.ParseMediaType(strings.ToLower(contentTypeHeader))
			if err != nil {
				// パース失敗時も拒否
				return errorResponse, nil
			}

			if _, ok := allowedMap[mediaType]; !ok {
				// 詳細なエラーメッセージが必要な場合はここで設定
				// errorResponse.Body = fmt.Sprintf("%s: Content-Type '%s' not allowed. Allowed: %v", config.ErrorBody, mediaType, config.AllowedTypes)
				return errorResponse, nil
			}

			return next(ctx, request)
		}
	}
}
