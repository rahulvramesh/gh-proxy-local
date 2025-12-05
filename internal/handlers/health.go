package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rahulvramesh/gh-proxy-local/internal/auth"
	"github.com/rahulvramesh/gh-proxy-local/internal/copilot"
)

// HealthHandler handles health and info endpoints.
type HealthHandler struct {
	authManager *auth.Manager
	client      *copilot.Client
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(authManager *auth.Manager, client *copilot.Client) *HealthHandler {
	return &HealthHandler{authManager: authManager, client: client}
}

// Health handles GET / and GET /health
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":  "ok",
		"service": "github-copilot-proxy",
		"version": "1.0.0",
		"endpoints": map[string][]string{
			"openai":    {"/v1/chat/completions", "/v1/responses", "/v1/models"},
			"anthropic": {"/v1/messages"},
			"info":      {"/health", "/info", "/v1/account"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Info handles GET /info
func (h *HealthHandler) Info(w http.ResponseWriter, r *http.Request) {
	creds, err := h.authManager.LoadCredentials()
	authenticated := err == nil && creds != nil && creds.GitHubToken != ""

	var modelIDs []string
	if authenticated {
		models, err := h.client.FetchModels(r.Context())
		if err == nil {
			modelIDs = make([]string, len(models))
			for i, m := range models {
				modelIDs[i] = m.ID
			}
		}
	}

	response := map[string]interface{}{
		"authenticated": authenticated,
		"models":        modelIDs,
		"endpoints": map[string]string{
			"openai_chat":      "/v1/chat/completions",
			"openai_responses": "/v1/responses",
			"openai_models":    "/v1/models",
			"anthropic_messages": "/v1/messages",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Account handles GET /v1/account and GET /account
func (h *HealthHandler) Account(w http.ResponseWriter, r *http.Request) {
	creds, err := h.authManager.GetCredentials()
	if err != nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	// Fetch all available information
	copilotInfo, err := h.authManager.GetCopilotAccountInfo(creds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	githubUser, _ := h.authManager.GetGitHubUser(creds)
	models, _ := h.client.FetchModels(r.Context())

	// Parse subscription details
	sku := copilotInfo.SKU
	subscriptionType := "unknown"
	skuLower := strings.ToLower(sku)
	switch {
	case strings.Contains(skuLower, "free"):
		subscriptionType = "free"
	case strings.Contains(skuLower, "plus"), strings.Contains(skuLower, "pro"):
		subscriptionType = "pro"
	case strings.Contains(skuLower, "business"):
		subscriptionType = "business"
	case strings.Contains(skuLower, "enterprise"):
		subscriptionType = "enterprise"
	}

	// Organize models by vendor
	modelsByVendor := make(map[string][]map[string]interface{})
	for _, m := range models {
		vendor := m.OwnedBy
		if vendor == "" {
			vendor = "unknown"
		}
		if modelsByVendor[vendor] == nil {
			modelsByVendor[vendor] = make([]map[string]interface{}, 0)
		}
		modelsByVendor[vendor] = append(modelsByVendor[vendor], map[string]interface{}{
			"id":           m.ID,
			"name":         m.Name,
			"preview":      m.Preview,
			"limits":       m.Limits,
			"capabilities": m.Capabilities,
		})
	}

	response := map[string]interface{}{
		"user": githubUser,
		"subscription": map[string]interface{}{
			"type":       subscriptionType,
			"sku":        sku,
			"individual": copilotInfo.Individual,
			"features": map[string]interface{}{
				"chat_enabled":        copilotInfo.ChatEnabled,
				"code_review_enabled": copilotInfo.CodeReviewEnabled,
				"codesearch":          copilotInfo.Codesearch,
			},
			"quotas": map[string]interface{}{
				"limited_user_quotas":     copilotInfo.LimitedUserQuotas,
				"limited_user_reset_date": copilotInfo.LimitedUserResetDate,
			},
			"telemetry":          copilotInfo.Telemetry,
			"public_suggestions": copilotInfo.PublicSuggestions,
		},
		"endpoints": copilotInfo.Endpoints,
		"models": map[string]interface{}{
			"total_count": len(models),
			"by_vendor":   modelsByVendor,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
