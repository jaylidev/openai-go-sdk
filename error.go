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
