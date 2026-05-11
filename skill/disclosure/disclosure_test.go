package disclosure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	openai "github.com/jaylidev/openai-go-sdk"
)

func TestCatalog_BuildSystemPrompt(t *testing.T) {
	catalog := NewCatalog(
		ToolRef{Name: "get_weather", Hint: "查询天气"},
		ToolRef{Name: "refund", Hint: "退款"},
	)

	result := catalog.BuildSystemPrompt("你是助手")
	expected := "你是助手\n\n可用工具（按需加载详细参数）：\n- get_weather: 查询天气\n- refund: 退款\n"
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestCatalog_ParseToolRequests(t *testing.T) {
	catalog := NewCatalog(
		ToolRef{Name: "get_weather", Hint: "查天气"},
		ToolRef{Name: "refund", Hint: "退款"},
	)

	names := catalog.ParseToolRequests("我需要用 refund 工具")
	if len(names) != 1 || names[0] != "refund" {
		t.Errorf("expected [refund], got %v", names)
	}
}

func TestCatalog_ResolveFullSchemas(t *testing.T) {
	tool := openai.Tool{Type: "function", Function: &openai.FunctionDef{Name: "refund", Description: "退款"}}
	catalog := NewCatalog(
		ToolRef{Name: "refund", Hint: "退款", Tool: tool},
	)

	tools := catalog.ResolveFullSchemas([]string{"refund"})
	if len(tools) != 1 {
		t.Errorf("expected 1, got %d", len(tools))
	}
	if tools[0].Function.Name != "refund" {
		t.Errorf("expected refund, got %s", tools[0].Function.Name)
	}
}

func TestCatalog_FilterByCategory(t *testing.T) {
	catalog := NewCatalog(
		ToolRef{Name: "refund", Hint: "退款", Category: "order"},
		ToolRef{Name: "get_weather", Hint: "查天气"},
	)

	filtered := catalog.FilterByCategory("order")
	if len(filtered.refs) != 1 || filtered.refs[0].Name != "refund" {
		t.Errorf("expected only refund in filtered catalog")
	}
}

func TestDisclosure_Chat(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var req struct {
			Tools []openai.Tool `json:"tools"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		if callCount == 1 {
			json.NewEncoder(w).Encode(openai.ChatCompletionResponse{
				Choices: []openai.Choice{{
					Message:      openai.Message{Role: openai.RoleAssistant, Content: "refund"},
					FinishReason: "stop",
				}},
			})
		} else {
			if len(req.Tools) != 1 || req.Tools[0].Function.Name != "refund" {
				t.Errorf("expected 1 tool 'refund', got %v", req.Tools)
			}
			json.NewEncoder(w).Encode(openai.ChatCompletionResponse{
				Choices: []openai.Choice{{
					Message:      openai.Message{Role: openai.RoleAssistant, Content: "退款已处理"},
					FinishReason: "stop",
				}},
			})
		}
	}))
	defer srv.Close()

	client := openai.NewClient(
		openai.WithModel(openai.DeepSeekV4Pro),
		openai.WithAPIKey("sk-test"),
		openai.WithCustomBaseURL(srv.URL),
	)

	refundTool := openai.Tool{
		Type: "function",
		Function: &openai.FunctionDef{
			Name:        "refund",
			Description: "退款",
			Parameters:  map[string]any{"type": "object"},
		},
	}

	catalog := NewCatalog(
		ToolRef{Name: "refund", Hint: "退款 — 参数: order_id, reason", Tool: refundTool},
	)

	d := New(client, catalog)
	resp, err := d.Chat(context.Background(), "我要退订单#1234")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Choices[0].Message.Content != "退款已处理" {
		t.Errorf("unexpected response: %s", resp.Choices[0].Message.Content)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}
