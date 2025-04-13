package middleware

import (
	"context"
	"reflect"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestChain_HandlerFunc_Order(t *testing.T) {
	var callOrder []string

	final := func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		callOrder = append(callOrder, "handler")
		return events.APIGatewayProxyResponse{Body: "ok"}, nil
	}

	mw1 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			callOrder = append(callOrder, "mw1_pre")
			resp, err := next(ctx, req)
			callOrder = append(callOrder, "mw1_post")
			return resp, err
		}
	}

	mw2 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			callOrder = append(callOrder, "mw2_pre")
			resp, err := next(ctx, req)
			callOrder = append(callOrder, "mw2_post")
			return resp, err
		}
	}

	chain := NewChain(mw1).Then(mw2)
	handler := chain.HandlerFunc(final)
	_, err := handler(context.Background(), events.APIGatewayProxyRequest{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	expected := []string{"mw1_pre", "mw2_pre", "handler", "mw2_post", "mw1_post"}
	if !reflect.DeepEqual(callOrder, expected) {
		t.Errorf("call order %v, expected %v", callOrder, expected)
	}
}

func TestUse_Function(t *testing.T) {
	var callOrder []string

	final := func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		callOrder = append(callOrder, "handler")
		return events.APIGatewayProxyResponse{Body: "ok"}, nil
	}

	mw := func(tag string) MiddlewareFunc {
		return func(next HandlerFunc) HandlerFunc {
			return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
				callOrder = append(callOrder, tag+"_pre")
				resp, err := next(ctx, req)
				callOrder = append(callOrder, tag+"_post")
				return resp, err
			}
		}
	}

	handler := Use(final, mw("mwA"), mw("mwB"))
	_, err := handler(context.Background(), events.APIGatewayProxyRequest{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	expected := []string{"mwA_pre", "mwB_pre", "handler", "mwB_post", "mwA_post"}
	if !reflect.DeepEqual(callOrder, expected) {
		t.Errorf("call order %v, expected %v", callOrder, expected)
	}
}
