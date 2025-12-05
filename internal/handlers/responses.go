package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rahulvramesh/gh-proxy-local/internal/converter"
	"github.com/rahulvramesh/gh-proxy-local/internal/copilot"
)

// ResponsesHandler handles OpenAI Responses API endpoints.
type ResponsesHandler struct {
	client *copilot.Client
	debug  bool
}

// NewResponsesHandler creates a new responses handler.
func NewResponsesHandler(client *copilot.Client, debug bool) *ResponsesHandler {
	return &ResponsesHandler{client: client, debug: debug}
}

// Responses handles POST /v1/responses and /responses
func (h *ResponsesHandler) Responses(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Model           string      `json:"model"`
		Input           interface{} `json:"input"`
		Instructions    string      `json:"instructions"`
		Temperature     *float64    `json:"temperature"`
		MaxOutputTokens *int        `json:"max_output_tokens"`
		Stream          bool        `json:"stream"`
		Tools           []interface{} `json:"tools"`
		ToolChoice      interface{} `json:"tool_choice"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	messages := converter.ConvertResponsesInputToMessages(req.Input, req.Instructions)
	tools := h.filterFunctionTools(req.Tools)

	temperature := 0.7
	if req.Temperature != nil {
		temperature = *req.Temperature
	}

	maxTokens := 4096
	if req.MaxOutputTokens != nil {
		maxTokens = *req.MaxOutputTokens
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
		h.streamResponses(w, r, chatReq, req.Model)
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

	// Convert to Responses API format
	output := h.convertToResponsesOutput(resp)

	responseID := "resp_" + uuid.New().String()[:24]
	created := time.Now().Unix()

	usage := h.extractUsage(resp)

	response := map[string]interface{}{
		"id":         responseID,
		"object":     "response",
		"created_at": created,
		"status":     "completed",
		"model":      req.Model,
		"output":     output,
		"usage":      usage,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// filterFunctionTools filters only function type tools.
func (h *ResponsesHandler) filterFunctionTools(tools []interface{}) []map[string]interface{} {
	if len(tools) == 0 {
		return nil
	}

	result := make([]map[string]interface{}, 0)

	for _, tool := range tools {
		toolMap, ok := tool.(map[string]interface{})
		if !ok {
			continue
		}

		toolType, _ := toolMap["type"].(string)
		if toolType != "function" {
			continue
		}

		funcData, ok := toolMap["function"].(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := funcData["name"].(string)
		if name == "" {
			continue
		}

		result = append(result, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        name,
				"description": funcData["description"],
				"parameters":  funcData["parameters"],
			},
		})
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// convertToResponsesOutput converts OpenAI response to Responses output format.
func (h *ResponsesHandler) convertToResponsesOutput(resp interface{}) []interface{} {
	output := make([]interface{}, 0)

	respMap, ok := resp.(map[string]interface{})
	if !ok {
		// Try to convert from struct
		data, err := json.Marshal(resp)
		if err != nil {
			return output
		}
		json.Unmarshal(data, &respMap)
	}

	choices, _ := respMap["choices"].([]interface{})
	if len(choices) == 0 {
		return output
	}

	choice := choices[0].(map[string]interface{})
	message, _ := choice["message"].(map[string]interface{})

	// Add text content
	if content, ok := message["content"].(string); ok && content != "" {
		output = append(output, map[string]interface{}{
			"type": "message",
			"role": "assistant",
			"content": []map[string]interface{}{
				{
					"type": "output_text",
					"text": content,
				},
			},
		})
	}

	// Handle tool calls
	if toolCalls, ok := message["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
		for _, tc := range toolCalls {
			tcMap := tc.(map[string]interface{})
			id, _ := tcMap["id"].(string)
			if id == "" {
				id = uuid.New().String()
			}

			function, _ := tcMap["function"].(map[string]interface{})
			name, _ := function["name"].(string)
			args, _ := function["arguments"].(string)
			if args == "" {
				args = "{}"
			}

			output = append(output, map[string]interface{}{
				"type":      "function_call",
				"id":        id,
				"name":      name,
				"arguments": args,
			})
		}
	}

	return output
}

// extractUsage extracts usage information from response.
func (h *ResponsesHandler) extractUsage(resp interface{}) map[string]interface{} {
	respMap, ok := resp.(map[string]interface{})
	if !ok {
		data, _ := json.Marshal(resp)
		json.Unmarshal(data, &respMap)
	}

	usage, _ := respMap["usage"].(map[string]interface{})
	promptTokens, _ := usage["prompt_tokens"].(float64)
	completionTokens, _ := usage["completion_tokens"].(float64)
	totalTokens, _ := usage["total_tokens"].(float64)

	promptDetails, _ := usage["prompt_tokens_details"].(map[string]interface{})
	cachedTokens, _ := promptDetails["cached_tokens"].(float64)

	completionDetails, _ := usage["completion_tokens_details"].(map[string]interface{})
	acceptedPrediction, _ := completionDetails["accepted_prediction_tokens"].(float64)
	rejectedPrediction, _ := completionDetails["rejected_prediction_tokens"].(float64)

	return map[string]interface{}{
		"input_tokens":  int(promptTokens),
		"output_tokens": int(completionTokens),
		"total_tokens":  int(totalTokens),
		"input_tokens_details": map[string]interface{}{
			"cached_tokens": int(cachedTokens),
		},
		"output_tokens_details": map[string]interface{}{
			"accepted_prediction_tokens": int(acceptedPrediction),
			"rejected_prediction_tokens": int(rejectedPrediction),
		},
	}
}

// streamResponses handles streaming for Responses API.
func (h *ResponsesHandler) streamResponses(w http.ResponseWriter, r *http.Request, req *copilot.ChatRequest, model string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	responseID := "resp_" + uuid.New().String()[:24]
	created := time.Now().Unix()
	outputIndex := 0
	contentIndex := 0

	usageData := map[string]int{
		"input_tokens":  0,
		"output_tokens": 0,
		"total_tokens":  0,
		"cached_tokens": 0,
	}

	// Response created event
	h.sendEvent(w, flusher, "response.created", map[string]interface{}{
		"type": "response.created",
		"response": map[string]interface{}{
			"id":         responseID,
			"object":     "response",
			"created_at": created,
			"status":     "in_progress",
			"model":      model,
			"output":     []interface{}{},
		},
	})

	// Output item added event
	h.sendEvent(w, flusher, "response.output_item.added", map[string]interface{}{
		"type":         "response.output_item.added",
		"output_index": outputIndex,
		"item": map[string]interface{}{
			"type":    "message",
			"role":    "assistant",
			"content": []interface{}{},
		},
	})

	// Content part added event
	h.sendEvent(w, flusher, "response.content_part.added", map[string]interface{}{
		"type":          "response.content_part.added",
		"output_index":  outputIndex,
		"content_index": contentIndex,
		"part": map[string]interface{}{
			"type": "output_text",
			"text": "",
		},
	})

	var fullText strings.Builder

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
					if tt, ok := usage["total_tokens"].(float64); ok {
						usageData["total_tokens"] = int(tt)
					}
					if pd, ok := usage["prompt_tokens_details"].(map[string]interface{}); ok {
						if cached, ok := pd["cached_tokens"].(float64); ok {
							usageData["cached_tokens"] = int(cached)
						}
					}
				}

				choices, _ := chunkData["choices"].([]interface{})
				if len(choices) > 0 {
					choice := choices[0].(map[string]interface{})
					delta, _ := choice["delta"].(map[string]interface{})
					content, _ := delta["content"].(string)

					if content != "" {
						fullText.WriteString(content)
						h.sendEvent(w, flusher, "response.output_text.delta", map[string]interface{}{
							"type":          "response.output_text.delta",
							"output_index":  outputIndex,
							"content_index": contentIndex,
							"delta":         content,
						})
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

	text := fullText.String()

	// Text done event
	h.sendEvent(w, flusher, "response.output_text.done", map[string]interface{}{
		"type":          "response.output_text.done",
		"output_index":  outputIndex,
		"content_index": contentIndex,
		"text":          text,
	})

	// Content part done event
	h.sendEvent(w, flusher, "response.content_part.done", map[string]interface{}{
		"type":          "response.content_part.done",
		"output_index":  outputIndex,
		"content_index": contentIndex,
		"part": map[string]interface{}{
			"type": "output_text",
			"text": text,
		},
	})

	// Output item done event
	h.sendEvent(w, flusher, "response.output_item.done", map[string]interface{}{
		"type":         "response.output_item.done",
		"output_index": outputIndex,
		"item": map[string]interface{}{
			"type": "message",
			"role": "assistant",
			"content": []map[string]interface{}{
				{
					"type": "output_text",
					"text": text,
				},
			},
		},
	})

	// Response completed event
	h.sendEvent(w, flusher, "response.completed", map[string]interface{}{
		"type": "response.completed",
		"response": map[string]interface{}{
			"id":         responseID,
			"object":     "response",
			"created_at": created,
			"status":     "completed",
			"model":      model,
			"output": []map[string]interface{}{
				{
					"type": "message",
					"role": "assistant",
					"content": []map[string]interface{}{
						{
							"type": "output_text",
							"text": text,
						},
					},
				},
			},
			"usage": map[string]interface{}{
				"input_tokens":  usageData["input_tokens"],
				"output_tokens": usageData["output_tokens"],
				"total_tokens":  usageData["total_tokens"],
				"input_tokens_details": map[string]interface{}{
					"cached_tokens": usageData["cached_tokens"],
				},
			},
		},
	})
}

// sendEvent sends an SSE event.
func (h *ResponsesHandler) sendEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) {
	dataJSON, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(dataJSON))
	flusher.Flush()
}
