package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatBuilder_Do(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("expected /v1/chat/completions, got %s", r.URL.Path)
		}

		var req ChatCompletionRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Model != "deepseek-v4-pro" {
			t.Errorf("expected deepseek-v4-pro, got %s", req.Model)
		}
		if len(req.Messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(req.Messages))
		}

		resp := ChatCompletionResponse{
			ID:    "chatcmpl-123",
			Model: "deepseek-v4-pro",
			Choices: []Choice{
				{
					Index:   0,
					Message: Message{Role: RoleAssistant, Content: "你好！Go 是一门静态类型编程语言。"},
					FinishReason: "stop",
				},
			},
			Usage: Usage{PromptTokens: 20, CompletionTokens: 15, TotalTokens: 35},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient(
		WithModel(DeepSeekV4Pro),
		WithAPIKey("sk-test"),
		WithCustomBaseURL(srv.URL),
	)

	resp, err := client.Chat().
		SystemPrompt("你是中文AI助手").
		AddUserMsg("你好").
		Temperature(0.7).
		MaxTokens(4096).
		Do(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Choices[0].Message.Content != "你好！Go 是一门静态类型编程语言。" {
		t.Errorf("unexpected content: %s", resp.Choices[0].Message.Content)
	}
	if resp.Usage.PromptTokens != 20 {
		t.Errorf("unexpected prompt tokens: %d", resp.Usage.PromptTokens)
	}
}

func TestChatBuilder_ModelOverride(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatCompletionRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Model != "deepseek-reasoner" {
			t.Errorf("expected deepseek-reasoner, got %s", req.Model)
		}
		json.NewEncoder(w).Encode(ChatCompletionResponse{
			Choices: []Choice{{Message: Message{Role: RoleAssistant, Content: "ok"}, FinishReason: "stop"}},
		})
	}))
	defer srv.Close()

	client := NewClient(
		WithModel(DeepSeekV4Pro),
		WithAPIKey("sk-test"),
		WithCustomBaseURL(srv.URL),
	)

	_, err := client.Chat().
		Model(DeepSeekReasoner).
		AddUserMsg("hi").
		Do(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChatBuilder_SystemPrompt(t *testing.T) {
	client := NewClient(WithModel(DeepSeekV4Pro), WithAPIKey("sk-test"), WithCustomBaseURL("http://localhost"))

	b := client.Chat().SystemPrompt("规则1", "规则2", "规则3")
	if len(b.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(b.messages))
	}
	if b.messages[0].Role != RoleSystem {
		t.Errorf("expected system role, got %s", b.messages[0].Role)
	}
	if b.messages[0].Content != "规则1\n\n规则2\n\n规则3" {
		t.Errorf("unexpected content: %s", b.messages[0].Content)
	}
}

func TestChatBuilder_AppendSystemPrompt(t *testing.T) {
	client := NewClient(WithModel(DeepSeekV4Pro), WithAPIKey("sk-test"), WithCustomBaseURL("http://localhost"))

	b := client.Chat().SystemPrompt("你是助手").AppendSystemPrompt("工具目录: get_weather")
	if len(b.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(b.messages))
	}
	expected := "你是助手\n\n工具目录: get_weather"
	if b.messages[0].Content != expected {
		t.Errorf("expected: %q, got: %q", expected, b.messages[0].Content)
	}
}

func TestChatBuilder_WithTool(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatCompletionRequest
		json.NewDecoder(r.Body).Decode(&req)
		if len(req.Tools) != 1 {
			t.Errorf("expected 1 tool, got %d", len(req.Tools))
		}
		if req.Tools[0].Function.Name != "get_weather" {
			t.Errorf("expected get_weather, got %s", req.Tools[0].Function.Name)
		}
		json.NewEncoder(w).Encode(ChatCompletionResponse{
			Choices: []Choice{{Message: Message{Role: RoleAssistant, Content: "ok"}, FinishReason: "stop"}},
		})
	}))
	defer srv.Close()

	client := NewClient(
		WithModel(DeepSeekV4Pro),
		WithAPIKey("sk-test"),
		WithCustomBaseURL(srv.URL),
	)

	_, err := client.Chat().
		AddUserMsg("查天气").
		Do(context.Background(),
			WithTool(Tool{
				Type: "function",
				Function: &FunctionDef{
					Name:        "get_weather",
					Description: "获取天气",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"city": map[string]string{"type": "string"},
						},
					},
				},
			}),
			WithToolChoice("auto"),
		)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChatBuilder_WithJSONSchema(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatCompletionRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.ResponseFormat.Type != "json_schema" {
			t.Errorf("expected json_schema, got %s", req.ResponseFormat.Type)
		}
		if !req.ResponseFormat.JSONSchema.Strict {
			t.Error("expected strict: true")
		}
		json.NewEncoder(w).Encode(ChatCompletionResponse{
			Choices: []Choice{{Message: Message{Role: RoleAssistant, Content: `{"answer":"ok"}`}, FinishReason: "stop"}},
		})
	}))
	defer srv.Close()

	client := NewClient(
		WithModel(DeepSeekV4Pro),
		WithAPIKey("sk-test"),
		WithCustomBaseURL(srv.URL),
	)

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"answer": map[string]string{"type": "string"},
		},
	}

	_, err := client.Chat().
		AddUserMsg("回答我").
		Do(context.Background(), WithJSONSchema("test", schema, true))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChatBuilder_WithThinking(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatCompletionRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Thinking == nil || req.Thinking.Type != "enabled" {
			t.Errorf("expected thinking enabled, got %v", req.Thinking)
		}
		json.NewEncoder(w).Encode(ChatCompletionResponse{
			Choices: []Choice{{Message: Message{Role: RoleAssistant, Content: "result"}, FinishReason: "stop"}},
		})
	}))
	defer srv.Close()

	client := NewClient(
		WithModel(DeepSeekV4Pro),
		WithAPIKey("sk-test"),
		WithCustomBaseURL(srv.URL),
	)

	_, err := client.Chat().
		AddUserMsg("复杂问题").
		Do(context.Background(), WithThinking(true))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
