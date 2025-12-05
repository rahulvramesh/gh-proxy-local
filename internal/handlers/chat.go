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
)

// ChatHandler handles OpenAI chat completions endpoints.
type ChatHandler struct {
	client *copilot.Client
	debug  bool
}

// NewChatHandler creates a new chat handler.
func NewChatHandler(client *copilot.Client, debug bool) *ChatHandler {
	return &ChatHandler{client: client, debug: debug}
}

// ChatCompletions handles POST /v1/chat/completions and /chat/completions
func (h *ChatHandler) ChatCompletions(w http.ResponseWriter, r *http.Request) {
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
		h.streamChatCompletions(w, r, chatReq)
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

	// Ensure proper format
	if resp.ID == "" {
		resp.ID = "chatcmpl-" + uuid.New().String()
	}
	resp.Object = "chat.completion"
	if resp.Created == 0 {
		resp.Created = time.Now().Unix()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// streamChatCompletions handles streaming chat completions.
func (h *ChatHandler) streamChatCompletions(w http.ResponseWriter, r *http.Request, req *copilot.ChatRequest) {
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

				outData, _ := json.Marshal(chunkData)
				fmt.Fprintf(w, "data: %s\n\n", string(outData))
				flusher.Flush()
			}
		}

		return nil
	})

	if err != nil {
		// Log error but don't send HTTP error (headers already sent)
		if h.debug {
			fmt.Printf("[DEBUG] Streaming error: %v\n", err)
		}
	}
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
