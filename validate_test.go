package openai

import "testing"

func TestValidateToolCall_OK(t *testing.T) {
	registered := map[string]*FunctionDef{
		"get_weather": {
			Name: "get_weather",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city": map[string]string{"type": "string"},
				},
			},
		},
	}

	call := ToolCall{
		ID:   "call_1",
		Type: "function",
		Function: FunctionCall{
			Name:      "get_weather",
			Arguments: `{"city": "北京"}`,
		},
	}

	if err := ValidateToolCall(registered, call); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateToolCall_UnknownTool(t *testing.T) {
	registered := map[string]*FunctionDef{}

	call := ToolCall{
		ID:   "call_1",
		Type: "function",
		Function: FunctionCall{
			Name:      "hack_system",
			Arguments: `{}`,
		},
	}

	err := ValidateToolCall(registered, call)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	verr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if verr.ToolName != "hack_system" {
		t.Errorf("expected tool name 'hack_system', got %s", verr.ToolName)
	}
}

func TestValidateToolCall_InvalidJSON(t *testing.T) {
	registered := map[string]*FunctionDef{
		"get_weather": {Name: "get_weather", Parameters: map[string]any{"type": "object"}},
	}

	call := ToolCall{
		ID:   "call_1",
		Type: "function",
		Function: FunctionCall{
			Name:      "get_weather",
			Arguments: `not json`,
		},
	}

	err := ValidateToolCall(registered, call)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRegisteredTools(t *testing.T) {
	tools := []Tool{
		{Type: "function", Function: &FunctionDef{Name: "f1"}},
		{Type: "function", Function: &FunctionDef{Name: "f2"}},
	}

	m := RegisteredTools(tools)
	if len(m) != 2 {
		t.Errorf("expected 2, got %d", len(m))
	}
	if m["f1"] == nil || m["f2"] == nil {
		t.Error("expected both tools to be registered")
	}
}
