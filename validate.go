package openai

import (
	"encoding/json"
	"fmt"
)

// ValidateToolCall 校验 LLM 返回的 tool_call 是否合法
func ValidateToolCall(registered map[string]*FunctionDef, call ToolCall) error {
	def, ok := registered[call.Function.Name]
	if !ok {
		return &ValidationError{
			ToolName: call.Function.Name,
			Reason:   "tool not registered",
		}
	}

	if def.Parameters == nil {
		return nil
	}

	if call.Function.Arguments == "" {
		return &ValidationError{
			ToolName: call.Function.Name,
			Reason:   "arguments is empty",
		}
	}

	var args any
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return &ValidationError{
			ToolName: call.Function.Name,
			Reason:   fmt.Sprintf("arguments is not valid JSON: %v", err),
		}
	}

	return nil
}

// RegisteredTools 将 Tool 列表转为 name→FunctionDef 的查找表
func RegisteredTools(tools []Tool) map[string]*FunctionDef {
	m := make(map[string]*FunctionDef, len(tools))
	for _, t := range tools {
		if t.Function != nil {
			m[t.Function.Name] = t.Function
		}
	}
	return m
}
