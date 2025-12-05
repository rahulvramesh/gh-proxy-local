package config

import (
	"os"
	"testing"
)

func TestNewConfig(t *testing.T) {
	// Test default values
	os.Unsetenv("COPILOT_PORT")
	os.Unsetenv("COPILOT_HOST")
	os.Unsetenv("COPILOT_DEBUG")

	cfg := NewConfig()

	if cfg.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Port)
	}

	if cfg.Host != "0.0.0.0" {
		t.Errorf("Expected default host 0.0.0.0, got %s", cfg.Host)
	}

	if cfg.Debug != false {
		t.Errorf("Expected debug false by default, got %v", cfg.Debug)
	}
}

func TestNewConfigWithEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("COPILOT_PORT", "9000")
	os.Setenv("COPILOT_HOST", "127.0.0.1")
	os.Setenv("COPILOT_DEBUG", "1")
	defer func() {
		os.Unsetenv("COPILOT_PORT")
		os.Unsetenv("COPILOT_HOST")
		os.Unsetenv("COPILOT_DEBUG")
	}()

	cfg := NewConfig()

	if cfg.Port != 9000 {
		t.Errorf("Expected port 9000, got %d", cfg.Port)
	}

	if cfg.Host != "127.0.0.1" {
		t.Errorf("Expected host 127.0.0.1, got %s", cfg.Host)
	}

	if cfg.Debug != true {
		t.Errorf("Expected debug true, got %v", cfg.Debug)
	}
}

func TestNewConfigInvalidPort(t *testing.T) {
	os.Setenv("COPILOT_PORT", "invalid")
	defer os.Unsetenv("COPILOT_PORT")

	cfg := NewConfig()

	// Should fall back to default
	if cfg.Port != 8080 {
		t.Errorf("Expected default port 8080 for invalid input, got %d", cfg.Port)
	}
}

func TestCopilotHeaders(t *testing.T) {
	if len(CopilotHeaders) == 0 {
		t.Error("Expected CopilotHeaders to have values")
	}

	if CopilotHeaders["User-Agent"] == "" {
		t.Error("Expected User-Agent header to be set")
	}
}

func TestConstants(t *testing.T) {
	if ClientID == "" {
		t.Error("Expected ClientID to be set")
	}

	if DeviceCodeURL == "" {
		t.Error("Expected DeviceCodeURL to be set")
	}

	if AccessTokenURL == "" {
		t.Error("Expected AccessTokenURL to be set")
	}

	if CopilotTokenURL == "" {
		t.Error("Expected CopilotTokenURL to be set")
	}

	if CopilotAPIBase == "" {
		t.Error("Expected CopilotAPIBase to be set")
	}

	if ModelsCacheTTL <= 0 {
		t.Error("Expected ModelsCacheTTL to be positive")
	}

	if RefreshBufferMS <= 0 {
		t.Error("Expected RefreshBufferMS to be positive")
	}
}
