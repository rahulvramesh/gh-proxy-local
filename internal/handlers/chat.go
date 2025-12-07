package handlers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rahulvramesh/gh-proxy-local/internal/converter"
	"github.com/rahulvramesh/gh-proxy-local/internal/copilot"
	"github.com/rahulvramesh/gh-proxy-local/internal/langfuse"
)

// ChatHandler handles OpenAI chat completions endpoints.
type ChatHandler struct {
	client   *copilot.Client
	langfuse *langfuse.Client
	debug    bool
}

// NewChatHandler creates a new chat handler.
func NewChatHandler(client *copilot.Client, langfuseClient *langfuse.Client, debug bool) *ChatHandler {
	return &ChatHandler{client: client, langfuse: langfuseClient, debug: debug}
}

// ChatCompletions handles POST /v1/chat/completions and /chat/completions
func (h *ChatHandler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	traceID := langfuse.GenerateTraceID()
	genID := langfuse.GenerateSpanID()

	var req struct {
		Model       string                   `json:"model"`
		Messages    []map[string]interface{} `json:"messages"`
		Temperature *float64                 `json:"temperature"`
		MaxTokens   *int                     `json:"max_tokens"`
		Stream      bool                     `json:"stream"`
		Tools       []interface{}            `json:"tools"`
		ToolChoice  interface{}              `json:"tool_choice"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	messages := converter.ConvertOpenAIToCopilotMessages(req.Messages)
	tools := converter.ConvertOpenAITools(req.Tools)

	temperature := 0.7
	if req.Temperature != nil {
		temperature = *req.Temperature
	}

	maxTokens := 4096
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
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
		h.streamChatCompletions(w, r, chatReq, traceID, genID, startTime, req.Messages)
		return
	}

	resp, err := h.client.ChatCompletions(r.Context(), chatReq)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "API error") {
			statusCode = http.StatusBadGateway
		}
		h.trackGeneration(traceID, genID, req.Model, req.Messages, nil, nil, startTime, "ERROR", err.Error(), r)
		http.Error(w, err.Error(), statusCode)
		return
	}

	// Ensure proper format
	if resp.ID == "" {
		resp.ID = "chatcmpl-" + uuid.New().String()
	}
	resp.Object = "chat.completion"
	if resp.Created == 0 {
		resp.Created = time.Now().Unix()
	}

	// Track to Langfuse
	var output interface{}
	if len(resp.Choices) > 0 {
		output = resp.Choices[0].Message
	}
	usage := &langfuse.UsageData{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}
	h.trackGeneration(traceID, genID, req.Model, req.Messages, output, usage, startTime, "", "", r)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// trackGeneration sends generation data to Langfuse.
func (h *ChatHandler) trackGeneration(traceID, genID, model string, input, output interface{}, usage *langfuse.UsageData, startTime time.Time, level, statusMessage string, r *http.Request) {
	if h.langfuse == nil || !h.langfuse.IsEnabled() {
		return
	}

	metadata := map[string]interface{}{
		"endpoint": "chat/completions",
		"api":      "openai",
	}

	gen := &langfuse.GenerationBody{
		ID:              genID,
		TraceID:         traceID,
		Name:            "chat-completion",
		Model:           model,
		Input:           input,
		Output:          output,
		Usage:           usage,
		Metadata:        metadata,
		StartTime:       startTime,
		EndTime:         time.Now(),
		Level:           level,
		StatusMessage:   statusMessage,
	}

	h.langfuse.TrackGeneration(gen)
}

// streamChatCompletions handles streaming chat completions.
func (h *ChatHandler) streamChatCompletions(w http.ResponseWriter, r *http.Request, req *copilot.ChatRequest, traceID, genID string, startTime time.Time, inputMessages []map[string]interface{}) {
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
	created := time.Now().Unix()

	var fullContent strings.Builder
	var usageData *langfuse.UsageData

	err := h.client.ChatCompletionsStream(r.Context(), req, func(chunk []byte) error {
		line := string(chunk)

		// Pass through SSE format directly
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				return nil
			}

			var chunkData map[string]interface{}
			if err := json.Unmarshal([]byte(data), &chunkData); err == nil {
				// Add id and created if missing
				if chunkData["id"] == nil {
					chunkData["id"] = "chatcmpl-" + requestID
				}
				if chunkData["created"] == nil {
					chunkData["created"] = created
				}

				// Capture content for Langfuse
				if choices, ok := chunkData["choices"].([]interface{}); ok && len(choices) > 0 {
					if choice, ok := choices[0].(map[string]interface{}); ok {
						if delta, ok := choice["delta"].(map[string]interface{}); ok {
							if content, ok := delta["content"].(string); ok {
								fullContent.WriteString(content)
							}
						}
					}
				}

				// Capture usage data if present
				if usage, ok := chunkData["usage"].(map[string]interface{}); ok {
					usageData = &langfuse.UsageData{}
					if pt, ok := usage["prompt_tokens"].(float64); ok {
						usageData.PromptTokens = int(pt)
					}
					if ct, ok := usage["completion_tokens"].(float64); ok {
						usageData.CompletionTokens = int(ct)
					}
					if tt, ok := usage["total_tokens"].(float64); ok {
						usageData.TotalTokens = int(tt)
					}
				}

				outData, _ := json.Marshal(chunkData)
				fmt.Fprintf(w, "data: %s\n\n", string(outData))
				flusher.Flush()
			}
		}

		return nil
	})

	// Track streaming completion to Langfuse
	level := ""
	statusMsg := ""
	if err != nil {
		level = "ERROR"
		statusMsg = err.Error()
		if h.debug {
			fmt.Printf("[DEBUG] Streaming error: %v\n", err)
		}
	}

	output := map[string]interface{}{
		"role":    "assistant",
		"content": fullContent.String(),
	}
	h.trackGeneration(traceID, genID, req.Model, inputMessages, output, usageData, startTime, level, statusMsg, r)
}

// StreamOpenAIResponse streams an OpenAI format response.
func StreamOpenAIResponse(w http.ResponseWriter, r *http.Request, reader *bufio.Reader) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	requestID := uuid.New().String()
	created := time.Now().Unix()

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		lineStr := string(line)
		if strings.HasPrefix(lineStr, "data: ") {
			data := strings.TrimPrefix(lineStr, "data: ")
			if data == "[DONE]" {
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				break
			}

			var chunk map[string]interface{}
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if chunk["id"] == nil {
					chunk["id"] = "chatcmpl-" + requestID
				}
				if chunk["created"] == nil {
					chunk["created"] = created
				}

				outData, _ := json.Marshal(chunk)
				fmt.Fprintf(w, "data: %s\n\n", string(outData))
				flusher.Flush()
			}
		}
	}

	return nil
}
