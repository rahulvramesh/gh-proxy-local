package converter

import (
	"encoding/json"

	"github.com/google/uuid"
)

// ConvertAnthropicToCopilotMessages converts Anthropic message format to Copilot format.
func ConvertAnthropicToCopilotMessages(messages []map[string]interface{}, system string) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(messages)+1)

	// Add system message if provided
	if system != "" {
		result = append(result, map[string]interface{}{
			"role":    "system",
			"content": system,
		})
	}

	for _, msg := range messages {
		role, _ := msg["role"].(string)
		content := msg["content"]

		switch c := content.(type) {
		case string:
			result = append(result, map[string]interface{}{
				"role":    role,
				"content": c,
			})
		case []interface{}:
			// Process content blocks
			openaiContent := make([]map[string]interface{}, 0)
			toolCalls := make([]map[string]interface{}, 0)
			toolResults := make([]map[string]interface{}, 0)

			for _, block := range c {
				blockMap, ok := block.(map[string]interface{})
				if !ok {
					continue
				}

				blockType, _ := blockMap["type"].(string)

				switch blockType {
				case "text":
					text, _ := blockMap["text"].(string)
					openaiContent = append(openaiContent, map[string]interface{}{
						"type": "text",
						"text": text,
					})
				case "image":
					source, _ := blockMap["source"].(map[string]interface{})
					if sourceType, _ := source["type"].(string); sourceType == "base64" {
						mediaType, _ := source["media_type"].(string)
						if mediaType == "" {
							mediaType = "image/png"
						}
						data, _ := source["data"].(string)
						dataURL := "data:" + mediaType + ";base64," + data
						openaiContent = append(openaiContent, map[string]interface{}{
							"type": "image_url",
							"image_url": map[string]interface{}{
								"url": dataURL,
							},
						})
					}
				case "tool_use":
					id, _ := blockMap["id"].(string)
					if id == "" {
						id = uuid.New().String()
					}
					name, _ := blockMap["name"].(string)
					input, _ := blockMap["input"].(map[string]interface{})
					inputJSON, _ := json.Marshal(input)

					toolCalls = append(toolCalls, map[string]interface{}{
						"id":   id,
						"type": "function",
						"function": map[string]interface{}{
							"name":      name,
							"arguments": string(inputJSON),
						},
					})
				case "tool_result":
					toolUseID, _ := blockMap["tool_use_id"].(string)
					resultContent := blockMap["content"]
					contentStr := extractTextContent(resultContent)

					toolResults = append(toolResults, map[string]interface{}{
						"tool_call_id": toolUseID,
						"content":      contentStr,
					})
				}
			}

			// Build the message(s)
			if role == "assistant" {
				converted := map[string]interface{}{
					"role": "assistant",
				}

				if len(openaiContent) > 0 {
					if len(openaiContent) == 1 && openaiContent[0]["type"] == "text" {
						converted["content"] = openaiContent[0]["text"]
					} else {
						converted["content"] = openaiContent
					}
				} else {
					converted["content"] = nil
				}

				if len(toolCalls) > 0 {
					converted["tool_calls"] = toolCalls
				}

				result = append(result, converted)
			} else if role == "user" {
				// Handle tool results
				for _, tr := range toolResults {
					result = append(result, map[string]interface{}{
						"role":         "tool",
						"tool_call_id": tr["tool_call_id"],
						"content":      tr["content"],
					})
				}

				// Add regular content if present
				if len(openaiContent) > 0 {
					if len(openaiContent) == 1 && openaiContent[0]["type"] == "text" {
						result = append(result, map[string]interface{}{
							"role":    "user",
							"content": openaiContent[0]["text"],
						})
					} else {
						result = append(result, map[string]interface{}{
							"role":    "user",
							"content": openaiContent,
						})
					}
				}
			} else {
				// Other roles
				if len(openaiContent) > 0 {
					if len(openaiContent) == 1 && openaiContent[0]["type"] == "text" {
						result = append(result, map[string]interface{}{
							"role":    role,
							"content": openaiContent[0]["text"],
						})
					} else {
						result = append(result, map[string]interface{}{
							"role":    role,
							"content": openaiContent,
						})
					}
				}
			}
		default:
			// Fallback
			if data, err := json.Marshal(c); err == nil {
				result = append(result, map[string]interface{}{
					"role":    role,
					"content": string(data),
				})
			}
		}
	}

	return result
}

// extractTextContent extracts text content from various formats.
func extractTextContent(content interface{}) string {
	if content == nil {
		return ""
	}

	switch c := content.(type) {
	case string:
		return c
	case []interface{}:
		var parts []string
		for _, item := range c {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if itemMap["type"] == "text" {
					if text, ok := itemMap["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			} else if str, ok := item.(string); ok {
				parts = append(parts, str)
			}
		}
		if len(parts) > 0 {
			result := ""
			for i, p := range parts {
				if i > 0 {
					result += "\n"
				}
				result += p
			}
			return result
		}
	}

	if data, err := json.Marshal(content); err == nil {
		return string(data)
	}
	return ""
}

// ExtractSystemText extracts system text from Anthropic system parameter.
func ExtractSystemText(system interface{}) string {
	if system == nil {
		return ""
	}

	switch s := system.(type) {
	case string:
		return s
	case []interface{}:
		var parts []string
		for _, block := range s {
			if blockMap, ok := block.(map[string]interface{}); ok {
				if blockMap["type"] == "text" {
					if text, ok := blockMap["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			} else if str, ok := block.(string); ok {
				parts = append(parts, str)
			}
		}
		if len(parts) > 0 {
			result := ""
			for i, p := range parts {
				if i > 0 {
					result += "\n"
				}
				result += p
			}
			return result
		}
	}

	return ""
}

// ConvertAnthropicTools converts Anthropic tools format to OpenAI format.
func ConvertAnthropicTools(tools []interface{}) []map[string]interface{} {
	if len(tools) == 0 {
		return nil
	}

	result := make([]map[string]interface{}, 0, len(tools))

	for _, tool := range tools {
		toolMap, ok := tool.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := toolMap["name"].(string)
		if name == "" {
			continue
		}

		description, _ := toolMap["description"].(string)
		inputSchema, _ := toolMap["input_schema"].(map[string]interface{})
		if inputSchema == nil {
			inputSchema = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}

		result = append(result, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        name,
				"description": description,
				"parameters":  inputSchema,
			},
		})
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// ConvertOpenAIResponseToAnthropic converts OpenAI response to Anthropic format.
func ConvertOpenAIResponseToAnthropic(resp map[string]interface{}, model string) map[string]interface{} {
	content := make([]interface{}, 0)
	stopReason := "end_turn"

	choices, _ := resp["choices"].([]interface{})
	if len(choices) > 0 {
		choice := choices[0].(map[string]interface{})
		message, _ := choice["message"].(map[string]interface{})

		// Add text content
		if text, ok := message["content"].(string); ok && text != "" {
			content = append(content, map[string]interface{}{
				"type": "text",
				"text": text,
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
				argsStr, _ := function["arguments"].(string)

				var args map[string]interface{}
				if argsStr != "" {
					json.Unmarshal([]byte(argsStr), &args)
				}
				if args == nil {
					args = map[string]interface{}{}
				}

				content = append(content, map[string]interface{}{
					"type":  "tool_use",
					"id":    id,
					"name":  name,
					"input": args,
				})
			}
			stopReason = "tool_use"
		}

		finishReason, _ := choice["finish_reason"].(string)
		if finishReason == "stop" {
			stopReason = "end_turn"
		} else if finishReason == "length" {
			stopReason = "max_tokens"
		}
	}

	usage, _ := resp["usage"].(map[string]interface{})
	promptTokens, _ := usage["prompt_tokens"].(float64)
	completionTokens, _ := usage["completion_tokens"].(float64)

	promptDetails, _ := usage["prompt_tokens_details"].(map[string]interface{})
	cachedTokens, _ := promptDetails["cached_tokens"].(float64)

	return map[string]interface{}{
		"id":            "msg_" + uuid.New().String(),
		"type":          "message",
		"role":          "assistant",
		"content":       content,
		"model":         model,
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage": map[string]interface{}{
			"input_tokens":               int(promptTokens),
			"output_tokens":              int(completionTokens),
			"cache_creation_input_tokens": 0,
			"cache_read_input_tokens":     int(cachedTokens),
		},
	}
}
