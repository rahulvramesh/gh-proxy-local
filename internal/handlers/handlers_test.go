package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// MockResponseWriter is a mock response writer for testing streaming
type MockResponseWriter struct {
	headers http.Header
	body    bytes.Buffer
	status  int
	flushed int
}

func NewMockResponseWriter() *MockResponseWriter {
	return &MockResponseWriter{
		headers: make(http.Header),
		status:  http.StatusOK,
	}
}

func (m *MockResponseWriter) Header() http.Header {
	return m.headers
}

func (m *MockResponseWriter) Write(b []byte) (int, error) {
	return m.body.Write(b)
}

func (m *MockResponseWriter) WriteHeader(statusCode int) {
	m.status = statusCode
}

func (m *MockResponseWriter) Flush() {
	m.flushed++
}

func TestHealthHandler_Health(t *testing.T) {
	handler := &HealthHandler{}

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	handler.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", response["status"])
	}

	if response["service"] != "github-copilot-proxy" {
		t.Errorf("Expected service 'github-copilot-proxy', got %v", response["service"])
	}

	if response["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %v", response["version"])
	}

	endpoints, ok := response["endpoints"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected endpoints to be a map")
	}

	if _, ok := endpoints["openai"]; !ok {
		t.Error("Expected openai endpoints")
	}

	if _, ok := endpoints["anthropic"]; !ok {
		t.Error("Expected anthropic endpoints")
	}
}

func TestChatHandler_InvalidRequest(t *testing.T) {
	handler := &ChatHandler{}

	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ChatCompletions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestResponsesHandler_InvalidRequest(t *testing.T) {
	handler := &ResponsesHandler{}

	req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Responses(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestAnthropicHandler_InvalidRequest(t *testing.T) {
	handler := &AnthropicHandler{}

	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Messages(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestAnthropicHandler_CountTokens(t *testing.T) {
	handler := &AnthropicHandler{}

	reqBody := map[string]interface{}{
		"model": "claude-sonnet-4",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "Hello, how are you?"},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/v1/messages/count_tokens", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CountTokens(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["input_tokens"]; !ok {
		t.Error("Expected input_tokens in response")
	}
}

func TestAnthropicHandler_CountTokens_InvalidRequest(t *testing.T) {
	handler := &AnthropicHandler{}

	req := httptest.NewRequest("POST", "/v1/messages/count_tokens", strings.NewReader("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CountTokens(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestAnthropicHandler_Batches(t *testing.T) {
	handler := &AnthropicHandler{}

	req := httptest.NewRequest("POST", "/v1/messages/batches", nil)
	rec := httptest.NewRecorder()

	handler.Batches(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Errorf("Expected status 501, got %d", rec.Code)
	}
}

func TestResponsesHandler_FilterFunctionTools(t *testing.T) {
	handler := &ResponsesHandler{}

	tests := []struct {
		name     string
		input    []interface{}
		expected int
	}{
		{
			name:     "empty tools",
			input:    []interface{}{},
			expected: 0,
		},
		{
			name: "function tools only",
			input: []interface{}{
				map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name": "test_func",
					},
				},
			},
			expected: 1,
		},
		{
			name: "mixed tool types",
			input: []interface{}{
				map[string]interface{}{"type": "code_interpreter"},
				map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name": "valid",
					},
				},
				map[string]interface{}{"type": "web_search"},
			},
			expected: 1,
		},
		{
			name: "function without name",
			input: []interface{}{
				map[string]interface{}{
					"type":     "function",
					"function": map[string]interface{}{},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.filterFunctionTools(tt.input)

			if tt.expected == 0 && result != nil {
				t.Errorf("Expected nil, got %v", result)
			}

			if tt.expected > 0 && len(result) != tt.expected {
				t.Errorf("Expected %d tools, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestResponsesHandler_ConvertToResponsesOutput(t *testing.T) {
	handler := &ResponsesHandler{}

	resp := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"content": "Hello there!",
				},
			},
		},
	}

	result := handler.convertToResponsesOutput(resp)

	if len(result) != 1 {
		t.Fatalf("Expected 1 output, got %d", len(result))
	}

	output := result[0].(map[string]interface{})
	if output["type"] != "message" {
		t.Errorf("Expected type 'message', got %v", output["type"])
	}

	if output["role"] != "assistant" {
		t.Errorf("Expected role 'assistant', got %v", output["role"])
	}
}

func TestResponsesHandler_ConvertToResponsesOutput_WithToolCalls(t *testing.T) {
	handler := &ResponsesHandler{}

	resp := map[string]interface{}{
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
								"arguments": `{"location":"NYC"}`,
							},
						},
					},
				},
			},
		},
	}

	result := handler.convertToResponsesOutput(resp)

	// Should have function_call output
	hasFunctionCall := false
	for _, r := range result {
		output := r.(map[string]interface{})
		if output["type"] == "function_call" {
			hasFunctionCall = true
			if output["name"] != "get_weather" {
				t.Errorf("Expected name 'get_weather', got %v", output["name"])
			}
		}
	}

	if !hasFunctionCall {
		t.Error("Expected function_call in output")
	}
}

func TestResponsesHandler_ExtractUsage(t *testing.T) {
	handler := &ResponsesHandler{}

	resp := map[string]interface{}{
		"usage": map[string]interface{}{
			"prompt_tokens":     float64(100),
			"completion_tokens": float64(50),
			"total_tokens":      float64(150),
			"prompt_tokens_details": map[string]interface{}{
				"cached_tokens": float64(20),
			},
			"completion_tokens_details": map[string]interface{}{
				"accepted_prediction_tokens": float64(10),
				"rejected_prediction_tokens": float64(5),
			},
		},
	}

	result := handler.extractUsage(resp)

	if result["input_tokens"] != 100 {
		t.Errorf("Expected input_tokens 100, got %v", result["input_tokens"])
	}

	if result["output_tokens"] != 50 {
		t.Errorf("Expected output_tokens 50, got %v", result["output_tokens"])
	}

	if result["total_tokens"] != 150 {
		t.Errorf("Expected total_tokens 150, got %v", result["total_tokens"])
	}

	inputDetails := result["input_tokens_details"].(map[string]interface{})
	if inputDetails["cached_tokens"] != 20 {
		t.Errorf("Expected cached_tokens 20, got %v", inputDetails["cached_tokens"])
	}
}

func TestCorsMiddleware(t *testing.T) {
	// Test actual handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "OK")
	})

	handler := corsMiddleware(nextHandler)

	// Test OPTIONS request
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS, got %d", rec.Code)
	}

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected Access-Control-Allow-Origin header")
	}

	// Test regular request
	req = httptest.NewRequest("GET", "/test", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected CORS headers on regular request")
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-API-Key, anthropic-version, anthropic-beta")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func TestHealthHandler_HealthResponse(t *testing.T) {
	handler := &HealthHandler{}

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.Health(rec, req)

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type application/json")
	}

	var response map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&response)

	// Check endpoints structure
	endpoints, ok := response["endpoints"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected endpoints to be present")
	}

	openai, ok := endpoints["openai"].([]interface{})
	if !ok {
		t.Fatal("Expected openai endpoints to be a slice")
	}

	hasChat := false
	hasModels := false
	for _, ep := range openai {
		if ep == "/v1/chat/completions" {
			hasChat = true
		}
		if ep == "/v1/models" {
			hasModels = true
		}
	}

	if !hasChat {
		t.Error("Expected /v1/chat/completions in openai endpoints")
	}

	if !hasModels {
		t.Error("Expected /v1/models in openai endpoints")
	}
}

func TestAnthropicHandler_CountTokens_WithSystem(t *testing.T) {
	handler := &AnthropicHandler{}

	reqBody := map[string]interface{}{
		"model":  "claude-sonnet-4",
		"system": "You are a helpful assistant.",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/v1/messages/count_tokens", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CountTokens(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&response)

	tokens, ok := response["input_tokens"].(float64)
	if !ok {
		t.Fatal("Expected input_tokens to be a number")
	}

	if tokens <= 0 {
		t.Error("Expected positive token count")
	}
}

func TestResponsesHandler_ExtractUsage_Empty(t *testing.T) {
	handler := &ResponsesHandler{}

	resp := map[string]interface{}{
		"usage": map[string]interface{}{},
	}

	result := handler.extractUsage(resp)

	if result["input_tokens"] != 0 {
		t.Errorf("Expected input_tokens 0 for empty usage, got %v", result["input_tokens"])
	}

	if result["output_tokens"] != 0 {
		t.Errorf("Expected output_tokens 0 for empty usage, got %v", result["output_tokens"])
	}
}

func TestResponsesHandler_ConvertToResponsesOutput_EmptyChoices(t *testing.T) {
	handler := &ResponsesHandler{}

	resp := map[string]interface{}{
		"choices": []interface{}{},
	}

	result := handler.convertToResponsesOutput(resp)

	if len(result) != 0 {
		t.Errorf("Expected empty output for empty choices, got %d", len(result))
	}
}
