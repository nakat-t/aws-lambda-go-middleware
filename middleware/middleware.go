package middleware

import (
	"context"
	"errors"

	"github.com/aws/aws-lambda-go/events"
)

// HandlerFunc は AWS Lambda の APIGatewayProxy イベントハンドラの型を表します。
// これは、ミドルウェアチェーンの最終的なターゲットとなる関数です。
type HandlerFunc func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

// MiddlewareFunc は HandlerFunc をラップして新しい HandlerFunc を返すミドルウェアの型を表します。
// ミドルウェアはリクエストの前処理、レスポンスの後処理、またはエラーハンドリングを行うために使用されます。
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// Chain はミドルウェアのチェーンを構築し、最終的なハンドラに適用するための構造体です。
// ミドルウェアは追加された順序で実行されます（最初に追加されたものが最も外側）。
type Chain struct {
	middlewares []MiddlewareFunc
}

// NewChain は新しいミドルウェアチェーンを作成します。
// 引数として渡されたミドルウェアが初期のチェーンを構成します。
func NewChain(middlewares ...MiddlewareFunc) Chain {
	// スライスのコピーを作成して、元のスライスへの変更を防ぐ
	newMiddlewares := make([]MiddlewareFunc, len(middlewares))
	copy(newMiddlewares, middlewares)
	return Chain{middlewares: newMiddlewares}
}

// Then は既存のチェーンの最後に新しいミドルウェアを追加します。
// このメソッドは新しい Chain インスタンスを返し、元の Chain は変更されません。
func (c Chain) Then(mw MiddlewareFunc) Chain {
	newMiddlewares := make([]MiddlewareFunc, len(c.middlewares)+1)
	copy(newMiddlewares, c.middlewares)
	newMiddlewares[len(c.middlewares)] = mw
	return Chain{middlewares: newMiddlewares}
}

// HandlerFunc はミドルウェアチェーンの最後に最終的な HandlerFunc を適用し、
// すべてのミドルウェアが適用された HandlerFunc を返します。
// ミドルウェアは適用された順（最初に追加されたものが最も外側）に実行されます。
// final ハンドラが nil の場合、デフォルトのエラーを返すハンドラが使用されます。
func (c Chain) HandlerFunc(final HandlerFunc) HandlerFunc {
	if final == nil {
		// デフォルトのハンドラ
		final = func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			return events.APIGatewayProxyResponse{}, errors.New("no handler provided")
		}
	}

	// スライスの逆順から適用していくことで、最初に追加されたミドルウェアが最も外側になる
	// 例: NewChain(m1, m2).Then(m3).HandlerFunc(h) の場合、実行順は m1 -> m2 -> m3 -> h -> m3 -> m2 -> m1
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		final = c.middlewares[i](final)
	}
	return final
}

// Use は複数のミドルウェアを単一の HandlerFunc に適用するヘルパー関数です。
// Chain 構造体を使わずに、直接ミドルウェアを適用したい場合に便利です。
// ミドルウェアは渡された順序の逆から適用されるため、実行順序は引数の順序と同じになります。
// 例: Use(h, m1, m2, m3) の場合、実行順は m1 -> m2 -> m3 -> h -> m3 -> m2 -> m1
func Use(h HandlerFunc, middlewares ...MiddlewareFunc) HandlerFunc {
	return NewChain(middlewares...).HandlerFunc(h)
}
