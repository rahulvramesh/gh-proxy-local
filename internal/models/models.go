// Package models provides data structures for API requests and responses.
package models

import "encoding/json"

// Credentials represents stored credentials for GitHub Copilot.
type Credentials struct {
	GitHubToken    string `json:"github_token"`
	CopilotToken   string `json:"copilot_token,omitempty"`
	CopilotExpires int64  `json:"copilot_expires,omitempty"`
}

// CopilotTokenResponse represents the response from Copilot token endpoint.
type CopilotTokenResponse struct {
	Token            string                 `json:"token"`
	ExpiresAt        int64                  `json:"expires_at"`
	SKU              string                 `json:"sku,omitempty"`
	Individual       bool                   `json:"individual,omitempty"`
	ChatEnabled      bool                   `json:"chat_enabled,omitempty"`
	CodeReviewEnabled bool                  `json:"code_review_enabled,omitempty"`
	Codesearch       bool                   `json:"codesearch,omitempty"`
	Telemetry        string                 `json:"telemetry,omitempty"`
	Endpoints        map[string]interface{} `json:"endpoints,omitempty"`
	LimitedUserQuotas interface{}           `json:"limited_user_quotas,omitempty"`
	LimitedUserResetDate string             `json:"limited_user_reset_date,omitempty"`
	PublicSuggestions string                `json:"public_suggestions,omitempty"`
}

// DeviceCodeResponse represents GitHub OAuth device code response.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// AccessTokenResponse represents GitHub OAuth access token response.
type AccessTokenResponse struct {
	AccessToken      string `json:"access_token,omitempty"`
	TokenType        string `json:"token_type,omitempty"`
	Scope            string `json:"scope,omitempty"`
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// CopilotModel represents a model from Copilot API.
type CopilotModel struct {
	ID           string            `json:"id"`
	Object       string            `json:"object"`
	Created      int64             `json:"created"`
	OwnedBy      string            `json:"owned_by"`
	Name         string            `json:"name,omitempty"`
	Version      string            `json:"version,omitempty"`
	Preview      bool              `json:"preview,omitempty"`
	Limits       *ModelLimits      `json:"limits,omitempty"`
	Capabilities *ModelCapabilities `json:"capabilities,omitempty"`
}

// ModelLimits represents model token limits.
type ModelLimits struct {
	MaxContextWindowTokens int `json:"max_context_window_tokens,omitempty"`
	MaxOutputTokens        int `json:"max_output_tokens,omitempty"`
	MaxPromptTokens        int `json:"max_prompt_tokens,omitempty"`
}

// ModelCapabilities represents model capabilities.
type ModelCapabilities struct {
	Vision             bool `json:"vision,omitempty"`
	ToolCalls          bool `json:"tool_calls,omitempty"`
	ParallelToolCalls  bool `json:"parallel_tool_calls,omitempty"`
	Streaming          bool `json:"streaming,omitempty"`
	StructuredOutputs  bool `json:"structured_outputs,omitempty"`
}

// CopilotModelsResponse represents the models list response from Copilot API.
type CopilotModelsResponse struct {
	Data []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Vendor       string `json:"vendor"`
		Version      string `json:"version"`
		Preview      bool   `json:"preview"`
		Capabilities struct {
			Limits struct {
				MaxContextWindowTokens int `json:"max_context_window_tokens"`
				MaxOutputTokens        int `json:"max_output_tokens"`
				MaxPromptTokens        int `json:"max_prompt_tokens"`
			} `json:"limits"`
			Supports struct {
				Vision            bool `json:"vision"`
				ToolCalls         bool `json:"tool_calls"`
				ParallelToolCalls bool `json:"parallel_tool_calls"`
				Streaming         bool `json:"streaming"`
				StructuredOutputs bool `json:"structured_outputs"`
			} `json:"supports"`
		} `json:"capabilities"`
	} `json:"data"`
}

// ChatMessage represents an OpenAI chat message.
type ChatMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content,omitempty"`
	Name       string          `json:"name,omitempty"`
	ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

// ToolFunction represents a function in a tool.
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// Tool represents a tool definition.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// OpenAIChatRequest represents an OpenAI chat completion request.
type OpenAIChatRequest struct {
	Model            string        `json:"model"`
	Messages         []ChatMessage `json:"messages"`
	Temperature      *float64      `json:"temperature,omitempty"`
	MaxTokens        *int          `json:"max_tokens,omitempty"`
	Stream           bool          `json:"stream,omitempty"`
	Tools            []Tool        `json:"tools,omitempty"`
	ToolChoice       interface{}   `json:"tool_choice,omitempty"`
	TopP             *float64      `json:"top_p,omitempty"`
	FrequencyPenalty *float64      `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64      `json:"presence_penalty,omitempty"`
	Stop             interface{}   `json:"stop,omitempty"`
	N                *int          `json:"n,omitempty"`
	User             string        `json:"user,omitempty"`
}

// OpenAIChatResponse represents an OpenAI chat completion response.
type OpenAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model,omitempty"`
	Choices []struct {
		Index        int `json:"index"`
		Message      struct {
			Role       string          `json:"role"`
			Content    string          `json:"content,omitempty"`
			ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Usage *Usage `json:"usage,omitempty"`
}

// Usage represents token usage information.
type Usage struct {
	PromptTokens            int `json:"prompt_tokens"`
	CompletionTokens        int `json:"completion_tokens"`
	TotalTokens             int `json:"total_tokens"`
	PromptTokensDetails     *TokenDetails `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *TokenDetails `json:"completion_tokens_details,omitempty"`
}

// TokenDetails represents detailed token information.
type TokenDetails struct {
	CachedTokens              int `json:"cached_tokens,omitempty"`
	AcceptedPredictionTokens  int `json:"accepted_prediction_tokens,omitempty"`
	RejectedPredictionTokens  int `json:"rejected_prediction_tokens,omitempty"`
}

// OpenAIResponsesRequest represents an OpenAI Responses API request.
type OpenAIResponsesRequest struct {
	Model           string      `json:"model"`
	Input           interface{} `json:"input"`
	Instructions    string      `json:"instructions,omitempty"`
	Temperature     *float64    `json:"temperature,omitempty"`
	MaxOutputTokens *int        `json:"max_output_tokens,omitempty"`
	Stream          bool        `json:"stream,omitempty"`
	Tools           []interface{} `json:"tools,omitempty"`
	ToolChoice      interface{} `json:"tool_choice,omitempty"`
	TopP            *float64    `json:"top_p,omitempty"`
	Store           *bool       `json:"store,omitempty"`
	Metadata        interface{} `json:"metadata,omitempty"`
	Truncation      string      `json:"truncation,omitempty"`
}

// AnthropicMessage represents an Anthropic API message.
type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// AnthropicRequest represents an Anthropic messages API request.
type AnthropicRequest struct {
	Model         string             `json:"model"`
	Messages      []AnthropicMessage `json:"messages"`
	MaxTokens     int                `json:"max_tokens"`
	Temperature   *float64           `json:"temperature,omitempty"`
	System        interface{}        `json:"system,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
	Tools         []interface{}      `json:"tools,omitempty"`
	ToolChoice    interface{}        `json:"tool_choice,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
	TopP          *float64           `json:"top_p,omitempty"`
	TopK          *int               `json:"top_k,omitempty"`
	Metadata      interface{}        `json:"metadata,omitempty"`
	Thinking      interface{}        `json:"thinking,omitempty"`
}

// AnthropicResponse represents an Anthropic messages API response.
type AnthropicResponse struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	Role         string        `json:"role"`
	Content      []interface{} `json:"content"`
	Model        string        `json:"model"`
	StopReason   string        `json:"stop_reason"`
	StopSequence *string       `json:"stop_sequence"`
	Usage        *AnthropicUsage `json:"usage"`
}

// AnthropicUsage represents Anthropic usage information.
type AnthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// GitHubUser represents GitHub user information.
type GitHubUser struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	HTMLURL   string `json:"html_url,omitempty"`
	Type      string `json:"type,omitempty"`
	Plan      interface{} `json:"plan,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}
