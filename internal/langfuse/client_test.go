package langfuse

import (
	"testing"
	"time"

	"github.com/rahulvramesh/gh-proxy-local/internal/config"
)

func TestNewClientDisabled(t *testing.T) {
	cfg := config.LangfuseConfig{
		Enabled: false,
	}
	client := NewClient(cfg, false)
	if client.IsEnabled() {
		t.Error("expected client to be disabled")
	}
}

func TestNewClientMissingKeys(t *testing.T) {
	cfg := config.LangfuseConfig{
		Enabled:   true,
		PublicKey: "",
		SecretKey: "",
	}
	client := NewClient(cfg, false)
	if client.IsEnabled() {
		t.Error("expected client to be disabled when keys are missing")
	}
}

func TestNewTraceEvent(t *testing.T) {
	trace := &TraceBody{
		ID:        "trace-123",
		Name:      "test-trace",
		UserID:    "user-1",
		SessionID: "session-1",
		Input:     "test input",
		Output:    "test output",
		Timestamp: time.Now(),
	}

	event := NewTraceEvent(trace)

	if event.ID != "trace-123" {
		t.Errorf("expected ID trace-123, got %s", event.ID)
	}
	if event.Type != "trace-create" {
		t.Errorf("expected type trace-create, got %s", event.Type)
	}
	if event.Body["name"] != "test-trace" {
		t.Errorf("expected name test-trace, got %v", event.Body["name"])
	}
}

func TestNewGenerationEvent(t *testing.T) {
	gen := &GenerationBody{
		ID:      "gen-123",
		TraceID: "trace-123",
		Name:    "test-generation",
		Model:   "gpt-4",
		Input:   []map[string]string{{"role": "user", "content": "hello"}},
		Output:  map[string]string{"role": "assistant", "content": "hi"},
		Usage: &UsageData{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Second),
	}

	event := NewGenerationEvent(gen)

	if event.ID != "gen-123" {
		t.Errorf("expected ID gen-123, got %s", event.ID)
	}
	if event.Type != "generation-create" {
		t.Errorf("expected type generation-create, got %s", event.Type)
	}
	if event.Body["model"] != "gpt-4" {
		t.Errorf("expected model gpt-4, got %v", event.Body["model"])
	}
	if event.Body["traceId"] != "trace-123" {
		t.Errorf("expected traceId trace-123, got %v", event.Body["traceId"])
	}
}

func TestTrackDisabledClient(t *testing.T) {
	cfg := config.LangfuseConfig{
		Enabled: false,
	}
	client := NewClient(cfg, false)

	// Should not panic
	client.Track(Event{ID: "test"})
	client.TrackGeneration(&GenerationBody{})
	client.Flush()
	client.Shutdown()
}

func TestGenerateIDs(t *testing.T) {
	traceID := GenerateTraceID()
	if traceID == "" {
		t.Error("expected non-empty trace ID")
	}

	spanID := GenerateSpanID()
	if spanID == "" {
		t.Error("expected non-empty span ID")
	}

	if traceID == spanID {
		t.Error("expected different IDs")
	}
}
