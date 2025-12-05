package models

import (
	"testing"
)

func TestResolveModel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Anthropic SDK model names
		{"claude-3-opus-20240229", "claude-opus-4.5"},
		{"claude-3-sonnet-20240229", "claude-sonnet-4"},
		{"claude-3-haiku-20240307", "claude-haiku-4.5"},
		{"claude-3-5-sonnet-20240620", "claude-sonnet-4.5"},
		{"claude-3-5-sonnet-20241022", "claude-sonnet-4.5"},
		{"claude-3-5-haiku-20241022", "claude-haiku-4.5"},
		{"claude-sonnet-4-20250514", "claude-sonnet-4"},
		{"claude-opus-4-20250514", "claude-opus-4.5"},
		// OpenAI aliases
		{"gpt-4-turbo", "gpt-4o"},
		{"gpt-4-turbo-preview", "gpt-4o"},
		{"gpt-4-1106-preview", "gpt-4o"},
		// Unknown models should pass through
		{"unknown-model", "unknown-model"},
		{"gpt-4o", "gpt-4o"},
		{"claude-sonnet-4", "claude-sonnet-4"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ResolveModel(tt.input)
			if result != tt.expected {
				t.Errorf("ResolveModel(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetVendorOwner(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Azure OpenAI", "openai"},
		{"OpenAI", "openai"},
		{"Anthropic", "anthropic"},
		{"Google", "google"},
		{"xAI", "xai"},
		{"Unknown Vendor", "Unknown Vendor"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := GetVendorOwner(tt.input)
			if result != tt.expected {
				t.Errorf("GetVendorOwner(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFallbackModels(t *testing.T) {
	models := FallbackModels()

	if len(models) == 0 {
		t.Error("FallbackModels should return at least one model")
	}

	// Check that all models have required fields
	for i, m := range models {
		if m.ID == "" {
			t.Errorf("Model %d has empty ID", i)
		}
		if m.Object != "model" {
			t.Errorf("Model %d has incorrect object type: %s", i, m.Object)
		}
		if m.Created == 0 {
			t.Errorf("Model %d has zero created timestamp", i)
		}
		if m.OwnedBy == "" {
			t.Errorf("Model %d has empty owned_by", i)
		}
	}

	// Check for expected models
	expectedModels := map[string]bool{
		"gpt-4o":          false,
		"claude-sonnet-4": false,
	}

	for _, m := range models {
		if _, ok := expectedModels[m.ID]; ok {
			expectedModels[m.ID] = true
		}
	}

	for model, found := range expectedModels {
		if !found {
			t.Errorf("Expected model %q not found in fallback models", model)
		}
	}
}

func TestModelAliasesComplete(t *testing.T) {
	// Ensure commonly used aliases are present
	requiredAliases := []string{
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"gpt-4-turbo",
	}

	for _, alias := range requiredAliases {
		if _, ok := ModelAliases[alias]; !ok {
			t.Errorf("Required alias %q not found in ModelAliases", alias)
		}
	}
}
