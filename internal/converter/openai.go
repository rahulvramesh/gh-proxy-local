// Package converter provides message format conversion utilities.
package converter

import (
	"encoding/json"
)

// ConvertOpenAIToCopilotMessages converts OpenAI message format to Copilot format.
func ConvertOpenAIToCopilotMessages(messages []map[string]interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(messages))

	for _, msg := range messages {
		converted := make(map[string]interface{})
		converted["role"] = msg["role"]

		// Handle content
		if content, ok := msg["content"]; ok {
			switch c := content.(type) {
			case string:
				converted["content"] = c
			case []interface{}:
				// Handle multimodal content
				converted["content"] = c
			case nil:
				converted["content"] = nil
			default:
				// Try to convert to string via JSON
				if data, err := json.Marshal(c); err == nil {
					converted["content"] = string(data)
				} else {
					converted["content"] = ""
				}
			}
		}

		// Handle tool calls
		if toolCalls, ok := msg["tool_calls"]; ok && toolCalls != nil {
			converted["tool_calls"] = toolCalls
		}

		// Handle tool response
		if toolCallID, ok := msg["tool_call_id"]; ok && toolCallID != nil {
			converted["tool_call_id"] = toolCallID
		}

		result = append(result, converted)
	}

	return result
}

// ConvertOpenAITools converts OpenAI tool format for Copilot.
func ConvertOpenAITools(tools []interface{}) []map[string]interface{} {
	if len(tools) == 0 {
		return nil
	}

	result := make([]map[string]interface{}, 0, len(tools))

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

// NormalizeContentForCopilot normalizes content to Copilot-compatible format.
func NormalizeContentForCopilot(content interface{}) interface{} {
	if content == nil {
		return ""
	}

	switch c := content.(type) {
	case string:
		return c
	case []interface{}:
		normalized := make([]map[string]interface{}, 0, len(c))

		for _, item := range c {
			switch i := item.(type) {
			case string:
				normalized = append(normalized, map[string]interface{}{
					"type": "text",
					"text": i,
				})
			case map[string]interface{}:
				itemType, _ := i["type"].(string)

				switch itemType {
				case "input_text", "output_text", "text":
					text, _ := i["text"].(string)
					normalized = append(normalized, map[string]interface{}{
						"type": "text",
						"text": text,
					})
				case "image_url":
					normalized = append(normalized, i)
				case "input_image":
					imageURL, _ := i["image_url"].(string)
					if imageURL == "" {
						if urlMap, ok := i["image_url"].(map[string]interface{}); ok {
							imageURL, _ = urlMap["url"].(string)
						}
					}
					if imageURL == "" {
						imageURL, _ = i["url"].(string)
					}
					normalized = append(normalized, map[string]interface{}{
						"type": "image_url",
						"image_url": map[string]interface{}{
							"url": imageURL,
						},
					})
				case "image":
					source, _ := i["source"].(map[string]interface{})
					if sourceType, _ := source["type"].(string); sourceType == "base64" {
						mediaType, _ := source["media_type"].(string)
						if mediaType == "" {
							mediaType = "image/png"
						}
						data, _ := source["data"].(string)
						dataURL := "data:" + mediaType + ";base64," + data
						normalized = append(normalized, map[string]interface{}{
							"type": "image_url",
							"image_url": map[string]interface{}{
								"url": dataURL,
							},
						})
					} else if url, ok := i["url"].(string); ok {
						normalized = append(normalized, map[string]interface{}{
							"type": "image_url",
							"image_url": map[string]interface{}{
								"url": url,
							},
						})
					}
				default:
					// Fallback - if item has text field, treat as text
					if text, ok := i["text"].(string); ok {
						normalized = append(normalized, map[string]interface{}{
							"type": "text",
							"text": text,
						})
					} else {
						// Convert to string
						if data, err := json.Marshal(i); err == nil {
							normalized = append(normalized, map[string]interface{}{
								"type": "text",
								"text": string(data),
							})
						}
					}
				}
			}
		}

		// Simplify if only one text item
		if len(normalized) == 1 {
			if normalized[0]["type"] == "text" {
				return normalized[0]["text"]
			}
		}
		return normalized
	default:
		// Fallback
		if data, err := json.Marshal(c); err == nil {
			return string(data)
		}
		return ""
	}
}

// ConvertResponsesInputToMessages converts OpenAI Responses API input to chat messages.
func ConvertResponsesInputToMessages(input interface{}, instructions string) []map[string]interface{} {
	messages := make([]map[string]interface{}, 0)

	// Add system instructions if provided
	if instructions != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": instructions,
		})
	}

	// Handle different input formats
	switch inp := input.(type) {
	case string:
		messages = append(messages, map[string]interface{}{
			"role":    "user",
			"content": inp,
		})
	case []interface{}:
		for _, item := range inp {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				if str, ok := item.(string); ok {
					messages = append(messages, map[string]interface{}{
						"role":    "user",
						"content": str,
					})
				}
				continue
			}

			itemType, _ := itemMap["type"].(string)
			role, _ := itemMap["role"].(string)
			if role == "" {
				role = "user"
			}

			switch itemType {
			case "message":
				content := itemMap["content"]
				normalizedContent := NormalizeContentForCopilot(content)
				messages = append(messages, map[string]interface{}{
					"role":    role,
					"content": normalizedContent,
				})
			case "input_text":
				text, _ := itemMap["text"].(string)
				messages = append(messages, map[string]interface{}{
					"role":    "user",
					"content": text,
				})
			case "output_text":
				text, _ := itemMap["text"].(string)
				messages = append(messages, map[string]interface{}{
					"role":    "assistant",
					"content": text,
				})
			default:
				if content, ok := itemMap["content"]; ok {
					normalizedContent := NormalizeContentForCopilot(content)
					messages = append(messages, map[string]interface{}{
						"role":    role,
						"content": normalizedContent,
					})
				} else if text, ok := itemMap["text"].(string); ok {
					messages = append(messages, map[string]interface{}{
						"role":    role,
						"content": text,
					})
				} else {
					if data, err := json.Marshal(itemMap); err == nil {
						messages = append(messages, map[string]interface{}{
							"role":    role,
							"content": string(data),
						})
					}
				}
			}
		}
	default:
		// Fallback: convert to string
		if data, err := json.Marshal(inp); err == nil {
			messages = append(messages, map[string]interface{}{
				"role":    "user",
				"content": string(data),
			})
		}
	}

	return messages
}
