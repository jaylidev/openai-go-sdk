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
