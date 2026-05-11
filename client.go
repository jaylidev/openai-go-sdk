package openai

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/jaylidev/openai-go-sdk/internal"
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

// ChatOption ChatBuilder.Do() 的运行时选项
type ChatOption func(*doOptions)

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

	httpClient := internal.NewHTTPClient(config.baseURL, config.apiKey, config.httpClient.Do, config.maxRetries)

	client := &Client{
		config: config,
		http:   httpClient,
	}

	if config.logger != nil {
		logLevel := config.logLevel

		httpClient.SetLogHooks(
			func(method, fullURL string, body []byte) {
				if logLevel >= LogLevelDebug {
					config.logger.Debug("",
						zap.String("method", method),
						zap.String("url", fullURL),
						zap.String("req", string(body)),
					)
				}
			},
			func(method, fullURL string, statusCode int, body []byte, dur time.Duration, err error) {
				fields := []zap.Field{
					zap.String("method", method),
					zap.String("url", fullURL),
					zap.Int("status", statusCode),
					zap.Duration("duration", dur),
				}
				if err != nil {
					fields = append(fields, zap.Error(err))
					config.logger.Error("", fields...)
					return
				}
				if logLevel >= LogLevelDebug {
					fields = append(fields, zap.String("resp", string(body)))
					config.logger.Debug("", fields...)
				} else {
					config.logger.Info("", fields...)
				}
			},
		)
	}

	return client
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
