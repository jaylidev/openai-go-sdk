package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogger_Info(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ChatCompletionResponse{
			Choices: []Choice{{Message: Message{Role: RoleAssistant, Content: "ok"}, FinishReason: "stop"}},
		})
	}))
	defer srv.Close()

	core, recorded := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	client := NewClient(
		WithModel(DeepSeekV4Pro),
		WithAPIKey("sk-test"),
		WithCustomBaseURL(srv.URL),
		WithLogger(logger),
		WithLogLevel(LogLevelInfo),
	)

	_, err := client.Chat().AddUserMsg("hi").Do(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := recorded.All()
	if len(entries) == 0 {
		t.Fatal("expected at least 1 log entry")
	}

	entry := entries[0]
	if entry.Level != zap.InfoLevel {
		t.Errorf("expected Info level, got %s", entry.Level)
	}

	fields := entry.ContextMap()
	if fields["method"] != "POST" {
		t.Errorf("expected POST method, got %v", fields["method"])
	}
	if _, ok := fields["url"]; !ok {
		t.Error("expected url field")
	}
	if _, ok := fields["status"]; !ok {
		t.Error("expected status field")
	}
	if _, ok := fields["duration"]; !ok {
		t.Error("expected duration field")
	}
	// Info 级别不应有 req 字段
	if _, ok := fields["req"]; ok {
		t.Error("Info level should not have req field")
	}
	// Info 级别不应有 resp 字段
	if _, ok := fields["resp"]; ok {
		t.Error("Info level should not have resp field")
	}
}

func TestLogger_Debug(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ChatCompletionResponse{
			Choices: []Choice{{Message: Message{Role: RoleAssistant, Content: "ok"}, FinishReason: "stop"}},
		})
	}))
	defer srv.Close()

	core, recorded := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	client := NewClient(
		WithModel(DeepSeekV4Pro),
		WithAPIKey("sk-test"),
		WithCustomBaseURL(srv.URL),
		WithLogger(logger),
		WithLogLevel(LogLevelDebug),
	)

	_, err := client.Chat().AddUserMsg("hi").Do(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := recorded.All()
	if len(entries) < 2 {
		t.Fatalf("expected 2 log entries (req + resp), got %d", len(entries))
	}

	// 第一条是 Debug 级别的请求日志
	reqEntry := entries[0]
	reqFields := reqEntry.ContextMap()
	if _, ok := reqFields["req"]; !ok {
		t.Error("Debug level should have req field on request entry")
	}
	if _, ok := reqFields["url"]; !ok {
		t.Error("expected url field on request entry")
	}

	// 第二条是响应日志
	respEntry := entries[1]
	respFields := respEntry.ContextMap()
	if _, ok := respFields["resp"]; !ok {
		t.Error("Debug level should have resp field on response entry")
	}
	if respFields["status"] == nil {
		t.Error("expected status field on response entry")
	}
}

func TestLogger_Default(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ChatCompletionResponse{
			Choices: []Choice{{Message: Message{Role: RoleAssistant, Content: "ok"}, FinishReason: "stop"}},
		})
	}))
	defer srv.Close()

	// 不传 WithLogger — 应无日志输出、不 panic
	client := NewClient(
		WithModel(DeepSeekV4Pro),
		WithAPIKey("sk-test"),
		WithCustomBaseURL(srv.URL),
	)

	_, err := client.Chat().AddUserMsg("hi").Do(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 不 panic 即通过
}

func TestLogger_DefaultLevel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ChatCompletionResponse{
			Choices: []Choice{{Message: Message{Role: RoleAssistant, Content: "ok"}, FinishReason: "stop"}},
		})
	}))
	defer srv.Close()

	core, recorded := observer.New(zap.DebugLevel)
	logger := zap.New(core)

	client := NewClient(
		WithModel(DeepSeekV4Pro),
		WithAPIKey("sk-test"),
		WithCustomBaseURL(srv.URL),
		WithLogger(logger),
		// 不传 WithLogLevel — 默认应为 Info
	)

	_, err := client.Chat().AddUserMsg("hi").Do(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := recorded.All()
	if len(entries) == 0 {
		t.Fatal("expected at least 1 log entry (default Info level)")
	}
	entry := entries[0]
	// 默认 Info 级别不应有 req 字段
	if _, ok := entry.ContextMap()["req"]; ok {
		t.Error("default Info level should not have req field")
	}
}

func TestLogger_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{"message": "invalid api key"},
		})
	}))
	defer srv.Close()

	core, recorded := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	client := NewClient(
		WithModel(DeepSeekV4Pro),
		WithAPIKey("sk-bad"),
		WithCustomBaseURL(srv.URL),
		WithLogger(logger),
		WithLogLevel(LogLevelInfo),
	)

	_, err := client.Chat().AddUserMsg("hi").Do(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	entries := recorded.All()
	if len(entries) < 1 {
		t.Fatal("expected at least 1 log entry")
	}

	entry := entries[0]
	if entry.Level != zap.ErrorLevel {
		t.Errorf("expected Error level for error response, got %s", entry.Level)
	}
	fields := entry.ContextMap()
	if _, ok := fields["error"]; !ok {
		t.Error("expected error field for error response")
	}
}
