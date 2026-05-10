# openai-go-sdk

面向中文互联网 AI Agent 开发者的 Go SDK，兼容 DeepSeek V4 Pro 的 OpenAI API。

## 安装

```bash
go get github.com/xxx/openai-go-sdk
```

## 快速开始

```go
package main

import (
    "context"
    "fmt"
    openai "github.com/xxx/openai-go-sdk"
)

func main() {
    client := openai.NewClient(
        openai.WithModel(openai.DeepSeekV4Pro),
        openai.WithAPIKey("sk-xxx"),
    )

    resp, err := client.Chat().
        SystemPrompt("你是中文AI助手").
        AddUserMsg("你好，介绍一下Go语言").
        Temperature(0.7).
        Do(context.Background())

    if err != nil {
        panic(err)
    }

    fmt.Println(resp.Choices[0].Message.Content)
}
```

## 功能

- **Chat Completions** — 对话补全，支持 Builder 链式调用
- **Streaming** — SSE 流式输出
- **Tool Call** — 函数调用 + 自动校验
- **Structured Outputs** — JSON Schema 强制输出
- **Deep Think** — 深度推理模式
- **Embeddings** — 文本向量嵌入
- **渐进式披露** — 工具目录 + 按需加载完整 Schema
- **语义路由** — 意图分类 + 置信度阈值兜底

## API

### 客户端初始化

```go
client := openai.NewClient(
    openai.WithModel(openai.DeepSeekV4Pro),          // 必填
    openai.WithAPIKey("sk-xxx"),                     // 必填
    openai.WithCustomBaseURL("https://your-proxy"),  // 可选
)
```

### Chat

```go
// 基础调用
resp, _ := client.Chat().
    SystemPrompt("你是中文AI助手").
    AddUserMsg("你好").
    Temperature(0.7).
    MaxTokens(4096).
    Do(ctx)

// Tool Call
resp, _ := client.Chat().
    AddUserMsg("查天气").
    Do(ctx,
        openai.WithTool(openai.Tool{
            Type: "function",
            Function: &openai.FunctionDef{
                Name:        "get_weather",
                Description: "获取城市天气",
                Parameters:  schema,
            },
        }),
        openai.WithToolChoice("auto"),
    )

// Structured Outputs
resp, _ := client.Chat().
    AddUserMsg("计算 8x + 7 = -23").
    Do(ctx, openai.WithJSONSchema("math_reasoning", schema, true))

// Deep Think
resp, _ := client.Chat().
    AddUserMsg("证明费马大定理").
    Do(ctx, openai.WithThinking(true))

// Streaming
stream, _ := client.Chat().AddUserMsg("讲个故事").Stream(ctx)
for stream.Next() {
    fmt.Print(stream.Delta().Content)
}
```

### Embedding

```go
emb, _ := client.Embedding().
    Input([]string{"你好", "世界"}).
    Do(ctx)
```

### 渐进式披露

```go
catalog := disclosure.NewCatalog(
    disclosure.ToolRef{Name: "refund", Hint: "退款 — 参数: order_id, reason", Tool: refundTool},
    disclosure.ToolRef{Name: "get_weather", Hint: "查询天气 — 参数: city, date", Tool: weatherTool},
)

d := disclosure.New(client, catalog)
resp, _ := d.Chat(ctx, "我要退订单#1234")
```

### 语义路由

```go
router := router.New(client, []router.Route{
    {Intent: "order", Handler: orderHandler},
    {Intent: "weather", Handler: weatherHandler},
    {Intent: "unknown", Handler: fallbackHandler},
})
router.WithThreshold(0.7)

result, _ := router.Route(ctx, "我要退订单")
```

## 模型

| 常量 | 模型名 |
|------|--------|
| `openai.DeepSeekV4Pro` | deepseek-v4-pro |
| `openai.DeepSeekReasoner` | deepseek-reasoner |
| `openai.DeepSeekChat` | deepseek-chat |

## License

Apache 2.0
