// Package config provides configuration management for the proxy server.
package config

import (
	"os"
	"strconv"
	"time"
)

const (
	// GitHub OAuth client ID for Copilot
	ClientID = "Iv1.b507a08c87ecfe98"

	// API endpoints
	DeviceCodeURL   = "https://github.com/login/device/code"
	AccessTokenURL  = "https://github.com/login/oauth/access_token"
	CopilotTokenURL = "https://api.github.com/copilot_internal/v2/token"
	CopilotAPIBase  = "https://api.githubcopilot.com"
	GitHubUserURL   = "https://api.github.com/user"

	// Cache TTL
	ModelsCacheTTL = 300 // 5 minutes in seconds

	// Token refresh buffer (5 minutes in milliseconds)
	RefreshBufferMS = 5 * 60 * 1000
)

// CopilotHeaders returns the standard headers for Copilot API requests.
var CopilotHeaders = map[string]string{
	"User-Agent":              "GitHubCopilotChat/0.32.4",
	"Editor-Version":          "vscode/1.105.1",
	"Editor-Plugin-Version":   "copilot-chat/0.32.4",
	"Copilot-Integration-Id":  "vscode-chat",
}

// Config holds the server configuration.
type Config struct {
	Host            string
	Port            int
	Debug           bool
	CredentialsFile string
	APIKey          string
	Langfuse        LangfuseConfig
}

// LangfuseConfig holds Langfuse observability configuration.
type LangfuseConfig struct {
	Enabled       bool
	Host          string
	PublicKey     string
	SecretKey     string
	BatchSize     int
	FlushInterval time.Duration
}

// NewConfig creates a new configuration from environment variables.
func NewConfig() *Config {
	port := 8080
	if p := os.Getenv("COPILOT_PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			port = parsed
		}
	}

	host := "0.0.0.0"
	if h := os.Getenv("COPILOT_HOST"); h != "" {
		host = h
	}

	debug := false
	if d := os.Getenv("COPILOT_DEBUG"); d == "1" || d == "true" || d == "yes" {
		debug = true
	}

	homeDir, _ := os.UserHomeDir()
	credFile := homeDir + "/.copilot_credentials.json"

	apiKey := os.Getenv("COPILOT_API_KEY")

	// Langfuse configuration
	langfuseEnabled := false
	if lf := os.Getenv("LANGFUSE_ENABLED"); lf == "1" || lf == "true" || lf == "yes" {
		langfuseEnabled = true
	}

	langfuseHost := "https://cloud.langfuse.com"
	if lh := os.Getenv("LANGFUSE_HOST"); lh != "" {
		langfuseHost = lh
	}

	langfuseBatchSize := 10
	if bs := os.Getenv("LANGFUSE_BATCH_SIZE"); bs != "" {
		if parsed, err := strconv.Atoi(bs); err == nil && parsed > 0 {
			langfuseBatchSize = parsed
		}
	}

	langfuseFlushInterval := 5 * time.Second
	if fi := os.Getenv("LANGFUSE_FLUSH_INTERVAL"); fi != "" {
		if parsed, err := time.ParseDuration(fi); err == nil && parsed > 0 {
			langfuseFlushInterval = parsed
		}
	}

	return &Config{
		Host:            host,
		Port:            port,
		Debug:           debug,
		CredentialsFile: credFile,
		APIKey:          apiKey,
		Langfuse: LangfuseConfig{
			Enabled:       langfuseEnabled,
			Host:          langfuseHost,
			PublicKey:     os.Getenv("LANGFUSE_PUBLIC_KEY"),
			SecretKey:     os.Getenv("LANGFUSE_SECRET_KEY"),
			BatchSize:     langfuseBatchSize,
			FlushInterval: langfuseFlushInterval,
		},
	}
}
