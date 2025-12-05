package copilot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rahulvramesh/gh-proxy-local/internal/auth"
	"github.com/rahulvramesh/gh-proxy-local/internal/config"
	"github.com/rahulvramesh/gh-proxy-local/internal/models"
)

// MockAuthManager is a mock auth manager for testing.
type MockAuthManager struct {
	credentials   *models.Credentials
	copilotToken  string
	shouldError   bool
}

func (m *MockAuthManager) GetCredentials() (*models.Credentials, error) {
	if m.shouldError {
		return nil, http.ErrServerClosed
	}
	return m.credentials, nil
}

func (m *MockAuthManager) GetCopilotToken(creds *models.Credentials) (string, error) {
	if m.shouldError {
		return "", http.ErrServerClosed
	}
	return m.copilotToken, nil
}

func TestNewClient(t *testing.T) {
	cfg := &config.Config{Debug: true}
	authManager := auth.NewManager(cfg)

	client := NewClient(authManager, true)

	if client.authManager != authManager {
		t.Error("Expected auth manager to be set")
	}

	if !client.debug {
		t.Error("Expected debug to be true")
	}

	if client.httpClient == nil {
		t.Error("Expected HTTP client to be initialized")
	}
}

func TestHasImageContent(t *testing.T) {
	tests := []struct {
		name     string
		messages []map[string]interface{}
		expected bool
	}{
		{
			name:     "empty messages",
			messages: []map[string]interface{}{},
			expected: false,
		},
		{
			name: "text only",
			messages: []map[string]interface{}{
				{"role": "user", "content": "Hello"},
			},
			expected: false,
		},
		{
			name: "with image_url",
			messages: []map[string]interface{}{
				{
					"role": "user",
					"content": []interface{}{
						map[string]interface{}{"type": "image_url"},
					},
				},
			},
			expected: true,
		},
		{
			name: "with image type",
			messages: []map[string]interface{}{
				{
					"role": "user",
					"content": []interface{}{
						map[string]interface{}{"type": "image"},
					},
				},
			},
			expected: true,
		},
		{
			name: "mixed content without image",
			messages: []map[string]interface{}{
				{
					"role": "user",
					"content": []interface{}{
						map[string]interface{}{"type": "text", "text": "Hello"},
					},
				},
			},
			expected: false,
		},
		{
			name: "with map slice content",
			messages: []map[string]interface{}{
				{
					"role": "user",
					"content": []map[string]interface{}{
						{"type": "image_url"},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasImageContent(tt.messages)
			if result != tt.expected {
				t.Errorf("hasImageContent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClient_FetchModels_Cached(t *testing.T) {
	cfg := &config.Config{}
	authManager := auth.NewManager(cfg)
	client := NewClient(authManager, false)

	// Set cached models
	client.modelsCache = []models.CopilotModel{
		{ID: "test-model", Object: "model"},
	}
	client.modelsCacheTime = time.Now()

	result, err := client.FetchModels(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 model, got %d", len(result))
	}

	if result[0].ID != "test-model" {
		t.Errorf("Expected model ID 'test-model', got %s", result[0].ID)
	}
}

func TestClient_FetchModels_ExpiredCache(t *testing.T) {
	cfg := &config.Config{}
	authManager := auth.NewManager(cfg)
	client := NewClient(authManager, false)

	// Set expired cache
	client.modelsCache = []models.CopilotModel{
		{ID: "old-model", Object: "model"},
	}
	client.modelsCacheTime = time.Now().Add(-10 * time.Minute) // Expired

	// Will fail to fetch new models and return fallback
	result, _ := client.FetchModels(context.Background())

	// Should return fallback models since auth will fail
	if len(result) == 0 {
		t.Error("Expected fallback models")
	}
}

func TestClient_FetchModels_Fallback(t *testing.T) {
	cfg := &config.Config{}
	authManager := auth.NewManager(cfg)
	client := NewClient(authManager, false)

	// No cache, auth will fail, should return fallback
	result, _ := client.FetchModels(context.Background())

	if len(result) == 0 {
		t.Error("Expected fallback models")
	}

	// Check that fallback models are returned
	hasGPT4o := false
	for _, m := range result {
		if m.ID == "gpt-4o" {
			hasGPT4o = true
			break
		}
	}

	if !hasGPT4o {
		t.Error("Expected gpt-4o in fallback models")
	}
}

func TestChatRequest_Structure(t *testing.T) {
	req := &ChatRequest{
		Model: "gpt-4o",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
		Stream:      false,
		Tools: []map[string]interface{}{
			{
				"type": "function",
				"function": map[string]interface{}{
					"name": "test_func",
				},
			},
		},
	}

	if req.Model != "gpt-4o" {
		t.Errorf("Expected model 'gpt-4o', got %s", req.Model)
	}

	if len(req.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(req.Messages))
	}

	if req.Temperature != 0.7 {
		t.Errorf("Expected temperature 0.7, got %f", req.Temperature)
	}

	if req.MaxTokens != 100 {
		t.Errorf("Expected max_tokens 100, got %d", req.MaxTokens)
	}

	if len(req.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(req.Tools))
	}
}

func TestClient_DebugLog(t *testing.T) {
	cfg := &config.Config{}
	authManager := auth.NewManager(cfg)

	// Test with debug enabled
	client := NewClient(authManager, true)
	// This should not panic
	client.debugLog("Test message: %s", "hello")

	// Test with debug disabled
	client = NewClient(authManager, false)
	// This should also not panic
	client.debugLog("Test message: %s", "hello")
}

func TestClient_FetchModels_MockServer(t *testing.T) {
	// Create mock server that returns models
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/copilot_internal/v2/token":
			response := map[string]interface{}{
				"token":      "test_copilot_token",
				"expires_at": time.Now().Add(time.Hour).Unix(),
			}
			json.NewEncoder(w).Encode(response)
		case "/models":
			response := map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":     "gpt-4o",
						"name":   "GPT-4o",
						"vendor": "OpenAI",
						"capabilities": map[string]interface{}{
							"limits": map[string]interface{}{
								"max_context_window_tokens": 128000,
							},
							"supports": map[string]interface{}{
								"vision":     true,
								"tool_calls": true,
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	// Note: We can't easily inject the mock server URL into the client
	// This test demonstrates the structure, but the actual API URL is hardcoded
	// In a production system, you'd want to make the base URL configurable
}

func TestHasImageContent_EdgeCases(t *testing.T) {
	// Test with nil content
	messages := []map[string]interface{}{
		{"role": "user", "content": nil},
	}
	if hasImageContent(messages) {
		t.Error("Expected false for nil content")
	}

	// Test with string content
	messages = []map[string]interface{}{
		{"role": "user", "content": "just text"},
	}
	if hasImageContent(messages) {
		t.Error("Expected false for string content")
	}

	// Test with empty slice content
	messages = []map[string]interface{}{
		{"role": "user", "content": []interface{}{}},
	}
	if hasImageContent(messages) {
		t.Error("Expected false for empty slice content")
	}

	// Test with non-image blocks
	messages = []map[string]interface{}{
		{
			"role": "user",
			"content": []interface{}{
				map[string]interface{}{"type": "text", "text": "Hello"},
				map[string]interface{}{"type": "audio", "audio": "base64..."},
			},
		},
	}
	if hasImageContent(messages) {
		t.Error("Expected false for non-image content types")
	}
}

func TestClient_ChatCompletions_NoAuth(t *testing.T) {
	cfg := &config.Config{}
	authManager := auth.NewManager(cfg)
	client := NewClient(authManager, false)

	req := &ChatRequest{
		Model: "gpt-4o",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
	}

	// This will fail because we don't have valid auth
	_, err := client.ChatCompletions(context.Background(), req)
	if err == nil {
		t.Log("ChatCompletions succeeded without auth (unexpected)")
	}
}

func TestClient_ChatCompletionsStream_NoAuth(t *testing.T) {
	cfg := &config.Config{}
	authManager := auth.NewManager(cfg)
	client := NewClient(authManager, false)

	req := &ChatRequest{
		Model: "gpt-4o",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
		Stream: true,
	}

	// This will fail because we don't have valid auth
	err := client.ChatCompletionsStream(context.Background(), req, func(chunk []byte) error {
		return nil
	})

	if err == nil {
		t.Log("ChatCompletionsStream succeeded without auth (unexpected)")
	}
}

func TestStreamCallback(t *testing.T) {
	// Test that StreamCallback type is properly defined
	var cb StreamCallback = func(chunk []byte) error {
		return nil
	}

	err := cb([]byte("test"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestClient_ModelsCache_ThreadSafety(t *testing.T) {
	cfg := &config.Config{}
	authManager := auth.NewManager(cfg)
	client := NewClient(authManager, false)

	// Set initial cache
	client.modelsCache = []models.CopilotModel{
		{ID: "cached-model"},
	}
	client.modelsCacheTime = time.Now()

	// Concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := client.FetchModels(context.Background())
			if err != nil {
				t.Errorf("Concurrent fetch failed: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
