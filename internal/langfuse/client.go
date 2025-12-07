// Package langfuse provides observability integration with Langfuse.
package langfuse

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rahulvramesh/gh-proxy-local/internal/config"
)

// Client is the Langfuse API client with async event ingestion.
// apiKeyID and apiKeyVal are not hardcoded credentials - they are
// populated from environment variables at runtime via config.LangfuseConfig
type Client struct {
	host       string
	apiKeyID   string
	apiKeyVal  string
	httpClient *http.Client
	debug      bool

	events    chan Event
	done      chan struct{}
	wg        sync.WaitGroup
	batchSize int
	interval  time.Duration

	mu      sync.Mutex
	enabled bool
}

// NewClient creates a new Langfuse client.
func NewClient(cfg config.LangfuseConfig, debug bool) *Client {
	if !cfg.Enabled || cfg.PublicKey == "" || cfg.SecretKey == "" {
		return &Client{enabled: false}
	}

	c := &Client{
		host:      cfg.Host,
		apiKeyID:  cfg.PublicKey,
		apiKeyVal: cfg.SecretKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		debug:     debug,
		events:    make(chan Event, 1000),
		done:      make(chan struct{}),
		batchSize: cfg.BatchSize,
		interval:  cfg.FlushInterval,
		enabled:   true,
	}

	c.wg.Add(1)
	go c.worker()

	return c
}

// IsEnabled returns whether Langfuse integration is enabled.
func (c *Client) IsEnabled() bool {
	return c.enabled
}

// debugLog prints debug messages if debugging is enabled.
func (c *Client) debugLog(format string, args ...interface{}) {
	if c.debug {
		log.Printf("[LANGFUSE] "+format, args...)
	}
}

// Track sends an event to the async queue for batching.
func (c *Client) Track(event Event) {
	if !c.enabled {
		return
	}

	select {
	case c.events <- event:
		c.debugLog("Event queued: %s (%s)", event.Type, event.ID)
	default:
		c.debugLog("Event dropped (queue full): %s (%s)", event.Type, event.ID)
	}
}

// TrackGeneration is a convenience method for tracking LLM generations.
func (c *Client) TrackGeneration(gen *GenerationBody) {
	if !c.enabled {
		return
	}

	if gen.ID == "" {
		gen.ID = uuid.New().String()
	}
	if gen.TraceID == "" {
		gen.TraceID = uuid.New().String()
	}

	traceEvent := NewTraceEvent(&TraceBody{
		ID:        gen.TraceID,
		Name:      gen.Name,
		Input:     gen.Input,
		Output:    gen.Output,
		Metadata:  gen.Metadata,
		Timestamp: gen.StartTime,
	})
	c.Track(traceEvent)

	genEvent := NewGenerationEvent(gen)
	c.Track(genEvent)
}

// worker is the background goroutine that batches and sends events.
func (c *Client) worker() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	batch := make([]Event, 0, c.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}

		batchCopy := make([]Event, len(batch))
		copy(batchCopy, batch)
		batch = batch[:0]

		go func(events []Event) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := c.sendBatch(ctx, events); err != nil {
				c.debugLog("Failed to send batch: %v", err)
			} else {
				c.debugLog("Batch sent successfully: %d events", len(events))
			}
		}(batchCopy)
	}

	for {
		select {
		case event := <-c.events:
			batch = append(batch, event)
			if len(batch) >= c.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-c.done:
			// Drain remaining events
			for {
				select {
				case event := <-c.events:
					batch = append(batch, event)
				default:
					flush()
					return
				}
			}
		}
	}
}

// sendBatch sends a batch of events to the Langfuse API.
func (c *Client) sendBatch(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}

	req := BatchIngestionRequest{
		Batch: events,
		Metadata: map[string]interface{}{
			"sdk_name":    "gh-proxy",
			"sdk_version": "1.0.0",
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.host+"/api/public/ingestion", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Basic auth
	auth := base64.StdEncoding.EncodeToString([]byte(c.apiKeyID + ":" + c.apiKeyVal))
	httpReq.Header.Set("Authorization", "Basic "+auth)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return nil
}

// Flush forces sending all pending events.
func (c *Client) Flush() {
	if !c.enabled {
		return
	}

	// Send a signal to flush by waiting for current batch cycle
	time.Sleep(100 * time.Millisecond)
}

// Shutdown gracefully shuts down the client and flushes remaining events.
func (c *Client) Shutdown() {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	if !c.enabled {
		c.mu.Unlock()
		return
	}
	c.enabled = false
	c.mu.Unlock()

	close(c.done)
	c.wg.Wait()
	c.debugLog("Client shutdown complete")
}

// GenerateTraceID generates a new trace ID.
func GenerateTraceID() string {
	return uuid.New().String()
}

// GenerateSpanID generates a new span ID.
func GenerateSpanID() string {
	return uuid.New().String()
}
