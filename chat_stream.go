package openai

import (
	"context"
	"encoding/json"

	"github.com/jaylidev/openai-go-sdk/internal"
)

// streamResponse 流式响应中的单条 SSE 事件
type streamResponse struct {
	Choices []streamResponseChoice `json:"choices"`
}

type streamResponseChoice struct {
	Index        int    `json:"index"`
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// ChatStream 聊天流式响应
type ChatStream struct {
	parser       *internal.SSEParser
	curDelta     Delta
	curFinish    string
	done         bool
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

	var resp streamResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		s.done = true
		return false
	}

	if len(resp.Choices) > 0 {
		s.curDelta = resp.Choices[0].Delta
		s.curFinish = resp.Choices[0].FinishReason
		return true
	}

	s.done = true
	return false
}

// Delta 返回当前 delta
func (s *ChatStream) Delta() Delta {
	return s.curDelta
}

// Close 关闭流
func (s *ChatStream) Close() error {
	return s.parser.Close()
}
