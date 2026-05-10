package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	openai "github.com/xxx/openai-go-sdk"
)

func TestRouter_Route(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openai.ChatCompletionResponse{
			Choices: []openai.Choice{{
				Message:      openai.Message{Role: openai.RoleAssistant, Content: `{"intent":"order","confidence":0.95}`},
				FinishReason: "stop",
			}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := openai.NewClient(
		openai.WithModel(openai.DeepSeekV4Pro),
		openai.WithAPIKey("sk-test"),
		openai.WithCustomBaseURL(srv.URL),
	)

	router := New(client, []Route{
		{Intent: "order", Handler: HandlerFunc(func(ctx context.Context, msg string) (string, error) {
			return "order handler: " + msg, nil
		})},
		{Intent: "weather", Handler: HandlerFunc(func(ctx context.Context, msg string) (string, error) {
			return "weather handler: " + msg, nil
		})},
	})

	result, err := router.Route(context.Background(), "我要退订单")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "order handler: 我要退订单" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestRouter_Threshold_Fallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openai.ChatCompletionResponse{
			Choices: []openai.Choice{{
				Message:      openai.Message{Role: openai.RoleAssistant, Content: `{"intent":"order","confidence":0.3}`},
				FinishReason: "stop",
			}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := openai.NewClient(
		openai.WithModel(openai.DeepSeekV4Pro),
		openai.WithAPIKey("sk-test"),
		openai.WithCustomBaseURL(srv.URL),
	)

	router := New(client, []Route{
		{Intent: "unknown", Handler: HandlerFunc(func(ctx context.Context, msg string) (string, error) {
			return "fallback: " + msg, nil
		})},
		{Intent: "order", Handler: HandlerFunc(func(ctx context.Context, msg string) (string, error) {
			return "order handler", nil
		})},
	})

	result, err := router.Route(context.Background(), "模糊的消息")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "fallback: 模糊的消息" {
		t.Errorf("unexpected result: %s", result)
	}
}
