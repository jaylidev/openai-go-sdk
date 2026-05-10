package openai

// WithTool 添加单个工具定义
func WithTool(tool Tool) ChatOption {
	return func(o *doOptions) {
		o.tools = append(o.tools, tool)
	}
}

// WithTools 批量添加工具定义
func WithTools(tools ...Tool) ChatOption {
	return func(o *doOptions) {
		o.tools = append(o.tools, tools...)
	}
}

// WithToolChoice 设置工具选择策略
func WithToolChoice(choice any) ChatOption {
	return func(o *doOptions) {
		o.toolChoice = choice
	}
}

// WithJSONSchema 设置 Structured Outputs
func WithJSONSchema(name string, schema any, strict bool) ChatOption {
	return func(o *doOptions) {
		o.responseFmt = &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &JSONSchemaConfig{
				Name:   name,
				Schema: schema,
				Strict: strict,
			},
		}
	}
}

// WithJSONMode 设置 JSON 模式（无 schema）
func WithJSONMode() ChatOption {
	return func(o *doOptions) {
		o.responseFmt = &ResponseFormat{Type: "json_object"}
	}
}

// WithThinking 开启/关闭深度推理
func WithThinking(enabled bool) ChatOption {
	t := "disabled"
	if enabled {
		t = "enabled"
	}
	return func(o *doOptions) {
		o.thinking = &ThinkingConfig{Type: t}
	}
}

// WithCacheControl 设置 Prompt 缓存控制
func WithCacheControl(control any) ChatOption {
	return func(o *doOptions) {
		o.cacheControl = control
	}
}
