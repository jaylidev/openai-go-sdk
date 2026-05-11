package disclosure

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/jaylidev/openai-go-sdk"
)

// ToolRef 工具引用（目录条目，不含完整 schema）
type ToolRef struct {
	Name     string      `json:"name"`
	Hint     string      `json:"hint"`
	Category string      `json:"category"`
	Tool     openai.Tool `json:"-"`
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
		if ref.Category == category {
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
	sysPrompt := d.catalog.BuildSystemPrompt("你是AI助手，根据用户需求从工具目录中选择合适的工具。只需回复需要的工具名即可。")
	resp, err := d.client.Chat().
		SystemPrompt(sysPrompt).
		AddUserMsg(userMsg).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	content := resp.Choices[0].Message.Content
	wanted := d.catalog.ParseToolRequests(content)
	tools := d.catalog.ResolveFullSchemas(wanted)
	if len(tools) == 0 {
		return resp, nil
	}

	allOpts := []openai.ChatOption{openai.WithTools(tools...), openai.WithToolChoice("auto")}
	allOpts = append(allOpts, opts...)

	return d.client.Chat().
		SystemPrompt("你是AI助手").
		AddUserMsg(userMsg).
		Do(ctx, allOpts...)
}
