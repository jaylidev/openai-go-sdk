# openai-go-sdk MVP 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建面向 DeepSeek V4 Pro 的 OpenAI 兼容 Go SDK，含 Chat/Embedding API + disclosure/router Skill。

**Architecture:** 平铺 API 层（对标 go-openai）+ 独立 Skill 层。API 层使用 Builder 链式调用覆盖高频参数，Option 函数覆盖不定长参数。Skill 层各自独立签名，通过构造函数注入 Client。

**Tech Stack:** Go 1.21+, net/http, encoding/json, strings, bufio, sync

**MVP 范围:**
- P0: Client/Config, Chat(Do+Stream), Options(Tool/JSON/Thinking), Embedding, ToolCall 校验, Disclosure Skill, Router Skill
- P1 预留: FIM(占位), RAG Skill, Cache Skill

---

## 文件结构

```
openai-go-sdk/
├── go.mod                  # module github.com/xxx/openai-go-sdk
├── client.go               # Client, NewClient, option functions
├── config.go               # ClientConfig, Model enum, BaseURL mapping
├── types.go                # Message, Tool, Usage, Response types
├── chat.go                 # ChatBuilder.Do()
├── chat_stream.go          # ChatBuilder.Stream() + ChatStream
├── options.go              # WithTool, WithJSONSchema, WithThinking
├── embedding.go            # EmbeddingBuilder.Do()
├── fim.go                  # FIMBuilder (占位)
├── error.go                # APIError, ValidationError
├── validate.go             # ToolCall 校验
├── internal/
│   ├── http.go             # RequestBuilder + HTTP helper
│   └── sse.go              # SSE parser
└── skill/
    ├── disclosure/
    │   └── disclosure.go   # ToolCatalog, Chat()
    └── router/
        └── router.go       # Router, Route()
```

---

### Task 1: 初始化 Go Module

**Files:**
- Create: `go.mod`

- [ ] **Step 1: 初始化 go module**

```bash
cd /Volumes/ESXI-6_7_0-/openai-go-sdk && go mod init github.com/xxx/openai-go-sdk
```

模块名先用 `github.com/xxx/openai-go-sdk`，后续替换为实际路径。

- [ ] **Step 2: 验证**

```bash
cat go.mod
```

输出应包含 `module github.com/xxx/openai-go-sdk` 和当前 Go 版本。

- [ ] **Step 3: 提交**

```bash
git add go.mod && git commit -m "chore: init go module"
```

---

### Task 2: 基础类型定义

**Files:**
- Create: `types.go`

- [ ] **Step 1: 编写 types.go**

```go
package openai

// Role 消息角色
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message 对话消息
type Message struct {
	Role             Role       `json:"role"`
	Content          string     `json:"content,omitempty"`
	Name             string     `json:"name,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
}

func SystemMessage(content string) Message {
	return Message{Role: RoleSystem, Content: content}
}

func UserMessage(content string) Message {
	return Message{Role: RoleUser, Content: content}
}

func AssistantMessage(content string) Message {
	return Message{Role: RoleAssistant, Content: content}
}

func ToolMessage(toolCallID, content string) Message {
	return Message{Role: RoleTool, Content: content, ToolCallID: toolCallID}
}

// Tool 工具定义
type Tool struct {
	Type     string       `json:"type"`
	Function *FunctionDef `json:"function,omitempty"`
}

// FunctionDef 函数定义
type FunctionDef struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

// ToolCall LLM 返回的工具调用
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall 函数调用详情
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatCompletionRequest OpenAI 兼容的 Chat 请求体
type ChatCompletionRequest struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	Temperature    *float64        `json:"temperature,omitempty"`
	MaxTokens      *int            `json:"max_tokens,omitempty"`
	TopP           *float64        `json:"top_p,omitempty"`
	Stream         bool            `json:"stream,omitempty"`
	Stop           []string        `json:"stop,omitempty"`
	Tools          []Tool          `json:"tools,omitempty"`
	ToolChoice     any             `json:"tool_choice,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	Thinking       *ThinkingConfig `json:"thinking,omitempty"`
}

// ResponseFormat 输出格式控制
type ResponseFormat struct {
	Type       string            `json:"type"`
	JSONSchema *JSONSchemaConfig `json:"json_schema,omitempty"`
}

// JSONSchemaConfig JSON Schema 配置
type JSONSchemaConfig struct {
	Name   string `json:"name"`
	Schema any    `json:"schema"`
	Strict bool   `json:"strict,omitempty"`
}

// ThinkingConfig DeepSeek 推理模式配置
type ThinkingConfig struct {
	Type string `json:"type"`
}

// ChatCompletionResponse 非流式响应
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 非流式选项
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// StreamChoice 流式 SSE 事件的单条选项
type StreamChoice struct {
	Index        int    `json:"index"`
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// Delta 流式增量
type Delta struct {
	Role             string     `json:"role,omitempty"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

// Usage token 用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// EmbeddingRequest 嵌入请求
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingResponse 嵌入响应
type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  Usage           `json:"usage"`
}

// EmbeddingData 单个向量
type EmbeddingData struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}
```

- [ ] **Step 2: 验证编译**

```bash
go build ./...
```

- [ ] **Step 3: 提交**

```bash
git add types.go && git commit -m "feat: add core types"
```

---

### Task 3: 配置与模型枚举

**Files:**
- Create: `config.go`

- [ ] **Step 1: 编写 config.go**

```go
package openai

import "net/http"

// Model 模型枚举
type Model string

const (
	DeepSeekV4Pro    Model = "deepseek-v4-pro"
	DeepSeekReasoner Model = "deepseek-reasoner"
	DeepSeekChat     Model = "deepseek-chat"
)

var defaultBaseURLs = map[Model]string{
	DeepSeekV4Pro:    "https://api.deepseek.com",
	DeepSeekReasoner: "https://api.deepseek.com",
	DeepSeekChat:     "https://api.deepseek.com",
}

// HTTPDoer HTTP 客户端接口
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// ClientConfig 客户端配置（不对外暴露字段）
type ClientConfig struct {
	model      Model
	apiKey     string
	baseURL    string
	httpClient HTTPDoer
	maxRetries int
}

func (c ClientConfig) BaseURL() string      { return c.baseURL }
func (c ClientConfig) Model() Model         { return c.model }
func (c ClientConfig) APIKey() string       { return c.apiKey }
func (c ClientConfig) HTTPClient() HTTPDoer { return c.httpClient }
func (c ClientConfig) MaxRetries() int      { return c.maxRetries }

// ClientOption NewClient 的配置函数
type ClientOption func(*ClientConfig)

// WithModel 设置全局默认模型（必填）
func WithModel(m Model) ClientOption {
	return func(c *ClientConfig) {
		c.model = m
	}
}

// WithAPIKey 设置 API Key（必填）
func WithAPIKey(key string) ClientOption {
	return func(c *ClientConfig) {
		c.apiKey = key
	}
}

// WithCustomBaseURL 覆盖默认 BaseURL
func WithCustomBaseURL(url string) ClientOption {
	return func(c *ClientConfig) {
		c.baseURL = url
	}
}

// WithHTTPClient 设置自定义 HTTP 客户端
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *ClientConfig) {
		c.httpClient = client
	}
}

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(n int) ClientOption {
	return func(c *ClientConfig) {
		c.maxRetries = n
	}
}
```

- [ ] **Step 2: 验证编译**

```bash
go build ./...
```

- [ ] **Step 3: 提交**

```bash
git add config.go && git commit -m "feat: add config and model enum"
```

---

### Task 4: 错误类型

**Files:**
- Create: `error.go`

- [ ] **Step 1: 编写 error.go**

```go
package openai

import (
	"encoding/json"
	"fmt"
)

// APIError OpenAI 兼容的 API 错误
type APIError struct {
	Message        string `json:"message"`
	Type           string `json:"type"`
	Param          string `json:"param,omitempty"`
	Code           string `json:"code,omitempty"`
	HTTPStatusCode int    `json:"-"`
}

func (e *APIError) Error() string {
	msg := e.Message
	if e.Param != "" {
		msg = fmt.Sprintf("%s (param: %s)", msg, e.Param)
	}
	return fmt.Sprintf("api error: %s (status: %d)", msg, e.HTTPStatusCode)
}

// ValidationError ToolCall 校验失败
type ValidationError struct {
	ToolName string
	Reason   string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("tool call validation failed for %s: %s", e.ToolName, e.Reason)
}

// parseAPIError 从 HTTP 响应体解析 API 错误
func parseAPIError(statusCode int, body []byte) error {
	var apiErr APIError
	if json.Unmarshal(body, &apiErr) != nil {
		return &APIError{
			Message:        string(body),
			HTTPStatusCode: statusCode,
		}
	}
	apiErr.HTTPStatusCode = statusCode
	return &apiErr
}
```

- [ ] **Step 2: 验证编译**

```bash
go build ./...
```

- [ ] **Step 3: 提交**

```bash
git add error.go && git commit -m "feat: add error types"
```

---

### Task 5: 内部 HTTP 与 SSE 层

**Files:**
- Create: `internal/http.go`
- Create: `internal/sse.go`

- [ ] **Step 1: 编写 internal/http.go**

```go
package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// HTTPClient 封装 HTTP 请求
type HTTPClient struct {
	BaseURL    string
	APIKey     string
	Doer       func(*http.Request) (*http.Response, error)
	MaxRetries int
}

func NewHTTPClient(baseURL, apiKey string, doer func(*http.Request) (*http.Response, error), maxRetries int) *HTTPClient {
	if doer == nil {
		doer = http.DefaultClient.Do
	}
	return &HTTPClient{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		Doer:       doer,
		MaxRetries: maxRetries,
	}
}

func (c *HTTPClient) POST(ctx context.Context, path string, body any, result any) error {
	req, err := c.buildRequest(ctx, http.MethodPost, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	return c.do(req, result)
}

func (c *HTTPClient) POSTStream(ctx context.Context, path string, body any) (io.ReadCloser, error) {
	req, err := c.buildRequest(ctx, http.MethodPost, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.Doer(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, &HTTPError{StatusCode: resp.StatusCode, Status: resp.Status, Body: respBody}
	}
	return resp.Body, nil
}

func (c *HTTPClient) buildRequest(ctx context.Context, method, url string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *HTTPClient) do(req *http.Request, result any) error {
	resp, err := c.Doer(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return &HTTPError{StatusCode: resp.StatusCode, Status: resp.Status, Body: respBody}
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// HTTPError 内部 HTTP 错误
type HTTPError struct {
	StatusCode int
	Status     string
	Body       []byte
}

func (e *HTTPError) Error() string {
	return "http error: " + e.Status
}
```

- [ ] **Step 2: 编写 internal/sse.go**

```go
package internal

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

// SSEParser SSE 事件流解析器
type SSEParser struct {
	reader *bufio.Reader
	body   io.Closer
}

// NewSSEParser 创建 SSE 解析器
func NewSSEParser(body io.ReadCloser) *SSEParser {
	return &SSEParser{
		reader: bufio.NewReader(body),
		body:   body,
	}
}

// Next 读取下一个事件，返回 JSON 原始数据
func (p *SSEParser) Next() (json.RawMessage, error) {
	for {
		line, err := p.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return nil, io.EOF
			}
			return json.RawMessage(data), nil
		}
	}
}

// Close 关闭 SSE 流
func (p *SSEParser) Close() error {
	return p.body.Close()
}
```

- [ ] **Step 3: 验证编译**

```bash
go build ./...
```

- [ ] **Step 4: 提交**

```bash
git add internal/ && git commit -m "feat: add internal HTTP and SSE layer"
```

---

### Task 6: Client 创建

**Files:**
- Create: `client.go`

- [ ] **Step 1: 编写 client.go**

```go
package openai

import (
	"encoding/json"
	"net/http"

	"github.com/xxx/openai-go-sdk/internal"
)

// Client SDK 客户端
type Client struct {
	config ClientConfig
	http   *internal.HTTPClient
}

// doOptions ChatBuilder.Do() 的运行时 option 存储
type doOptions struct {
	tools        []Tool
	toolChoice   any
	responseFmt  *ResponseFormat
	thinking     *ThinkingConfig
	extraBody    map[string]any
	cacheControl any
}

// NewClient 创建客户端
func NewClient(opts ...ClientOption) *Client {
	config := ClientConfig{
		baseURL:    "https://api.deepseek.com",
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(&config)
	}

	if config.baseURL == "" {
		config.baseURL = "https://api.deepseek.com"
	}

	return &Client{
		config: config,
		http:   internal.NewHTTPClient(config.baseURL, config.apiKey, config.httpClient.Do, config.maxRetries),
	}
}

// buildChatReq 构建 Chat 请求对象，合并 builder 参数和 options
func (c *Client) buildChatReq(b *ChatBuilder, opts []ChatOption) (*ChatCompletionRequest, error) {
	model := string(c.config.model)
	if b.model != "" {
		model = string(b.model)
	}

	doOpts := &doOptions{}
	for _, opt := range opts {
		opt(doOpts)
	}

	req := &ChatCompletionRequest{
		Model:          model,
		Messages:       b.messages,
		Temperature:    b.temperature,
		MaxTokens:      b.maxTokens,
		TopP:           b.topP,
		Stop:           b.stop,
		Tools:          doOpts.tools,
		ToolChoice:     doOpts.toolChoice,
		ResponseFormat: doOpts.responseFmt,
		Thinking:       doOpts.thinking,
	}

	if doOpts.cacheControl != nil || len(doOpts.extraBody) > 0 {
		extra := make(map[string]any)
		if doOpts.cacheControl != nil {
			extra["cache_control"] = doOpts.cacheControl
		}
		for k, v := range doOpts.extraBody {
			extra[k] = v
		}
		mergeExtraBody(req, extra)
	}

	return req, nil
}

// handleAPIError 将内部 HTTP 错误转换为公开 APIError
func handleAPIError(err error) error {
	if httpErr, ok := err.(*internal.HTTPError); ok {
		return parseAPIError(httpErr.StatusCode, httpErr.Body)
	}
	return err
}

// mergeExtraBody 将 extraBody 合并到请求体（用于 cache_control 等扩展字段）
func mergeExtraBody(req *ChatCompletionRequest, extra map[string]any) {
	if extra == nil {
		return
	}
	raw, _ := json.Marshal(req)
	var merged map[string]any
	json.Unmarshal(raw, &merged)
	for k, v := range extra {
		merged[k] = v
	}
	raw, _ = json.Marshal(merged)
	json.Unmarshal(raw, req)
}
```

- [ ] **Step 2: 验证编译**

（client.go 引用了 ChatBuilder，需要在 Task 7 写完后才能编译，此步暂记）

```bash
go build ./...
```

- [ ] **Step 3: 提交**

和 Task 7 一起提交。

---

### Task 7: Chat Completions（非流式）

**Files:**
- Create: `chat.go`

- [ ] **Step 1: 编写 chat.go**

```go
package openai

import (
	"context"
	"strings"
)

// ChatBuilder 链式构建 Chat 请求
type ChatBuilder struct {
	client      *Client
	model       Model
	messages    []Message
	temperature *float64
	maxTokens   *int
	topP        *float64
	stop        []string
}

// Chat 创建 Chat Builder
func (c *Client) Chat() *ChatBuilder {
	return &ChatBuilder{client: c}
}

func (b *ChatBuilder) Model(m Model) *ChatBuilder {
	b.model = m
	return b
}

// SystemPrompt 设置 system prompt，多段用 \n\n 拼接
func (b *ChatBuilder) SystemPrompt(parts ...string) *ChatBuilder {
	b.messages = append(b.messages, SystemMessage(strings.Join(parts, "\n\n")))
	return b
}

// AddUserMsg 添加用户消息
func (b *ChatBuilder) AddUserMsg(content string) *ChatBuilder {
	b.messages = append(b.messages, UserMessage(content))
	return b
}

// AddAssistantMsg 添加助手消息（用于多轮对话）
func (b *ChatBuilder) AddAssistantMsg(content string) *ChatBuilder {
	b.messages = append(b.messages, AssistantMessage(content))
	return b
}

// AppendSystemPrompt 追加内容到已有的 system message 末尾
func (b *ChatBuilder) AppendSystemPrompt(content string) *ChatBuilder {
	if len(b.messages) > 0 && b.messages[len(b.messages)-1].Role == RoleSystem {
		b.messages[len(b.messages)-1].Content += "\n\n" + content
	} else {
		b.messages = append(b.messages, SystemMessage(content))
	}
	return b
}

// Messages 直接设置完整消息列表（高级用法）
func (b *ChatBuilder) Messages(msgs []Message) *ChatBuilder {
	b.messages = make([]Message, len(msgs))
	copy(b.messages, msgs)
	return b
}

func (b *ChatBuilder) Temperature(t float64) *ChatBuilder {
	b.temperature = &t
	return b
}

func (b *ChatBuilder) MaxTokens(n int) *ChatBuilder {
	b.maxTokens = &n
	return b
}

func (b *ChatBuilder) TopP(p float64) *ChatBuilder {
	b.topP = &p
	return b
}

func (b *ChatBuilder) Stop(words []string) *ChatBuilder {
	b.stop = words
	return b
}

// Do 执行非流式请求
func (b *ChatBuilder) Do(ctx context.Context, opts ...ChatOption) (*ChatCompletionResponse, error) {
	req, err := b.client.buildChatReq(b, opts)
	if err != nil {
		return nil, err
	}

	var resp ChatCompletionResponse
	err = b.client.http.POST(ctx, "/v1/chat/completions", req, &resp)
	if err != nil {
		return nil, handleAPIError(err)
	}
	return &resp, nil
}
```

- [ ] **Step 2: 编写测试**

Create: `chat_test.go`

```go
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
```

- [ ] **Step 3: 运行测试**

```bash
go test -v -run TestChatBuilder
```

期望: 全部 PASS

- [ ] **Step 4: 提交**

```bash
git add client.go chat.go chat_test.go && git commit -m "feat: add client and chat builder"
```

---

### Task 8: Chat Options（不定长参数）

**Files:**
- Create: `options.go`

- [ ] **Step 1: 编写 options.go**

```go
package openai

// ChatOption ChatBuilder.Do() 的不定长 option
type ChatOption func(*doOptions)

// WithTool 添加单个工具定义
func WithTool(tool Tool) ChatOption {
	return func(o *doOptions) {
		o.tools = append(o.tools, tool)
	}
}

// WithTools 批量添加工具定义
func WithTools(tools ...Tool) ChatOption {
	return func(o *doOptions) {
		o.tools = append(o.tools, tools...)
	}
}

// WithToolChoice 设置工具选择策略
func WithToolChoice(choice any) ChatOption {
	return func(o *doOptions) {
		o.toolChoice = choice
	}
}

// WithJSONSchema 设置 Structured Outputs
func WithJSONSchema(name string, schema any, strict bool) ChatOption {
	return func(o *doOptions) {
		o.responseFmt = &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &JSONSchemaConfig{
				Name:   name,
				Schema: schema,
				Strict: strict,
			},
		}
	}
}

// WithJSONMode 设置 JSON 模式（无 schema）
func WithJSONMode() ChatOption {
	return func(o *doOptions) {
		o.responseFmt = &ResponseFormat{Type: "json_object"}
	}
}

// WithThinking 开启/关闭深度推理
func WithThinking(enabled bool) ChatOption {
	t := "disabled"
	if enabled {
		t = "enabled"
	}
	return func(o *doOptions) {
		o.thinking = &ThinkingConfig{Type: t}
	}
}

// WithCacheControl 设置 Prompt 缓存控制（待 DeepSeek 支持原生缓存时启用）
func WithCacheControl(control any) ChatOption {
	return func(o *doOptions) {
		o.cacheControl = control
	}
}
```

- [ ] **Step 2: 编写 options 测试**

添加到 `chat_test.go`:

```go
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
```

- [ ] **Step 3: 运行测试**

```bash
go test -v -run TestChatBuilder
```

期望: 全部 PASS

- [ ] **Step 4: 提交**

```bash
git add options.go chat_test.go && git commit -m "feat: add chat options"
```

---

### Task 9: Chat Streaming

**Files:**
- Create: `chat_stream.go`

- [ ] **Step 1: 编写 chat_stream.go**

```go
package openai

import (
	"context"
	"encoding/json"
	"io"

	"github.com/xxx/openai-go-sdk/internal"
)

// ChatStream 聊天流式响应
type ChatStream struct {
	parser   *internal.SSEParser
	curDelta Delta
	done     bool
}

// Stream 执行流式请求
func (b *ChatBuilder) Stream(ctx context.Context) (*ChatStream, error) {
	req, err := b.client.buildChatReq(b, nil)
	if err != nil {
		return nil, err
	}
	req.Stream = true

	body, err := b.client.http.POSTStream(ctx, "/v1/chat/completions", req)
	if err != nil {
		return nil, handleAPIError(err)
	}

	return &ChatStream{parser: internal.NewSSEParser(body)}, nil
}

// Next 前进到下一个 delta，返回 false 表示流结束
func (s *ChatStream) Next() bool {
	if s.done {
		return false
	}

	raw, err := s.parser.Next()
	if err != nil {
		s.done = true
		return false
	}

	var choice StreamChoice
	if err := json.Unmarshal(raw, &choice); err != nil {
		s.done = true
		return false
	}

	s.curDelta = choice.Delta
	return true
}

// Delta 返回当前 delta
func (s *ChatStream) Delta() Delta {
	return s.curDelta
}

// Close 关闭流
func (s *ChatStream) Close() error {
	return s.parser.Close()
}
```

- [ ] **Step 2: 编写流式测试**

添加到 `chat_test.go`:

```go
func TestChatStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.Flusher")
		}

		events := []string{
			`{"choices":[{"index":0,"delta":{"role":"assistant","content":""}}]}`,
			`{"choices":[{"index":0,"delta":{"content":"你"}}]}`,
			`{"choices":[{"index":0,"delta":{"content":"好"}}]}`,
			`{"choices":[{"index":0,"delta":{"content":"！"},"finish_reason":"stop"}]}`,
		}

		for _, e := range events {
			w.Write([]byte("data: " + e + "\n\n"))
			flusher.Flush()
		}
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	client := NewClient(
		WithModel(DeepSeekV4Pro),
		WithAPIKey("sk-test"),
		WithCustomBaseURL(srv.URL),
	)

	stream, err := client.Chat().AddUserMsg("hi").Stream(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer stream.Close()

	var full string
	for stream.Next() {
		full += stream.Delta().Content
	}

	if full != "你好！" {
		t.Errorf("expected '你好！', got %q", full)
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
go test -v -run TestChatStream
```

期望: PASS

- [ ] **Step 4: 提交**

```bash
git add chat_stream.go chat_test.go && git commit -m "feat: add chat streaming"
```

---

### Task 10: ToolCall 校验

**Files:**
- Create: `validate.go`

- [ ] **Step 1: 编写 validate.go**

```go
package openai

import (
	"encoding/json"
	"fmt"
)

// ValidateToolCall 校验 LLM 返回的 tool_call 是否合法
func ValidateToolCall(registered map[string]*FunctionDef, call ToolCall) error {
	def, ok := registered[call.Function.Name]
	if !ok {
		return &ValidationError{
			ToolName: call.Function.Name,
			Reason:   "tool not registered",
		}
	}

	if def.Parameters == nil {
		return nil // 无 parameters schema，跳过参数校验
	}

	if call.Function.Arguments == "" {
		return &ValidationError{
			ToolName: call.Function.Name,
			Reason:   "arguments is empty",
		}
	}

	var args any
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return &ValidationError{
			ToolName: call.Function.Name,
			Reason:   fmt.Sprintf("arguments is not valid JSON: %v", err),
		}
	}

	return nil
}

// RegisteredTools 将 Tool 列表转为 name→FunctionDef 的查找表
func RegisteredTools(tools []Tool) map[string]*FunctionDef {
	m := make(map[string]*FunctionDef, len(tools))
	for _, t := range tools {
		if t.Function != nil {
			m[t.Function.Name] = t.Function
		}
	}
	return m
}
```

- [ ] **Step 2: 编写校验测试**

Create: `validate_test.go`

```go
package openai

import "testing"

func TestValidateToolCall_OK(t *testing.T) {
	registered := map[string]*FunctionDef{
		"get_weather": {
			Name: "get_weather",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city": map[string]string{"type": "string"},
				},
			},
		},
	}

	call := ToolCall{
		ID:   "call_1",
		Type: "function",
		Function: FunctionCall{
			Name:      "get_weather",
			Arguments: `{"city": "北京"}`,
		},
	}

	if err := ValidateToolCall(registered, call); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateToolCall_UnknownTool(t *testing.T) {
	registered := map[string]*FunctionDef{}

	call := ToolCall{
		ID:   "call_1",
		Type: "function",
		Function: FunctionCall{
			Name:      "hack_system",
			Arguments: `{}`,
		},
	}

	err := ValidateToolCall(registered, call)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	verr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if verr.ToolName != "hack_system" {
		t.Errorf("expected tool name 'hack_system', got %s", verr.ToolName)
	}
}

func TestValidateToolCall_InvalidJSON(t *testing.T) {
	registered := map[string]*FunctionDef{
		"get_weather": {Name: "get_weather", Parameters: map[string]any{"type": "object"}},
	}

	call := ToolCall{
		ID:   "call_1",
		Type: "function",
		Function: FunctionCall{
			Name:      "get_weather",
			Arguments: `not json`,
		},
	}

	err := ValidateToolCall(registered, call)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRegisteredTools(t *testing.T) {
	tools := []Tool{
		{Type: "function", Function: &FunctionDef{Name: "f1"}},
		{Type: "function", Function: &FunctionDef{Name: "f2"}},
	}

	m := RegisteredTools(tools)
	if len(m) != 2 {
		t.Errorf("expected 2, got %d", len(m))
	}
	if m["f1"] == nil || m["f2"] == nil {
		t.Error("expected both tools to be registered")
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
go test -v -run TestValidate
```

期望: 全部 PASS

- [ ] **Step 4: 提交**

```bash
git add validate.go validate_test.go && git commit -m "feat: add tool call validation"
```

---

### Task 11: Embeddings API

**Files:**
- Create: `embedding.go`

- [ ] **Step 1: 编写 embedding.go**

```go
package openai

import "context"

// EmbeddingBuilder 链式构建 Embedding 请求
type EmbeddingBuilder struct {
	client *Client
	model  Model
	input  []string
}

// Embedding 创建 Embedding Builder
func (c *Client) Embedding() *EmbeddingBuilder {
	return &EmbeddingBuilder{client: c}
}

func (b *EmbeddingBuilder) Model(m Model) *EmbeddingBuilder {
	b.model = m
	return b
}

func (b *EmbeddingBuilder) Input(input []string) *EmbeddingBuilder {
	b.input = input
	return b
}

// Do 执行嵌入请求
func (b *EmbeddingBuilder) Do(ctx context.Context) (*EmbeddingResponse, error) {
	model := string(b.client.config.model)
	if b.model != "" {
		model = string(b.model)
	}

	req := &EmbeddingRequest{
		Model: model,
		Input: b.input,
	}

	var resp EmbeddingResponse
	err := b.client.http.POST(ctx, "/v1/embeddings", req, &resp)
	if err != nil {
		return nil, handleAPIError(err)
	}
	return &resp, nil
}
```

- [ ] **Step 2: 编写测试**

Create: `embedding_test.go`

```go
package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbeddingBuilder_Do(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Errorf("expected /v1/embeddings, got %s", r.URL.Path)
		}
		resp := EmbeddingResponse{
			Object: "list",
			Data: []EmbeddingData{
				{Object: "embedding", Index: 0, Embedding: []float64{0.1, 0.2, 0.3}},
			},
			Model: "deepseek-chat",
			Usage: Usage{PromptTokens: 3, TotalTokens: 3},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient(
		WithModel(DeepSeekChat),
		WithAPIKey("sk-test"),
		WithCustomBaseURL(srv.URL),
	)

	resp, err := client.Embedding().
		Input([]string{"你好", "世界"}).
		Do(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 data, got %d", len(resp.Data))
	}
	if resp.Data[0].Embedding[0] != 0.1 {
		t.Errorf("unexpected first value: %f", resp.Data[0].Embedding[0])
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
go test -v -run TestEmbedding
```

期望: PASS

- [ ] **Step 4: 提交**

```bash
git add embedding.go embedding_test.go && git commit -m "feat: add embedding API"
```

---

### Task 12: FIM 占位

**Files:**
- Create: `fim.go`

- [ ] **Step 1: 编写 fim.go**

```go
package openai

import (
	"context"
	"fmt"
)

// FIMBuilder Fill-in-the-Middle 请求构建器（占位）
type FIMBuilder struct {
	client *Client
	model  Model
	prefix string
	suffix string
}

// FIM 创建 FIM Builder
func (c *Client) FIM() *FIMBuilder {
	return &FIMBuilder{client: c}
}

func (b *FIMBuilder) Model(m Model) *FIMBuilder {
	b.model = m
	return b
}

func (b *FIMBuilder) Prefix(p string) *FIMBuilder {
	b.prefix = p
	return b
}

func (b *FIMBuilder) Suffix(s string) *FIMBuilder {
	b.suffix = s
	return b
}

// Do 执行 FIM 请求（占位）
func (b *FIMBuilder) Do(ctx context.Context) (*ChatCompletionResponse, error) {
	return nil, fmt.Errorf("FIM not yet implemented")
}
```

- [ ] **Step 2: 验证编译**

```bash
go build ./...
```

- [ ] **Step 3: 提交**

```bash
git add fim.go && git commit -m "feat: add FIM placeholder"
```

---

### Task 13: Disclosure Skill（渐进式披露）

**Files:**
- Create: `skill/disclosure/disclosure.go`

- [ ] **Step 1: 创建目录**

```bash
mkdir -p skill/disclosure skill/router
```

- [ ] **Step 2: 编写 disclosure.go**

```go
package disclosure

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/xxx/openai-go-sdk"
)

// ToolRef 工具引用（目录中的条目，不含完整 schema）
type ToolRef struct {
	Name     string          `json:"name"`
	Hint     string          `json:"hint"`
	Category string          `json:"category"`
	Tool     openai.Tool     `json:"-"` // 完整 schema，按需加载
}

// Catalog 工具目录
type Catalog struct {
	refs []ToolRef
}

// NewCatalog 创建工具目录
func NewCatalog(refs ...ToolRef) *Catalog {
	return &Catalog{refs: refs}
}

// BuildSystemPrompt 生成工具目录文本
func (c *Catalog) BuildSystemPrompt(basePrompt string) string {
	var b strings.Builder
	b.WriteString(basePrompt)
	b.WriteString("\n\n可用工具（按需加载详细参数）：\n")
	for _, ref := range c.refs {
		b.WriteString(fmt.Sprintf("- %s: %s\n", ref.Name, ref.Hint))
	}
	return b.String()
}

// ParseToolRequests 从 LLM 响应中解析所需工具名
func (c *Catalog) ParseToolRequests(content string) []string {
	// 简单匹配：若 content 中包含工具名则选中
	var names []string
	for _, ref := range c.refs {
		if strings.Contains(content, ref.Name) {
			names = append(names, ref.Name)
		}
	}
	return names
}

// ResolveFullSchemas 按名称展开完整 Tool schema
func (c *Catalog) ResolveFullSchemas(names []string) []openai.Tool {
	var tools []openai.Tool
	for _, ref := range c.refs {
		for _, n := range names {
			if ref.Name == n {
				tools = append(tools, ref.Tool)
			}
		}
	}
	return tools
}

// FilterByCategory 按分类筛选工具引用
func (c *Catalog) FilterByCategory(category string) *Catalog {
	var filtered []ToolRef
	for _, ref := range c.refs {
		if ref.Category == category || ref.Category == "" {
			filtered = append(filtered, ref)
		}
	}
	return &Catalog{refs: filtered}
}

// Disclosure 渐进式披露 Agent
type Disclosure struct {
	client  *openai.Client
	catalog *Catalog
}

// New 创建 Disclosure
func New(client *openai.Client, catalog *Catalog) *Disclosure {
	return &Disclosure{client: client, catalog: catalog}
}

// Chat 执行两轮对话：目录 → 按需加载 → 完整调用
func (d *Disclosure) Chat(ctx context.Context, userMsg string, opts ...openai.ChatOption) (*openai.ChatCompletionResponse, error) {
	// 第1轮：只发工具目录
	sysPrompt := d.catalog.BuildSystemPrompt("你是AI助手，根据用户需求从工具目录中选择合适的工具。只需回复需要的工具名即可。")
	resp, err := d.client.Chat().
		SystemPrompt(sysPrompt).
		AddUserMsg(userMsg).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	content := resp.Choices[0].Message.Content

	// LLM 从目录中选了哪些工具
	wanted := d.catalog.ParseToolRequests(content)

	// 第2轮：加载完整 schema
	tools := d.catalog.ResolveFullSchemas(wanted)
	if len(tools) == 0 {
		// 没有匹配到工具，直接返回第1轮响应
		return resp, nil
	}

	allOpts := []openai.ChatOption{openai.WithTools(tools...), openai.WithToolChoice("auto")}
	allOpts = append(allOpts, opts...)

	return d.client.Chat().
		SystemPrompt("你是AI助手").
		AddUserMsg(userMsg).
		Do(ctx, allOpts...)
}
```

- [ ] **Step 3: 编写测试**

Create: `skill/disclosure/disclosure_test.go`

```go
package disclosure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	openai "github.com/xxx/openai-go-sdk"
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
			// 第1轮：LLM 回复工具名
			json.NewEncoder(w).Encode(openai.ChatCompletionResponse{
				Choices: []openai.Choice{{
					Message:      openai.Message{Role: openai.RoleAssistant, Content: "refund"},
					FinishReason: "stop",
				}},
			})
		} else {
			// 第2轮：包含完整 tool schema
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
			Parameters: map[string]any{"type": "object"},
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
```

- [ ] **Step 4: 运行测试**

```bash
go test -v ./skill/disclosure/...
```

期望: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add skill/disclosure/ && git commit -m "feat: add disclosure skill"
```

---

### Task 14: Router Skill（语义路由）

**Files:**
- Create: `skill/router/router.go`

- [ ] **Step 1: 编写 router.go**

```go
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

// classificationResult LLM 分类输出（Structured Outputs）
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
	// 构建意图列表
	var intents []string
	for _, route := range r.routes {
		intents = append(intents, route.Intent)
	}

	// 用 Structured Outputs 强制输出 {intent, confidence}
	intentPrompt := buildIntentPrompt(intents, userMsg)

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"intent":     map[string]any{"type": "string", "enum": intents},
			"confidence": map[string]string{"type": "number"},
		},
		"required": []string{"intent", "confidence"},
	}

	resp, err := r.client.Chat().
		SystemPrompt(intentPrompt).
		Do(ctx, openai.WithJSONSchema("router_classification", schema, true))
	if err != nil {
		return "", err
	}

	var result classificationResult
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		return "", err
	}

	// 置信度低于阈值，走兜底
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

func buildIntentPrompt(intents []string, userMsg string) string {
	return "你是一个语义路由器。根据用户消息判断意图并输出 JSON。用户消息: " + userMsg
}
```

- [ ] **Step 2: 编写测试**

Create: `skill/router/router_test.go`

```go
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
```

- [ ] **Step 3: 运行测试**

```bash
go test -v ./skill/router/...
```

期望: PASS

- [ ] **Step 4: 提交**

```bash
git add skill/router/ && git commit -m "feat: add router skill"
```

---

### Task 15: 全量测试与验证

**Files:**
- 无新文件

- [ ] **Step 1: 运行全部单元测试**

```bash
go test -v ./...
```

期望: 全部 PASS

- [ ] **Step 2: 检查 vet 和编译**

```bash
go vet ./...
go build ./...
```

期望: 无错误

- [ ] **Step 3: 最终提交**

```bash
git add -A
git status
git commit -m "chore: finalize MVP with all tests passing"
```
