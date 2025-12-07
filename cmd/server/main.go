// GitHub Copilot Proxy Server
//
// An OpenAI and Anthropic API-compatible proxy server that routes requests
// through GitHub Copilot's API.
//
// Usage:
//
//	go run cmd/server/main.go [--port PORT] [--host HOST]
//
// First run will perform OAuth device flow authentication.
// Credentials are cached in ~/.copilot_credentials.json
//
// Environment Variables:
//
//	COPILOT_DEBUG=1     Enable debug logging
//	COPILOT_PORT=8080   Server port (default: 8080)
//	COPILOT_HOST=0.0.0.0  Server host (default: 0.0.0.0)
//	COPILOT_API_KEY=key   API key for bearer auth (optional)
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rahulvramesh/gh-proxy-local/internal/auth"
	"github.com/rahulvramesh/gh-proxy-local/internal/config"
	"github.com/rahulvramesh/gh-proxy-local/internal/copilot"
	"github.com/rahulvramesh/gh-proxy-local/internal/handlers"
	"github.com/rahulvramesh/gh-proxy-local/internal/langfuse"
)

func main() {
	cfg := config.NewConfig()

	// Parse command line flags
	port := flag.Int("port", cfg.Port, "Port to listen on")
	flag.IntVar(port, "p", cfg.Port, "Port to listen on (shorthand)")
	host := flag.String("host", cfg.Host, "Host to bind to")
	flag.StringVar(host, "H", cfg.Host, "Host to bind to (shorthand)")
	flag.Parse()

	cfg.Port = *port
	cfg.Host = *host

	// Initialize auth manager
	authManager := auth.NewManager(cfg)

	// Ensure credentials exist before starting server
	fmt.Println("Checking GitHub Copilot authentication...")
	if _, err := authManager.GetCredentials(); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}
	fmt.Println("Authentication verified!")

	// Initialize Copilot client
	client := copilot.NewClient(authManager, cfg.Debug)

	// Initialize Langfuse client
	langfuseClient := langfuse.NewClient(cfg.Langfuse, cfg.Debug)

	// Initialize handlers
	modelsHandler := handlers.NewModelsHandler(client)
	chatHandler := handlers.NewChatHandler(client, langfuseClient, cfg.Debug)
	responsesHandler := handlers.NewResponsesHandler(client, langfuseClient, cfg.Debug)
	anthropicHandler := handlers.NewAnthropicHandler(client, langfuseClient, cfg.Debug)
	healthHandler := handlers.NewHealthHandler(authManager, client)

	// Set up router
	mux := http.NewServeMux()

	// Health endpoints
	mux.HandleFunc("GET /", healthHandler.Health)
	mux.HandleFunc("GET /health", healthHandler.Health)
	mux.HandleFunc("GET /info", healthHandler.Info)

	// Account endpoints
	mux.HandleFunc("GET /v1/account", healthHandler.Account)
	mux.HandleFunc("GET /account", healthHandler.Account)

	// Models endpoints
	mux.HandleFunc("GET /v1/models", modelsHandler.ListModels)
	mux.HandleFunc("GET /models", modelsHandler.ListModels)
	mux.HandleFunc("GET /v1/models/{model_id}", modelsHandler.GetModel)
	mux.HandleFunc("GET /models/{model_id}", modelsHandler.GetModel)

	// OpenAI chat completions
	mux.HandleFunc("POST /v1/chat/completions", chatHandler.ChatCompletions)
	mux.HandleFunc("POST /chat/completions", chatHandler.ChatCompletions)

	// OpenAI responses API
	mux.HandleFunc("POST /v1/responses", responsesHandler.Responses)
	mux.HandleFunc("POST /responses", responsesHandler.Responses)

	// Anthropic messages API
	mux.HandleFunc("POST /v1/messages", anthropicHandler.Messages)
	mux.HandleFunc("POST /messages", anthropicHandler.Messages)
	mux.HandleFunc("POST /v1/messages/count_tokens", anthropicHandler.CountTokens)
	mux.HandleFunc("POST /messages/count_tokens", anthropicHandler.CountTokens)

	// Anthropic batches (not supported, but handle gracefully)
	mux.HandleFunc("POST /v1/messages/batches", anthropicHandler.Batches)
	mux.HandleFunc("GET /v1/messages/batches", anthropicHandler.Batches)
	mux.HandleFunc("GET /v1/messages/batches/{batch_id}", anthropicHandler.Batches)

	// Add CORS middleware
	handler := apiKeyMiddleware(cfg.APIKey, corsMiddleware(mux))

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second, // Longer for streaming
		IdleTimeout:  120 * time.Second,
	}

	// Print startup info
	fmt.Println()
	fmt.Println("🚀 Starting GitHub Copilot Proxy Server")
	fmt.Printf("   Host: %s\n", cfg.Host)
	fmt.Printf("   Port: %d\n", cfg.Port)
	if cfg.APIKey != "" {
		fmt.Println("   Auth: API key required")
	} else {
		fmt.Println("   Auth: None (open access)")
	}
	if langfuseClient.IsEnabled() {
		fmt.Printf("   Langfuse: Enabled (%s)\n", cfg.Langfuse.Host)
	} else {
		fmt.Println("   Langfuse: Disabled")
	}
	fmt.Println()
	fmt.Println("📡 Endpoints:")
	fmt.Printf("   OpenAI Chat:      http://%s/v1/chat/completions\n", addr)
	fmt.Printf("   OpenAI Responses: http://%s/v1/responses\n", addr)
	fmt.Printf("   Anthropic:        http://%s/v1/messages\n", addr)
	fmt.Printf("   Models:           http://%s/v1/models\n", addr)
	fmt.Printf("   Health:           http://%s/health\n", addr)
	fmt.Println()

	// Graceful shutdown
	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nShutting down server...")

		// Shutdown Langfuse client first
		langfuseClient.Shutdown()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Could not gracefully shutdown: %v\n", err)
		}
		close(done)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}

	<-done
	fmt.Println("Server stopped")
}

// corsMiddleware adds CORS headers to responses.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-API-Key, anthropic-version, anthropic-beta")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// apiKeyMiddleware validates the API key if configured.
func apiKeyMiddleware(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Check Authorization header (Bearer token)
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") && strings.TrimPrefix(auth, "Bearer ") == apiKey {
			next.ServeHTTP(w, r)
			return
		}

		// Check X-API-Key header
		xApiKey := r.Header.Get("X-API-Key")
		if xApiKey == apiKey {
			next.ServeHTTP(w, r)
			return
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
