package converter

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestConvertOpenAIToCopilotMessages(t *testing.T) {
	tests := []struct {
		name     string
		input    []map[string]interface{}
		expected []map[string]interface{}
	}{
		{
			name: "simple string content",
			input: []map[string]interface{}{
				{"role": "user", "content": "Hello"},
				{"role": "assistant", "content": "Hi there"},
			},
			expected: []map[string]interface{}{
				{"role": "user", "content": "Hello"},
				{"role": "assistant", "content": "Hi there"},
			},
		},
		{
			name: "nil content",
			input: []map[string]interface{}{
				{"role": "assistant", "content": nil, "tool_calls": []interface{}{}},
			},
			expected: []map[string]interface{}{
				{"role": "assistant", "content": nil, "tool_calls": []interface{}{}},
			},
		},
		{
			name: "multimodal content",
			input: []map[string]interface{}{
				{
					"role": "user",
					"content": []interface{}{
						map[string]interface{}{"type": "text", "text": "What's this?"},
						map[string]interface{}{"type": "image_url", "image_url": map[string]interface{}{"url": "http://example.com/img.png"}},
					},
				},
			},
			expected: []map[string]interface{}{
				{
					"role": "user",
					"content": []interface{}{
						map[string]interface{}{"type": "text", "text": "What's this?"},
						map[string]interface{}{"type": "image_url", "image_url": map[string]interface{}{"url": "http://example.com/img.png"}},
					},
				},
			},
		},
		{
			name: "tool call id",
			input: []map[string]interface{}{
				{"role": "tool", "content": "result", "tool_call_id": "call_123"},
			},
			expected: []map[string]interface{}{
				{"role": "tool", "content": "result", "tool_call_id": "call_123"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertOpenAIToCopilotMessages(tt.input)

			// Compare JSON representations for deep equality
			expectedJSON, _ := json.Marshal(tt.expected)
			resultJSON, _ := json.Marshal(result)

			if string(expectedJSON) != string(resultJSON) {
				t.Errorf("ConvertOpenAIToCopilotMessages() = %s, want %s", string(resultJSON), string(expectedJSON))
			}
		})
	}
}

func TestConvertOpenAITools(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected []map[string]interface{}
	}{
		{
			name:     "empty tools",
			input:    []interface{}{},
			expected: nil,
		},
		{
			name: "valid function tool",
			input: []interface{}{
				map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name":        "get_weather",
						"description": "Get weather",
						"parameters":  map[string]interface{}{"type": "object"},
					},
				},
			},
			expected: []map[string]interface{}{
				{
					"type": "function",
					"function": map[string]interface{}{
						"name":        "get_weather",
						"description": "Get weather",
						"parameters":  map[string]interface{}{"type": "object"},
					},
				},
			},
		},
		{
			name: "skip non-function tools",
			input: []interface{}{
				map[string]interface{}{"type": "code_interpreter"},
				map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name": "valid_func",
					},
				},
			},
			expected: []map[string]interface{}{
				{
					"type": "function",
					"function": map[string]interface{}{
						"name":        "valid_func",
						"description": nil,
						"parameters":  nil,
					},
				},
			},
		},
		{
			name: "skip function without name",
			input: []interface{}{
				map[string]interface{}{
					"type":     "function",
					"function": map[string]interface{}{},
				},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertOpenAITools(tt.input)

			if tt.expected == nil && result != nil {
				t.Errorf("ConvertOpenAITools() = %v, want nil", result)
				return
			}

			if tt.expected != nil && result == nil {
				t.Errorf("ConvertOpenAITools() = nil, want %v", tt.expected)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("ConvertOpenAITools() returned %d tools, want %d", len(result), len(tt.expected))
			}
		})
	}
}

func TestNormalizeContentForCopilot(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "nil content",
			input:    nil,
			expected: "",
		},
		{
			name:     "string content",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name: "input_text type",
			input: []interface{}{
				map[string]interface{}{"type": "input_text", "text": "Hello"},
			},
			expected: "Hello",
		},
		{
			name: "output_text type",
			input: []interface{}{
				map[string]interface{}{"type": "output_text", "text": "Response"},
			},
			expected: "Response",
		},
		{
			name: "text type",
			input: []interface{}{
				map[string]interface{}{"type": "text", "text": "Plain text"},
			},
			expected: "Plain text",
		},
		{
			name: "image_url passthrough",
			input: []interface{}{
				map[string]interface{}{"type": "image_url", "image_url": map[string]interface{}{"url": "http://example.com/img.png"}},
			},
			expected: []map[string]interface{}{
				{"type": "image_url", "image_url": map[string]interface{}{"url": "http://example.com/img.png"}},
			},
		},
		{
			name: "multiple items",
			input: []interface{}{
				map[string]interface{}{"type": "text", "text": "First"},
				map[string]interface{}{"type": "text", "text": "Second"},
			},
			expected: []map[string]interface{}{
				{"type": "text", "text": "First"},
				{"type": "text", "text": "Second"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeContentForCopilot(tt.input)

			expectedJSON, _ := json.Marshal(tt.expected)
			resultJSON, _ := json.Marshal(result)

			if string(expectedJSON) != string(resultJSON) {
				t.Errorf("NormalizeContentForCopilot() = %s, want %s", string(resultJSON), string(expectedJSON))
			}
		})
	}
}

func TestConvertResponsesInputToMessages(t *testing.T) {
	tests := []struct {
		name         string
		input        interface{}
		instructions string
		wantSystem   bool
		wantMsgCount int
	}{
		{
			name:         "simple string input",
			input:        "Hello",
			instructions: "",
			wantSystem:   false,
			wantMsgCount: 1,
		},
		{
			name:         "string input with instructions",
			input:        "Hello",
			instructions: "You are helpful",
			wantSystem:   true,
			wantMsgCount: 2,
		},
		{
			name: "list of messages",
			input: []interface{}{
				map[string]interface{}{"type": "message", "role": "user", "content": "Hi"},
				map[string]interface{}{"type": "message", "role": "assistant", "content": "Hello"},
			},
			instructions: "",
			wantSystem:   false,
			wantMsgCount: 2,
		},
		{
			name: "input_text and output_text",
			input: []interface{}{
				map[string]interface{}{"type": "input_text", "text": "User message"},
				map[string]interface{}{"type": "output_text", "text": "Assistant response"},
			},
			instructions: "",
			wantSystem:   false,
			wantMsgCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertResponsesInputToMessages(tt.input, tt.instructions)

			if len(result) != tt.wantMsgCount {
				t.Errorf("ConvertResponsesInputToMessages() returned %d messages, want %d", len(result), tt.wantMsgCount)
			}

			if tt.wantSystem && (len(result) == 0 || result[0]["role"] != "system") {
				t.Error("Expected system message as first message")
			}
		})
	}
}

func TestConvertOpenAIToCopilotMessages_EmptyInput(t *testing.T) {
	result := ConvertOpenAIToCopilotMessages(nil)
	if result == nil {
		result = []map[string]interface{}{}
	}
	if len(result) != 0 {
		t.Errorf("Expected empty result for nil input, got %d messages", len(result))
	}

	result = ConvertOpenAIToCopilotMessages([]map[string]interface{}{})
	if len(result) != 0 {
		t.Errorf("Expected empty result for empty input, got %d messages", len(result))
	}
}

func TestNormalizeContentForCopilot_Base64Image(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{
			"type": "image",
			"source": map[string]interface{}{
				"type":       "base64",
				"media_type": "image/jpeg",
				"data":       "base64data",
			},
		},
	}

	result := NormalizeContentForCopilot(input)
	resultSlice, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatal("Expected slice result")
	}

	if len(resultSlice) != 1 {
		t.Errorf("Expected 1 item, got %d", len(resultSlice))
	}

	if resultSlice[0]["type"] != "image_url" {
		t.Errorf("Expected image_url type, got %s", resultSlice[0]["type"])
	}

	imageURL, ok := resultSlice[0]["image_url"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected image_url to be a map")
	}

	url, _ := imageURL["url"].(string)
	if url != "data:image/jpeg;base64,base64data" {
		t.Errorf("Expected data URL, got %s", url)
	}
}

func TestConvertOpenAITools_InvalidInputTypes(t *testing.T) {
	// Test with non-map items
	input := []interface{}{
		"not a map",
		123,
		nil,
	}

	result := ConvertOpenAITools(input)
	if result != nil {
		t.Errorf("Expected nil for invalid inputs, got %v", result)
	}
}

func TestNormalizeContentForCopilot_InputImage(t *testing.T) {
	// Test input_image with nested url
	input := []interface{}{
		map[string]interface{}{
			"type": "input_image",
			"image_url": map[string]interface{}{
				"url": "http://example.com/image.png",
			},
		},
	}

	result := NormalizeContentForCopilot(input)
	resultSlice, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatal("Expected slice result")
	}

	if len(resultSlice) != 1 {
		t.Errorf("Expected 1 item, got %d", len(resultSlice))
	}

	if resultSlice[0]["type"] != "image_url" {
		t.Errorf("Expected image_url type, got %s", resultSlice[0]["type"])
	}
}

func TestConvertResponsesInputToMessages_FallbackTypes(t *testing.T) {
	// Test fallback for unknown item types
	input := []interface{}{
		map[string]interface{}{
			"role":    "user",
			"content": "Hello from content field",
		},
	}

	result := ConvertResponsesInputToMessages(input, "")

	if len(result) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result))
	}

	if result[0]["role"] != "user" {
		t.Errorf("Expected user role, got %s", result[0]["role"])
	}
}

func TestConvertOpenAIToCopilotMessages_WithToolCalls(t *testing.T) {
	toolCalls := []interface{}{
		map[string]interface{}{
			"id":   "call_123",
			"type": "function",
			"function": map[string]interface{}{
				"name":      "get_weather",
				"arguments": `{"location": "NYC"}`,
			},
		},
	}

	input := []map[string]interface{}{
		{
			"role":       "assistant",
			"content":    nil,
			"tool_calls": toolCalls,
		},
	}

	result := ConvertOpenAIToCopilotMessages(input)

	if len(result) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result))
	}

	if result[0]["tool_calls"] == nil {
		t.Error("Expected tool_calls to be preserved")
	}

	if !reflect.DeepEqual(result[0]["tool_calls"], toolCalls) {
		t.Error("Tool calls were not preserved correctly")
	}
}
