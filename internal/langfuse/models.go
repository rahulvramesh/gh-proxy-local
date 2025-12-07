// Package langfuse provides observability integration with Langfuse.
package langfuse

import "time"

// Event represents a generic Langfuse event.
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Body      map[string]interface{} `json:"body"`
}

// TraceBody represents the body of a trace-create event.
type TraceBody struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name,omitempty"`
	UserID    string                 `json:"userId,omitempty"`
	SessionID string                 `json:"sessionId,omitempty"`
	Input     interface{}            `json:"input,omitempty"`
	Output    interface{}            `json:"output,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// GenerationBody represents the body of a generation-create event.
type GenerationBody struct {
	ID                   string                 `json:"id"`
	TraceID              string                 `json:"traceId"`
	Name                 string                 `json:"name,omitempty"`
	Model                string                 `json:"model,omitempty"`
	ModelParameters      map[string]interface{} `json:"modelParameters,omitempty"`
	Input                interface{}            `json:"input,omitempty"`
	Output               interface{}            `json:"output,omitempty"`
	Usage                *UsageData             `json:"usage,omitempty"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	StartTime            time.Time              `json:"startTime"`
	EndTime              time.Time              `json:"endTime,omitempty"`
	CompletionStartTime  *time.Time             `json:"completionStartTime,omitempty"`
	Level                string                 `json:"level,omitempty"`
	StatusMessage        string                 `json:"statusMessage,omitempty"`
}

// UsageData represents token usage information.
type UsageData struct {
	PromptTokens     int `json:"promptTokens,omitempty"`
	CompletionTokens int `json:"completionTokens,omitempty"`
	TotalTokens      int `json:"totalTokens,omitempty"`
}

// BatchIngestionRequest represents the batch ingestion API request.
type BatchIngestionRequest struct {
	Batch    []Event                `json:"batch"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// BatchIngestionResponse represents the batch ingestion API response.
type BatchIngestionResponse struct {
	Successes []string `json:"successes"`
	Errors    []struct {
		ID      string `json:"id"`
		Status  int    `json:"status"`
		Message string `json:"message"`
	} `json:"errors"`
}

// GenerationEvent creates a new generation event for the batch.
func NewGenerationEvent(gen *GenerationBody) Event {
	body := map[string]interface{}{
		"id":        gen.ID,
		"traceId":   gen.TraceID,
		"startTime": gen.StartTime.Format(time.RFC3339Nano),
	}

	if gen.Name != "" {
		body["name"] = gen.Name
	}
	if gen.Model != "" {
		body["model"] = gen.Model
	}
	if gen.ModelParameters != nil {
		body["modelParameters"] = gen.ModelParameters
	}
	if gen.Input != nil {
		body["input"] = gen.Input
	}
	if gen.Output != nil {
		body["output"] = gen.Output
	}
	if gen.Usage != nil {
		body["usage"] = map[string]interface{}{
			"promptTokens":     gen.Usage.PromptTokens,
			"completionTokens": gen.Usage.CompletionTokens,
			"totalTokens":      gen.Usage.TotalTokens,
		}
	}
	if gen.Metadata != nil {
		body["metadata"] = gen.Metadata
	}
	if !gen.EndTime.IsZero() {
		body["endTime"] = gen.EndTime.Format(time.RFC3339Nano)
	}
	if gen.CompletionStartTime != nil {
		body["completionStartTime"] = gen.CompletionStartTime.Format(time.RFC3339Nano)
	}
	if gen.Level != "" {
		body["level"] = gen.Level
	}
	if gen.StatusMessage != "" {
		body["statusMessage"] = gen.StatusMessage
	}

	return Event{
		ID:        gen.ID,
		Type:      "generation-create",
		Timestamp: gen.StartTime,
		Body:      body,
	}
}

// NewTraceEvent creates a new trace event for the batch.
func NewTraceEvent(trace *TraceBody) Event {
	body := map[string]interface{}{
		"id":        trace.ID,
		"timestamp": trace.Timestamp.Format(time.RFC3339Nano),
	}

	if trace.Name != "" {
		body["name"] = trace.Name
	}
	if trace.UserID != "" {
		body["userId"] = trace.UserID
	}
	if trace.SessionID != "" {
		body["sessionId"] = trace.SessionID
	}
	if trace.Input != nil {
		body["input"] = trace.Input
	}
	if trace.Output != nil {
		body["output"] = trace.Output
	}
	if trace.Metadata != nil {
		body["metadata"] = trace.Metadata
	}
	if len(trace.Tags) > 0 {
		body["tags"] = trace.Tags
	}

	return Event{
		ID:        trace.ID,
		Type:      "trace-create",
		Timestamp: trace.Timestamp,
		Body:      body,
	}
}
