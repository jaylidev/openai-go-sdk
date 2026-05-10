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
