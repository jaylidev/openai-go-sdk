package openai

import (
	"net/http"

	"go.uber.org/zap"
)

// Model 模型枚举
type Model string

const (
	DeepSeekV4Pro    Model = "deepseek-v4-pro"
	DeepSeekV4Flash  Model = "deepseek-v4-flash"
	DeepSeekReasoner Model = "deepseek-reasoner"
	DeepSeekChat     Model = "deepseek-chat"
)

var defaultBaseURLs = map[Model]string{
	DeepSeekV4Pro:    "https://api.deepseek.com",
	DeepSeekV4Flash:  "https://api.deepseek.com",
	DeepSeekReasoner: "https://api.deepseek.com",
	DeepSeekChat:     "https://api.deepseek.com",
}

// HTTPDoer HTTP 客户端接口
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// LogLevel 日志级别
type LogLevel int

const (
	LogLevelInfo  LogLevel = iota
	LogLevelDebug
)

// ClientConfig 客户端配置（不对外暴露字段）
type ClientConfig struct {
	model      Model
	apiKey     string
	baseURL    string
	httpClient HTTPDoer
	maxRetries int
	logger     *zap.Logger
	logLevel   LogLevel
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

// WithLogger 设置 zap.Logger（可选，用于 debug 日志）
func WithLogger(logger *zap.Logger) ClientOption {
	return func(c *ClientConfig) {
		c.logger = logger
	}
}

// WithLogLevel 设置日志级别，默认 Info
func WithLogLevel(level LogLevel) ClientOption {
	return func(c *ClientConfig) {
		c.logLevel = level
	}
}
