// Package handlers provides HTTP handlers for the proxy server.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rahulvramesh/gh-proxy-local/internal/copilot"
)

// ModelsHandler handles model-related endpoints.
type ModelsHandler struct {
	client *copilot.Client
}

// NewModelsHandler creates a new models handler.
func NewModelsHandler(client *copilot.Client) *ModelsHandler {
	return &ModelsHandler{client: client}
}

// ListModels handles GET /v1/models and /models
func (h *ModelsHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.client.FetchModels(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"object": "list",
		"data":   models,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetModel handles GET /v1/models/{model_id}
func (h *ModelsHandler) GetModel(w http.ResponseWriter, r *http.Request) {
	modelID := r.PathValue("model_id")
	if modelID == "" {
		http.Error(w, "model_id is required", http.StatusBadRequest)
		return
	}

	models, err := h.client.FetchModels(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, model := range models {
		if model.ID == modelID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(model)
			return
		}
	}

	http.Error(w, "Model not found", http.StatusNotFound)
}
