package middleware

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
)

// reqIDKey はコンテキスト内でリクエスト ID を格納するためのキーの型です。
// 非公開の型を使用することで、他のパッケージとのキー衝突を防ぎます。
type reqIDKey struct{}

// RequestID は、API Gateway のリクエストコンテキストからリクエスト ID を抽出し、
// Go の context.Context に設定するミドルウェアです。
// リクエスト ID が存在しない場合、空文字列が設定されます。
// 後続のハンドラやミドルウェアは GetReqID 関数を使用してコンテキストからリクエスト ID を取得できます。
func RequestID(next HandlerFunc) HandlerFunc {
	return func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		// APIGatewayProxyRequestContext からリクエスト ID を取得
		reqID := request.RequestContext.RequestID

		// 新しいコンテキストにリクエスト ID を設定
		// context.WithValue を使用して、キー reqIDKey{} に reqID を関連付けます。
		ctxWithReqID := context.WithValue(ctx, reqIDKey{}, reqID)

		// リクエスト ID が設定された新しいコンテキストで次のハンドラを呼び出す
		return next(ctxWithReqID, request)
	}
}

// GetReqID は context.Context から RequestID ミドルウェアによって設定されたリクエスト ID を取得します。
// リクエスト ID がコンテキストに存在しない場合、空文字列を返します。
func GetReqID(ctx context.Context) string {
	// context.Value を使用して、キー reqIDKey{} に関連付けられた値を取得
	if reqID, ok := ctx.Value(reqIDKey{}).(string); ok {
		return reqID
	}
	// キーが存在しない、または型が string でない場合は空文字列を返す
	return ""
}
