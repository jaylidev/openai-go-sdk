# openai-go-sdk 设计文档

## 目标

面向中文互联网 AI Agent 开发者的 Go SDK，兼容 DeepSeek V4 Pro，提供 OpenAI 兼容的 API 调用能力 + Agent 场景高级抽象。

## 技术栈覆盖

- **LLM**: DeepSeek V4 Pro（现阶段唯一目标模型）
- **Agent 场景**: RAG、语义路由、渐进式披露、Prompt 缓存

## 项目结构

```
openai-go-sdk/
├── go.mod
├── client.go              # NewClient(WithXxx...)
├── config.go              # ClientConfig, Model 枚举, BaseURL 映射
├── chat.go                # ChatBuilder: 链式方法 + Do/Stream
├── chat_options.go        # WithTool, WithJSONSchema, WithThinking, WithCacheControl...
├── embedding.go           # EmbeddingBuilder
├── fim.go                 # FIMBuilder（预留+占位）
├── types.go               # Message, ToolCall, FunctionDef, Usage, ResponseFormat...
├── stream.go              # StreamReader[T] 通用 SSE
├── error.go               # APIError, ValidationError
│
├── internal/              # HTTP, JSON marshal, SSE parser（不对外暴露）
│
└── skill/
    ├── rag/               # RAG 检索增强生成
    ├── router/            # 语义路由 + 置信度阈值
    ├── disclosure/        # 渐进式披露 + 工具目录
    └── cache/             # Prompt 缓存策略
```

## Client 初始化

```go
client := openai.NewClient(
    openai.WithModel(openai.DeepSeekV4Pro),       // 必填
    openai.WithAPIKey("sk-xxx"),                  // 必填
    openai.WithCustomBaseURL("https://proxy.example.com/v1"), // 可选，覆盖默认 URL
    openai.WithHTTPClient(http.DefaultClient),     // 可选
    openai.WithMaxRetries(3),                     // 可选
)
```

### Model 枚举与 BaseURL 映射

SDK 内置模型枚举与多对一映射：

```go
type Model string
const (
    DeepSeekV4Pro    Model = "deepseek-v4-pro"
    DeepSeekReasoner Model = "deepseek-reasoner"
    DeepSeekChat     Model = "deepseek-chat"
)
```

默认 BaseURL: `https://api.deepseek.com`。可通过 `WithCustomBaseURL` 覆盖，支持代理/私有化部署。

## API 层设计

设计原则：**链式调用覆盖高频参数，Option 函数覆盖不定长/低频参数**。

### Chat Completions

**基础调用:**
```go
resp, err := client.Chat().
    Model(openai.DeepSeekV4Pro).           // 不传用全局默认
    SystemPrompt("你是中文AI助手").
    AddUserMsg("你好").
    Temperature(0.7).
    MaxTokens(4096).
    Do(ctx)
```

**SystemPrompt 多段拼接:**
```go
client.Chat().
    SystemPrompt("你是助手", "规则1: xxx", "规则2: xxx").  // 内部 \n\n 拼接
    AppendSystemPrompt(toolCatalog).                       // 追加工具目录
    AddUserMsg(msg)
```

**Tool Call (Option 不定长):**
```go
resp, err := client.Chat().
    AddUserMsg("查天气").
    Do(ctx,
        openai.WithTool(openai.Tool{...}),
        openai.WithToolChoice("auto"),
    )
```

**Structured Outputs:**
```go
resp, err := client.Chat().
    AddUserMsg(msg).
    Do(ctx, openai.WithJSONSchema("name", schema, true))
```

**Deep Think (Reasoning):**
```go
resp, err := client.Chat().
    AddUserMsg("证明费马大定理").
    Do(ctx, openai.WithThinking(true), openai.WithThinkingTokens(32000))
```

**Prompt 缓存:**
```go
resp, err := client.Chat().
    AddUserMsg(msg).
    Do(ctx, openai.WithCacheControl(openai.CacheBreakpoints{...}))
```

**Streaming:**
```go
stream := client.Chat().AddUserMsg(msg).Stream(ctx)
for stream.Next() {
    delta := stream.Delta()
    fmt.Print(delta.Content)
}
```

### Embeddings

```go
emb, err := client.Embedding().
    Model(openai.DeepSeekChat).
    Input([]string{"hello", "world"}).
    Do(ctx)
// emb.Data[0].Embedding → []float64
```

### FIM（Fill-in-the-Middle）

预留+占位，MVP 不实现，后续迭代补。

```go
resp, err := client.FIM().
    Model(openai.DeepSeekChat).
    Prefix("func fib(n int) int {").
    Suffix("}").
    Do(ctx)
```

### ToolCall 校验（SDK 内置）

Chat 返回的 `tool_call` 会自动校验：
- `name` 是否在已注册工具列表中
- `arguments` 是否符合对应 `FunctionDef.Parameters` 的 JSON Schema

不通过则返回 `ValidationError`，由 Agent 自行决策重试或降级。

## Skill 层设计

每个 Skill 拥有独立的 API 签名，不强行统一接口。各自通过构造函数注入 `*openai.Client`。

### Disclosure 渐进式披露（P0）

核心链路：**只发工具目录 → LLM 按需选取 → 再加载完整 schema → 最终调用**

```
用户消息 → Chat(目录) → LLM 返回工具名 → ResolveTools → Chat(完整schema) → ToolCall
```

API:
```go
catalog := disclosure.NewCatalog(
    disclosure.ToolRef{Name: "refund", Hint: "退款 — 参数: order_id, reason", Category: "order"},
    disclosure.ToolRef{Name: "get_weather", Hint: "查询天气 — 参数: city, date"},
)

result := disclosure.New(client, catalog).Chat(ctx, userMsg)
```

### Router 语义路由（P0）

Structured Outputs 强制 LLM 输出 `{intent, confidence}`，低于阈值路由到兜底处理器。

API:
```go
router := router.New(client, []Route{
    {Intent: "order", Handler: orderHandler},
    {Intent: "weather", Handler: weatherHandler},
    {Intent: "unknown", Handler: fallbackHandler},
})
router.WithThreshold(0.7) // 置信度阈值
router.Route(ctx, userMsg)
```

### RAG 检索增强生成（P1）

文档 → Embedding → 向量检索 → 上下文拼入 SystemPrompt → Chat。

API:
```go
ragSkill := rag.New(client, rag.WithRetriever(vecDBRetriever))
answer := ragSkill.Ask(ctx, "今天的新政策是什么？")
```

### Cache Prompt 缓存策略（P1）

管理 prompt 缓存的命中、失效与统计。

API:
```go
cacheSkill := cache.New(client)
cacheSkill.WithTTL(10 * time.Minute)
cacheSkill.WithPolicy(cache.LRU)
stats := cacheSkill.Stats() // 命中率统计
```

## LLM 幻觉应对

| 幻觉类型 | 手段 | 所在层 |
|---------|------|--------|
| 输出格式错误 | Structured Outputs strict mode | API |
| 编造工具/参数 | ToolCall 自动校验 | API |
| 编造事实 | RAG 上下文锚定 | Skill |
| 选错意图 | Router 置信度阈值 | Skill |
| 逻辑错误 | Reasoning 模式 | API |

## MVP 优先级

| 模块 | 状态 |
|------|------|
| Client + Config + Model 枚举 | MVP |
| Chat Completions (Do + Stream) | MVP |
| Tool Call + 校验 | MVP |
| Structured Outputs | MVP |
| Deep Think (Thinking) | MVP |
| Prompt Caching | MVP |
| Embeddings | MVP |
| FIM | 预留+占位 |
| Skill: disclosure | MVP |
| Skill: router | MVP |
| Skill: rag | P1 |
| Skill: cache | P1 |
