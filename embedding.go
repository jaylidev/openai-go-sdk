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
	model := string(b.client.config.Model())
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
