package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rahulvramesh/gh-proxy-local/internal/converter"
	"github.com/rahulvramesh/gh-proxy-local/internal/copilot"
)

// AnthropicHandler handles Anthropic messages API endpoints.
type AnthropicHandler struct {
	client *copilot.Client
	debug  bool
}

// NewAnthropicHandler creates a new Anthropic handler.
func NewAnthropicHandler(client *copilot.Client, debug bool) *AnthropicHandler {
	return &AnthropicHandler{client: client, debug: debug}
}

// Messages handles POST /v1/messages and /messages
func (h *AnthropicHandler) Messages(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Model         string                   `json:"model"`
		Messages      []map[string]interface{} `json:"messages"`
		MaxTokens     int                      `json:"max_tokens"`
		Temperature   *float64                 `json:"temperature"`
		System        interface{}              `json:"system"`
		Stream        bool                     `json:"stream"`
		Tools         []interface{}            `json:"tools"`
		ToolChoice    interface{}              `json:"tool_choice"`
		StopSequences []string                 `json:"stop_sequences"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	systemText := converter.ExtractSystemText(req.System)
	messages := converter.ConvertAnthropicToCopilotMessages(req.Messages, systemText)
	tools := converter.ConvertAnthropicTools(req.Tools)

	temperature := 0.7
	if req.Temperature != nil {
		temperature = *req.Temperature
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	chatReq := &copilot.ChatRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Stream:      req.Stream,
		Tools:       tools,
	}

	if req.Stream {
		h.streamMessages(w, r, chatReq, req.Model)
		return
	}

	resp, err := h.client.ChatCompletions(r.Context(), chatReq)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "API error") {
			statusCode = http.StatusBadGateway
		}
		http.Error(w, err.Error(), statusCode)
		return
	}

	// Convert to Anthropic format
	respMap := make(map[string]interface{})
	data, _ := json.Marshal(resp)
	json.Unmarshal(data, &respMap)

	anthropicResp := converter.ConvertOpenAIResponseToAnthropic(respMap, req.Model)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(anthropicResp)
}

// streamMessages handles streaming for Anthropic messages.
func (h *AnthropicHandler) streamMessages(w http.ResponseWriter, r *http.Request, req *copilot.ChatRequest, model string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	requestID := uuid.New().String()
	contentBlockIndex := 0
	hasTextContent := false
	toolCallsInProgress := make(map[int]map[string]interface{})
	stopReason := "end_turn"

	usageData := map[string]int{
		"input_tokens":            0,
		"output_tokens":           0,
		"cache_read_input_tokens": 0,
	}

	// Message start event
	h.sendAnthropicEvent(w, flusher, "message_start", map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":            requestID,
			"type":          "message",
			"role":          "assistant",
			"content":       []interface{}{},
			"model":         model,
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]interface{}{
				"input_tokens":  0,
				"output_tokens": 0,
			},
		},
	})

	// Content block start for text
	h.sendAnthropicEvent(w, flusher, "content_block_start", map[string]interface{}{
		"type":  "content_block_start",
		"index": contentBlockIndex,
		"content_block": map[string]interface{}{
			"type": "text",
			"text": "",
		},
	})

	err := h.client.ChatCompletionsStream(r.Context(), req, func(chunk []byte) error {
		line := string(chunk)

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return nil
			}

			var chunkData map[string]interface{}
			if err := json.Unmarshal([]byte(data), &chunkData); err == nil {
				// Capture usage data if present
				if usage, ok := chunkData["usage"].(map[string]interface{}); ok {
					if pt, ok := usage["prompt_tokens"].(float64); ok {
						usageData["input_tokens"] = int(pt)
					}
					if ct, ok := usage["completion_tokens"].(float64); ok {
						usageData["output_tokens"] = int(ct)
					}
					if pd, ok := usage["prompt_tokens_details"].(map[string]interface{}); ok {
						if cached, ok := pd["cached_tokens"].(float64); ok {
							usageData["cache_read_input_tokens"] = int(cached)
						}
					}
				}

				choices, _ := chunkData["choices"].([]interface{})
				if len(choices) > 0 {
					choice := choices[0].(map[string]interface{})
					delta, _ := choice["delta"].(map[string]interface{})
					finishReason, _ := choice["finish_reason"].(string)

					if finishReason == "tool_calls" {
						stopReason = "tool_use"
					}

					// Handle text content
					if content, ok := delta["content"].(string); ok && content != "" {
						hasTextContent = true
						h.sendAnthropicEvent(w, flusher, "content_block_delta", map[string]interface{}{
							"type":  "content_block_delta",
							"index": contentBlockIndex,
							"delta": map[string]interface{}{
								"type": "text_delta",
								"text": content,
							},
						})
					}

					// Handle tool calls
					if toolCalls, ok := delta["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
						for _, tc := range toolCalls {
							tcMap := tc.(map[string]interface{})
							tcIndex := int(tcMap["index"].(float64))
							tcID, _ := tcMap["id"].(string)
							tcFunc, _ := tcMap["function"].(map[string]interface{})

							if tcID != "" {
								// New tool call starting
								if hasTextContent || len(toolCallsInProgress) > 0 {
									// Close previous content block
									h.sendAnthropicEvent(w, flusher, "content_block_stop", map[string]interface{}{
										"type":  "content_block_stop",
										"index": contentBlockIndex,
									})
									contentBlockIndex++
								}

								funcName, _ := tcFunc["name"].(string)
								toolCallsInProgress[tcIndex] = map[string]interface{}{
									"id":        tcID,
									"name":      funcName,
									"arguments": "",
								}

								// Start new tool_use content block
								h.sendAnthropicEvent(w, flusher, "content_block_start", map[string]interface{}{
									"type":  "content_block_start",
									"index": contentBlockIndex,
									"content_block": map[string]interface{}{
										"type":  "tool_use",
										"id":    tcID,
										"name":  funcName,
										"input": map[string]interface{}{},
									},
								})
							} else if existing, ok := toolCallsInProgress[tcIndex]; ok {
								// Continuing existing tool call
								if args, ok := tcFunc["arguments"].(string); ok && args != "" {
									existing["arguments"] = existing["arguments"].(string) + args
									h.sendAnthropicEvent(w, flusher, "content_block_delta", map[string]interface{}{
										"type":  "content_block_delta",
										"index": contentBlockIndex,
										"delta": map[string]interface{}{
											"type":         "input_json_delta",
											"partial_json": args,
										},
									})
								}
							}
						}
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		if h.debug {
			fmt.Printf("[DEBUG] Streaming error: %v\n", err)
		}
	}

	// Close the last content block
	h.sendAnthropicEvent(w, flusher, "content_block_stop", map[string]interface{}{
		"type":  "content_block_stop",
		"index": contentBlockIndex,
	})

	// Message delta (final)
	h.sendAnthropicEvent(w, flusher, "message_delta", map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]interface{}{
			"stop_reason":   stopReason,
			"stop_sequence": nil,
		},
		"usage": map[string]interface{}{
			"output_tokens":           usageData["output_tokens"],
			"cache_read_input_tokens": usageData["cache_read_input_tokens"],
		},
	})

	// Message stop
	h.sendAnthropicEvent(w, flusher, "message_stop", map[string]interface{}{
		"type": "message_stop",
	})
}

// sendAnthropicEvent sends an Anthropic SSE event.
func (h *AnthropicHandler) sendAnthropicEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	dataJSON, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(dataJSON))
	flusher.Flush()
}

// CountTokens handles POST /v1/messages/count_tokens (stub implementation)
func (h *AnthropicHandler) CountTokens(w http.ResponseWriter, r *http.Request) {
	// This is a stub - actual token counting would require a tokenizer
	// For now, we'll return an estimate based on character count
	var req struct {
		Model    string                   `json:"model"`
		Messages []map[string]interface{} `json:"messages"`
		System   interface{}              `json:"system"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Simple estimation: ~4 characters per token
	charCount := 0

	if systemText := converter.ExtractSystemText(req.System); systemText != "" {
		charCount += len(systemText)
	}

	for _, msg := range req.Messages {
		if content, ok := msg["content"].(string); ok {
			charCount += len(content)
		}
	}

	estimatedTokens := charCount / 4
	if estimatedTokens == 0 {
		estimatedTokens = 1
	}

	response := map[string]interface{}{
		"input_tokens": estimatedTokens,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Batches handles batch endpoints (not supported)
func (h *AnthropicHandler) Batches(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Batch API not supported by Copilot proxy", http.StatusNotImplemented)
}
