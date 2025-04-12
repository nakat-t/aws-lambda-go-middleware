# AWS Lambda Go Middleware

[![Go Reference](https://pkg.go.dev/badge/github.com/armai/aws-lambda-go-middleware/middleware.svg)](https://pkg.go.dev/github.com/armai/aws-lambda-go-middleware/middleware)
<!-- Add other badges like build status, code coverage, license etc. if applicable -->

`aws-lambda-go-middleware` は、AWS Lambda の Go ハンドラ (`events.APIGatewayProxyRequest` を扱う) に対して、`net/http` スタイルのミドルウェア機能を提供するライブラリです。リクエストの前処理、レスポンスの後処理、エラーハンドリングなどをモジュール化し、再利用可能なコンポーネントとしてハンドラに適用できます。

## インストール

```bash
go get github.com/nakat-t/aws-lambda-go-middleware/middleware
```

## コアコンセプト

### `HandlerFunc`

AWS Lambda の API Gateway Proxy 統合ハンドラのシグネチャを表す関数型です。ミドルウェアチェーンの最終的なターゲットとなります。

```go
type HandlerFunc func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)
```

### `MiddlewareFunc`

`HandlerFunc` を受け取り、新しい `HandlerFunc` を返す関数型です。これを実装する関数がミドルウェアになります。

```go
type MiddlewareFunc func(next HandlerFunc) HandlerFunc
```

### `Use`

ミドルウェアを `HandlerFunc` に適用するための関数です。

```go
// ハンドラ h にミドルウェア m1, m2, m3 を適用
wrappedHandler := middleware.Use(h, m1, m2, m3)
// 実行順: m1 -> m2 -> m3 -> h -> m3 -> m2 -> m1
```

## 使用方法

```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	mw "github.com/nakat-t/aws-lambda-go-middleware/middleware"
)

// 実際のビジネスロジック
func myHandler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	reqID := mw.GetReqID(ctx) // RequestID ミドルウェアから値を取得
	log.Printf("Processing request: %s", reqID)
	// ... business logic ...
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "Hello from Lambda!",
	}, nil
}

func main() {
    // リクエストIDをコンテキストに追加
	m1 := mw.RequestID
	// application/json のみを許可
	m2 := mw.AllowContentType([]string{"application/json"})

	// チェーンをハンドラに適用
	wrappedHandler := mw.Use(myHandler, m1, m2)

	// Lambda を開始
	lambda.Start(wrappedHandler)
}

```

## 提供されているミドルウェア

### `RequestID`

API Gateway リクエストコンテキストからリクエスト ID (`RequestContext.RequestID`) を抽出し、`context.Context` に設定します。後続のミドルウェアやハンドラは `GetReqID(ctx)` を使用してこの ID を取得できます。これはログやトレースに役立ちます。

**シグネチャ:**

```go
func RequestID(next HandlerFunc) HandlerFunc
func GetReqID(ctx context.Context) string
```

### `AllowContentType`

リクエストの `Content-Type` ヘッダーが、指定された許可リストに含まれているかを検証します。

**シグネチャ:**

```go
func AllowContentType(contentTypes []string, opts ...AllowContentTypeOption) MiddlewareFunc
```

**オプション:**

```go
// Content-Type が許可されていない場合に返すレスポンスボディをカスタマイズします。
func WithResponseBody(body string) AllowContentTypeOption
```

**比較ルール:**

*   メディアタイプ部分のみを比較します (例: `application/json` は `application/json; charset=utf-8` と一致)。
*   比較は大文字小文字を区別しません。
*   `Content-Type` ヘッダーが存在しない場合、または許可リストに含まれない場合は `415 Unsupported Media Type` を返します。

## サンプルコード

`RequestID` と `AllowContentType` の使用例を含む実行可能なサンプルコードは、リポジトリの `examples/middleware` ディレクトリを参照してください。

```bash
# リポジトリのルートディレクトリから実行
go run examples/middleware/main.go
```

## ライセンス

このプロジェクトは [LICENSE](LICENSE) ファイルで定義されているライセンスの下で公開されています。
