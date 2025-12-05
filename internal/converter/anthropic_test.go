package converter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestConvertAnthropicToCopilotMessages(t *testing.T) {
	tests := []struct {
		name         string
		messages     []map[string]interface{}
		system       string
		wantMsgCount int
		wantSystem   bool
	}{
		{
			name: "simple string content",
			messages: []map[string]interface{}{
				{"role": "user", "content": "Hello"},
				{"role": "assistant", "content": "Hi there"},
			},
			system:       "",
			wantMsgCount: 2,
			wantSystem:   false,
		},
		{
			name: "with system message",
			messages: []map[string]interface{}{
				{"role": "user", "content": "Hello"},
			},
			system:       "You are helpful",
			wantMsgCount: 2,
			wantSystem:   true,
		},
		{
			name: "content blocks with text",
			messages: []map[string]interface{}{
				{
					"role": "user",
					"content": []interface{}{
						map[string]interface{}{"type": "text", "text": "Hello"},
					},
				},
			},
			system:       "",
			wantMsgCount: 1,
			wantSystem:   false,
		},
		{
			name: "tool use in assistant message",
			messages: []map[string]interface{}{
				{
					"role": "assistant",
					"content": []interface{}{
						map[string]interface{}{"type": "text", "text": "Let me check"},
						map[string]interface{}{
							"type":  "tool_use",
							"id":    "tool_123",
							"name":  "get_weather",
							"input": map[string]interface{}{"location": "NYC"},
						},
					},
				},
			},
			system:       "",
			wantMsgCount: 1,
			wantSystem:   false,
		},
		{
			name: "tool result in user message",
			messages: []map[string]interface{}{
				{
					"role": "user",
					"content": []interface{}{
						map[string]interface{}{
							"type":        "tool_result",
							"tool_use_id": "tool_123",
							"content":     "Sunny, 72°F",
						},
					},
				},
			},
			system:       "",
			wantMsgCount: 1, // Tool results become tool messages
			wantSystem:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertAnthropicToCopilotMessages(tt.messages, tt.system)

			if len(result) != tt.wantMsgCount {
				t.Errorf("ConvertAnthropicToCopilotMessages() returned %d messages, want %d", len(result), tt.wantMsgCount)
			}

			if tt.wantSystem && (len(result) == 0 || result[0]["role"] != "system") {
				t.Error("Expected system message as first message")
			}
		})
	}
}

func TestExtractSystemText(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
		{
			name:     "string input",
			input:    "You are a helpful assistant",
			expected: "You are a helpful assistant",
		},
		{
			name: "text blocks",
			input: []interface{}{
				map[string]interface{}{"type": "text", "text": "First part"},
				map[string]interface{}{"type": "text", "text": "Second part"},
			},
			expected: "First part\nSecond part",
		},
		{
			name: "mixed content",
			input: []interface{}{
				map[string]interface{}{"type": "text", "text": "Important"},
				"Also this",
			},
			expected: "Important\nAlso this",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractSystemText(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractSystemText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConvertAnthropicTools(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected int // number of tools expected
	}{
		{
			name:     "empty tools",
			input:    []interface{}{},
			expected: 0,
		},
		{
			name: "valid tool",
			input: []interface{}{
				map[string]interface{}{
					"name":        "get_weather",
					"description": "Get weather info",
					"input_schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{"type": "string"},
						},
					},
				},
			},
			expected: 1,
		},
		{
			name: "tool without name",
			input: []interface{}{
				map[string]interface{}{
					"description": "Missing name",
				},
			},
			expected: 0,
		},
		{
			name: "multiple tools",
			input: []interface{}{
				map[string]interface{}{"name": "tool1"},
				map[string]interface{}{"name": "tool2"},
				map[string]interface{}{"name": "tool3"},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertAnthropicTools(tt.input)

			if tt.expected == 0 && result != nil {
				t.Errorf("Expected nil for %s, got %v", tt.name, result)
				return
			}

			if tt.expected > 0 && len(result) != tt.expected {
				t.Errorf("ConvertAnthropicTools() returned %d tools, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestConvertOpenAIResponseToAnthropic(t *testing.T) {
	tests := []struct {
		name         string
		input        map[string]interface{}
		model        string
		wantContent  bool
		wantToolUse  bool
		wantStopReason string
	}{
		{
			name: "simple text response",
			input: map[string]interface{}{
				"choices": []interface{}{
					map[string]interface{}{
						"message": map[string]interface{}{
							"content": "Hello there!",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 5,
				},
			},
			model:        "claude-sonnet-4",
			wantContent:  true,
			wantToolUse:  false,
			wantStopReason: "end_turn",
		},
		{
			name: "response with tool calls",
			input: map[string]interface{}{
				"choices": []interface{}{
					map[string]interface{}{
						"message": map[string]interface{}{
							"content": nil,
							"tool_calls": []interface{}{
								map[string]interface{}{
									"id":   "call_123",
									"type": "function",
									"function": map[string]interface{}{
										"name":      "get_weather",
										"arguments": `{"location": "NYC"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
				"usage": map[string]interface{}{},
			},
			model:        "claude-sonnet-4",
			wantContent:  false,
			wantToolUse:  true,
			wantStopReason: "tool_use",
		},
		{
			name: "max tokens finish",
			input: map[string]interface{}{
				"choices": []interface{}{
					map[string]interface{}{
						"message": map[string]interface{}{
							"content": "Truncated...",
						},
						"finish_reason": "length",
					},
				},
				"usage": map[string]interface{}{},
			},
			model:        "claude-sonnet-4",
			wantContent:  true,
			wantToolUse:  false,
			wantStopReason: "max_tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertOpenAIResponseToAnthropic(tt.input, tt.model)

			// Check required fields
			if result["type"] != "message" {
				t.Errorf("Expected type 'message', got %v", result["type"])
			}

			if result["role"] != "assistant" {
				t.Errorf("Expected role 'assistant', got %v", result["role"])
			}

			if result["model"] != tt.model {
				t.Errorf("Expected model %q, got %v", tt.model, result["model"])
			}

			if result["stop_reason"] != tt.wantStopReason {
				t.Errorf("Expected stop_reason %q, got %v", tt.wantStopReason, result["stop_reason"])
			}

			// Check content
			content, ok := result["content"].([]interface{})
			if !ok {
				t.Fatal("Expected content to be a slice")
			}

			hasText := false
			hasToolUse := false
			for _, c := range content {
				cMap := c.(map[string]interface{})
				if cMap["type"] == "text" {
					hasText = true
				}
				if cMap["type"] == "tool_use" {
					hasToolUse = true
				}
			}

			if tt.wantContent && !hasText {
				t.Error("Expected text content in response")
			}

			if tt.wantToolUse && !hasToolUse {
				t.Error("Expected tool_use in response")
			}

			// Check usage
			usage, ok := result["usage"].(map[string]interface{})
			if !ok {
				t.Fatal("Expected usage to be present")
			}

			if _, ok := usage["input_tokens"]; !ok {
				t.Error("Expected input_tokens in usage")
			}

			if _, ok := usage["output_tokens"]; !ok {
				t.Error("Expected output_tokens in usage")
			}
		})
	}
}

func TestConvertAnthropicToCopilotMessages_ImageContent(t *testing.T) {
	messages := []map[string]interface{}{
		{
			"role": "user",
			"content": []interface{}{
				map[string]interface{}{
					"type": "image",
					"source": map[string]interface{}{
						"type":       "base64",
						"media_type": "image/png",
						"data":       "iVBORw0KGgo=",
					},
				},
				map[string]interface{}{"type": "text", "text": "What is this?"},
			},
		},
	}

	result := ConvertAnthropicToCopilotMessages(messages, "")

	if len(result) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result))
	}

	content, ok := result[0]["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected content to be a slice")
	}

	hasImageURL := false
	for _, c := range content {
		if c["type"] == "image_url" {
			hasImageURL = true
			imageURL, ok := c["image_url"].(map[string]interface{})
			if !ok {
				t.Fatal("Expected image_url to be a map")
			}
			url, _ := imageURL["url"].(string)
			if !strings.HasPrefix(url, "data:image/png;base64,") {
				t.Errorf("Expected data URL, got %s", url)
			}
		}
	}

	if !hasImageURL {
		t.Error("Expected image to be converted to image_url")
	}
}

func TestConvertAnthropicTools_WithInputSchema(t *testing.T) {
	tools := []interface{}{
		map[string]interface{}{
			"name":        "calculate",
			"description": "Perform calculations",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"expression": map[string]interface{}{
						"type":        "string",
						"description": "Math expression",
					},
				},
				"required": []interface{}{"expression"},
			},
		},
	}

	result := ConvertAnthropicTools(tools)

	if len(result) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(result))
	}

	if result[0]["type"] != "function" {
		t.Errorf("Expected type 'function', got %v", result[0]["type"])
	}

	function, ok := result[0]["function"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected function to be a map")
	}

	if function["name"] != "calculate" {
		t.Errorf("Expected name 'calculate', got %v", function["name"])
	}

	params, ok := function["parameters"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected parameters to be present")
	}

	if params["type"] != "object" {
		t.Errorf("Expected parameters type 'object', got %v", params["type"])
	}
}

func TestConvertAnthropicToCopilotMessages_ToolResultContent(t *testing.T) {
	// Test tool result with complex content
	messages := []map[string]interface{}{
		{
			"role": "user",
			"content": []interface{}{
				map[string]interface{}{
					"type":        "tool_result",
					"tool_use_id": "tool_456",
					"content": []interface{}{
						map[string]interface{}{"type": "text", "text": "Result line 1"},
						map[string]interface{}{"type": "text", "text": "Result line 2"},
					},
				},
			},
		},
	}

	result := ConvertAnthropicToCopilotMessages(messages, "")

	if len(result) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result))
	}

	if result[0]["role"] != "tool" {
		t.Errorf("Expected role 'tool', got %v", result[0]["role"])
	}

	if result[0]["tool_call_id"] != "tool_456" {
		t.Errorf("Expected tool_call_id 'tool_456', got %v", result[0]["tool_call_id"])
	}

	content, ok := result[0]["content"].(string)
	if !ok {
		t.Fatal("Expected content to be a string")
	}

	if !strings.Contains(content, "Result line 1") {
		t.Error("Expected content to contain 'Result line 1'")
	}
}

func TestConvertOpenAIResponseToAnthropic_EmptyChoices(t *testing.T) {
	input := map[string]interface{}{
		"choices": []interface{}{},
		"usage":   map[string]interface{}{},
	}

	result := ConvertOpenAIResponseToAnthropic(input, "claude-sonnet-4")

	content, ok := result["content"].([]interface{})
	if !ok {
		t.Fatal("Expected content to be a slice")
	}

	if len(content) != 0 {
		t.Errorf("Expected empty content for empty choices, got %d items", len(content))
	}
}

func TestExtractTextContent(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "string input",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name: "text blocks",
			input: []interface{}{
				map[string]interface{}{"type": "text", "text": "Part 1"},
				map[string]interface{}{"type": "text", "text": "Part 2"},
			},
			expected: "Part 1\nPart 2",
		},
		{
			name: "string items",
			input: []interface{}{
				"First",
				"Second",
			},
			expected: "First\nSecond",
		},
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTextContent(tt.input)
			if result != tt.expected {
				t.Errorf("extractTextContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConvertAnthropicToCopilotMessages_AssistantWithToolCalls(t *testing.T) {
	messages := []map[string]interface{}{
		{
			"role": "assistant",
			"content": []interface{}{
				map[string]interface{}{
					"type":  "tool_use",
					"id":    "toolu_01",
					"name":  "search",
					"input": map[string]interface{}{"query": "test"},
				},
			},
		},
	}

	result := ConvertAnthropicToCopilotMessages(messages, "")

	if len(result) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result))
	}

	toolCalls, ok := result[0]["tool_calls"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected tool_calls to be present")
	}

	if len(toolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(toolCalls))
	}

	if toolCalls[0]["id"] != "toolu_01" {
		t.Errorf("Expected tool call id 'toolu_01', got %v", toolCalls[0]["id"])
	}

	function, ok := toolCalls[0]["function"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected function to be present")
	}

	if function["name"] != "search" {
		t.Errorf("Expected function name 'search', got %v", function["name"])
	}

	var args map[string]interface{}
	if argsStr, ok := function["arguments"].(string); ok {
		json.Unmarshal([]byte(argsStr), &args)
	}

	if args["query"] != "test" {
		t.Errorf("Expected query 'test', got %v", args["query"])
	}
}
