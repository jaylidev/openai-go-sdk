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
