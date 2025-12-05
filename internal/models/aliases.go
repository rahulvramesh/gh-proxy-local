package models

// ModelAliases maps common model names to Copilot equivalents.
var ModelAliases = map[string]string{
	// Anthropic SDK model names -> Copilot names
	"claude-3-opus-20240229":     "claude-opus-4.5",
	"claude-3-sonnet-20240229":   "claude-sonnet-4",
	"claude-3-haiku-20240307":    "claude-haiku-4.5",
	"claude-3-5-sonnet-20240620": "claude-sonnet-4.5",
	"claude-3-5-sonnet-20241022": "claude-sonnet-4.5",
	"claude-3-5-haiku-20241022":  "claude-haiku-4.5",
	"claude-sonnet-4-20250514":   "claude-sonnet-4",
	"claude-opus-4-20250514":     "claude-opus-4.5",
	// OpenAI aliases
	"gpt-4-turbo":         "gpt-4o",
	"gpt-4-turbo-preview": "gpt-4o",
	"gpt-4-1106-preview":  "gpt-4o",
}

// VendorOwnerMap maps Copilot vendor names to owner strings.
var VendorOwnerMap = map[string]string{
	"Azure OpenAI": "openai",
	"OpenAI":       "openai",
	"Anthropic":    "anthropic",
	"Google":       "google",
	"xAI":          "xai",
}

// ResolveModel resolves a model alias to the actual Copilot model name.
func ResolveModel(model string) string {
	if resolved, ok := ModelAliases[model]; ok {
		return resolved
	}
	return model
}

// GetVendorOwner returns the owner string for a vendor name.
func GetVendorOwner(vendor string) string {
	if owner, ok := VendorOwnerMap[vendor]; ok {
		return owner
	}
	return vendor
}

// FallbackModels returns fallback models when API fails.
func FallbackModels() []CopilotModel {
	return []CopilotModel{
		{ID: "gpt-4o", Object: "model", Created: 1700000000, OwnedBy: "openai"},
		{ID: "gpt-4.1", Object: "model", Created: 1700000000, OwnedBy: "openai"},
		{ID: "claude-sonnet-4", Object: "model", Created: 1700000000, OwnedBy: "anthropic"},
		{ID: "claude-sonnet-4.5", Object: "model", Created: 1700000000, OwnedBy: "anthropic"},
		{ID: "gemini-2.5-pro", Object: "model", Created: 1700000000, OwnedBy: "google"},
	}
}
