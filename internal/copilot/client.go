// Package copilot provides the Copilot API client.
package copilot

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rahulvramesh/gh-proxy-local/internal/auth"
	"github.com/rahulvramesh/gh-proxy-local/internal/config"
	"github.com/rahulvramesh/gh-proxy-local/internal/models"
)

// Client is the Copilot API client.
type Client struct {
	authManager *auth.Manager
	httpClient  *http.Client
	debug       bool

	// Models cache
	modelsCache     []models.CopilotModel
	modelsCacheTime time.Time
	modelsMu        sync.RWMutex
}

// NewClient creates a new Copilot client.
func NewClient(authManager *auth.Manager, debug bool) *Client {
	return &Client{
		authManager: authManager,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Longer timeout for streaming
		},
		debug: debug,
	}
}

// debugLog prints debug messages if debugging is enabled.
func (c *Client) debugLog(format string, args ...interface{}) {
	if c.debug {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// FetchModels fetches available models from Copilot API.
func (c *Client) FetchModels(ctx context.Context) ([]models.CopilotModel, error) {
	c.modelsMu.RLock()
	if c.modelsCache != nil && time.Since(c.modelsCacheTime).Seconds() < config.ModelsCacheTTL {
		models := c.modelsCache
		c.modelsMu.RUnlock()
		return models, nil
	}
	c.modelsMu.RUnlock()

	creds, err := c.authManager.GetCredentials()
	if err != nil {
		return models.FallbackModels(), nil
	}

	copilotToken, err := c.authManager.GetCopilotToken(creds)
	if err != nil {
		return models.FallbackModels(), nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", config.CopilotAPIBase+"/models", nil)
	if err != nil {
		return models.FallbackModels(), nil
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+copilotToken)
	for k, v := range config.CopilotHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.debugLog("Failed to fetch models: %v", err)
		return models.FallbackModels(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.debugLog("Failed to fetch models: %d %s", resp.StatusCode, string(body))
		return models.FallbackModels(), nil
	}

	var modelsResp models.CopilotModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		c.debugLog("Failed to decode models: %v", err)
		return models.FallbackModels(), nil
	}

	// Convert to our model format, filter out embedding and oswe models
	result := make([]models.CopilotModel, 0, len(modelsResp.Data))
	for _, m := range modelsResp.Data {
		// Skip embedding models
		if strings.Contains(strings.ToLower(m.ID), "embedding") {
			continue
		}
		// Skip oswe models
		if strings.HasPrefix(m.ID, "oswe-") {
			continue
		}

		result = append(result, models.CopilotModel{
			ID:      m.ID,
			Object:  "model",
			Created: 1700000000,
			OwnedBy: models.GetVendorOwner(m.Vendor),
			Name:    m.Name,
			Version: m.Version,
			Preview: m.Preview,
			Limits: &models.ModelLimits{
				MaxContextWindowTokens: m.Capabilities.Limits.MaxContextWindowTokens,
				MaxOutputTokens:        m.Capabilities.Limits.MaxOutputTokens,
				MaxPromptTokens:        m.Capabilities.Limits.MaxPromptTokens,
			},
			Capabilities: &models.ModelCapabilities{
				Vision:            m.Capabilities.Supports.Vision,
				ToolCalls:         m.Capabilities.Supports.ToolCalls,
				ParallelToolCalls: m.Capabilities.Supports.ParallelToolCalls,
				Streaming:         m.Capabilities.Supports.Streaming,
				StructuredOutputs: m.Capabilities.Supports.StructuredOutputs,
			},
		})
	}

	c.modelsMu.Lock()
	c.modelsCache = result
	c.modelsCacheTime = time.Now()
	c.modelsMu.Unlock()

	c.debugLog("Fetched %d models from Copilot API", len(result))
	return result, nil
}

// ChatRequest represents a request to the chat completions API.
type ChatRequest struct {
	Model       string
	Messages    []map[string]interface{}
	Temperature float64
	MaxTokens   int
	Stream      bool
	Tools       []map[string]interface{}
}

// ChatCompletions makes a chat completions request to Copilot API.
func (c *Client) ChatCompletions(ctx context.Context, req *ChatRequest) (*models.OpenAIChatResponse, error) {
	creds, err := c.authManager.GetCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	copilotToken, err := c.authManager.GetCopilotToken(creds)
	if err != nil {
		return nil, fmt.Errorf("failed to get copilot token: %w", err)
	}

	resolvedModel := models.ResolveModel(req.Model)
	c.debugLog("Request to model: %s (original: %s)", resolvedModel, req.Model)

	payload := map[string]interface{}{
		"model":       resolvedModel,
		"messages":    req.Messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
		"stream":      false,
	}

	if len(req.Tools) > 0 {
		payload["tools"] = req.Tools
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", config.CopilotAPIBase+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+copilotToken)
	httpReq.Header.Set("Openai-Intent", "conversation-edits")
	for k, v := range config.CopilotHeaders {
		httpReq.Header.Set(k, v)
	}

	// Add vision header if images are present
	if hasImageContent(req.Messages) {
		httpReq.Header.Set("Copilot-Vision-Request", "true")
		c.debugLog("Added Copilot-Vision-Request header for image content")
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result models.OpenAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// StreamCallback is called for each chunk in a streaming response.
type StreamCallback func(chunk []byte) error

// ChatCompletionsStream makes a streaming chat completions request.
func (c *Client) ChatCompletionsStream(ctx context.Context, req *ChatRequest, callback StreamCallback) error {
	creds, err := c.authManager.GetCredentials()
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	copilotToken, err := c.authManager.GetCopilotToken(creds)
	if err != nil {
		return fmt.Errorf("failed to get copilot token: %w", err)
	}

	resolvedModel := models.ResolveModel(req.Model)
	c.debugLog("Streaming request to model: %s (original: %s)", resolvedModel, req.Model)

	payload := map[string]interface{}{
		"model":       resolvedModel,
		"messages":    req.Messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
		"stream":      true,
	}

	if len(req.Tools) > 0 {
		payload["tools"] = req.Tools
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", config.CopilotAPIBase+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+copilotToken)
	httpReq.Header.Set("Openai-Intent", "conversation-edits")
	for k, v := range config.CopilotHeaders {
		httpReq.Header.Set(k, v)
	}

	// Add vision header if images are present
	if hasImageContent(req.Messages) {
		httpReq.Header.Set("Copilot-Vision-Request", "true")
		c.debugLog("Added Copilot-Vision-Request header for image content")
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read error: %w", err)
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if err := callback(line); err != nil {
			return err
		}
	}

	return nil
}

// hasImageContent checks if any message contains image content.
func hasImageContent(messages []map[string]interface{}) bool {
	for _, msg := range messages {
		content, ok := msg["content"]
		if !ok {
			continue
		}

		switch c := content.(type) {
		case []interface{}:
			for _, block := range c {
				if blockMap, ok := block.(map[string]interface{}); ok {
					blockType, _ := blockMap["type"].(string)
					if blockType == "image_url" || blockType == "image" {
						return true
					}
				}
			}
		case []map[string]interface{}:
			for _, block := range c {
				blockType, _ := block["type"].(string)
				if blockType == "image_url" || blockType == "image" {
					return true
				}
			}
		}
	}
	return false
}
