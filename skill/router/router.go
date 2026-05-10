package router

import (
	"context"
	"encoding/json"

	openai "github.com/xxx/openai-go-sdk"
)

// Route 路由定义
type Route struct {
	Intent  string
	Handler Handler
}

// Handler 路由处理器
type Handler interface {
	Handle(ctx context.Context, userMsg string) (string, error)
}

// HandlerFunc 函数式 Handler
type HandlerFunc func(ctx context.Context, userMsg string) (string, error)

func (f HandlerFunc) Handle(ctx context.Context, userMsg string) (string, error) {
	return f(ctx, userMsg)
}

// classificationResult LLM 分类输出
type classificationResult struct {
	Intent     string  `json:"intent"`
	Confidence float64 `json:"confidence"`
}

// Router 语义路由器
type Router struct {
	client    *openai.Client
	routes    []Route
	threshold float64
}

// New 创建路由器
func New(client *openai.Client, routes []Route) *Router {
	return &Router{
		client:    client,
		routes:    routes,
		threshold: 0.7,
	}
}

// WithThreshold 设置置信度阈值
func (r *Router) WithThreshold(t float64) *Router {
	r.threshold = t
	return r
}

// Route 路由用户消息
func (r *Router) Route(ctx context.Context, userMsg string) (string, error) {
	var intents []string
	for _, route := range r.routes {
		intents = append(intents, route.Intent)
	}

	// 用 Structured Outputs 强制输出 {intent, confidence}
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"intent":     map[string]any{"type": "string", "enum": intents},
			"confidence": map[string]string{"type": "number"},
		},
		"required": []string{"intent", "confidence"},
	}

	resp, err := r.client.Chat().
		SystemPrompt("你是一个语义路由器。根据用户消息判断意图并输出 JSON。用户消息: " + userMsg).
		Do(ctx, openai.WithJSONSchema("router_classification", schema, true))
	if err != nil {
		return "", err
	}

	var result classificationResult
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		return "", err
	}

	if result.Confidence < r.threshold {
		result.Intent = "unknown"
	}

	for _, route := range r.routes {
		if route.Intent == result.Intent {
			return route.Handler.Handle(ctx, userMsg)
		}
	}

	return "", nil
}
