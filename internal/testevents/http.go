package testevents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/okian/cuju/pkg/logger"
)

// Constants for event submission results.
const (
	ResultSuccess   = "success"
	ResultDuplicate = "duplicate"
	ResultFailed    = "failed"
)

// HTTPClient wraps http.Client with timeout.
type HTTPClient struct {
	client  *http.Client
	timeout time.Duration
}

// newHTTPClient creates a new HTTP client with timeout.
func newHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// Get performs a GET request with context.
func (c *HTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	return resp, nil
}

// Post performs a POST request with JSON body and context.
func (c *HTTPClient) Post(ctx context.Context, url string, body interface{}) (*http.Response, error) {
	jsonData, err := marshalJSON(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	return resp, nil
}

// marshalJSON marshals a struct to JSON.
func marshalJSON(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return data, nil
}

// unmarshalJSON unmarshals JSON to a struct.
func unmarshalJSON(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return nil
}

// readResponseBody reads and closes the response body.
func readResponseBody(resp *http.Response) ([]byte, error) {
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Get().Error(context.Background(), "failed to close response body", logger.Error(err))
		}
	}()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return data, nil
}

// submitEvents submits events concurrently using worker pools.
func submitEvents(ctx context.Context, config *Config, events []Event, stats *Stats) error { //nolint:gocognit,unparam // complex function required for concurrent event submission, error return for consistency
	logger.Get().Info(ctx, "submitting events", logger.Int("events", len(events)), logger.Int("workers", config.Workers))

	client := newHTTPClient(config.Timeout)
	url := config.BaseURL + "/events"

	// Counters for statistics
	var (
		successful int64
		duplicate  int64
		failed     int64
		submitted  int64
	)

	// Progress reporting
	var lastReport time.Time
	reportInterval := 1 * time.Second

	// Create worker pool
	eventChan := make(chan Event, config.Workers*WorkerChannelMultiplier)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()

			for event := range eventChan {
				select {
				case <-ctx.Done():
					return
				default:
					result := submitSingleEvent(ctx, client, url, event)

					// Update counters
					atomic.AddInt64(&submitted, 1)
					switch result {
					case ResultSuccess:
						atomic.AddInt64(&successful, 1)
					case ResultDuplicate:
						atomic.AddInt64(&duplicate, 1)
					case ResultFailed:
						atomic.AddInt64(&failed, 1)
					}

					// Progress reporting
					if time.Since(lastReport) >= reportInterval {
						lastReport = time.Now()
						total := atomic.LoadInt64(&submitted)
						succ := atomic.LoadInt64(&successful)
						dup := atomic.LoadInt64(&duplicate)
						fail := atomic.LoadInt64(&failed)

						if config.Verbose {
							logger.Get().Info(ctx, "progress update",
								logger.Int("total", int(total)),
								logger.Int("events", len(events)),
								logger.Int("success", int(succ)),
								logger.Int("duplicate", int(dup)),
								logger.Int("failed", int(fail)))
						} else {
							logger.Get().Info(ctx, "submission progress",
								logger.Int("total", int(total)),
								logger.Int("events", len(events)),
								logger.Int("success", int(succ)),
								logger.Int("duplicate", int(dup)),
								logger.Int("failed", int(fail)))
						}
					}
				}
			}
		}(i)
	}

	// Send events to workers
	go func() {
		defer close(eventChan)
		for _, event := range events {
			select {
			case <-ctx.Done():
				return
			case eventChan <- event:
			}
		}
	}()

	// Wait for all workers to complete
	wg.Wait()

	// Final progress report
	if !config.Verbose {
		logger.Get().Info(ctx, "progress indicator complete")
	}

	// Update stats
	stats.EventsSubmitted = int(atomic.LoadInt64(&submitted))
	stats.EventsSuccessful = int(atomic.LoadInt64(&successful))
	stats.EventsDuplicate = int(atomic.LoadInt64(&duplicate))
	stats.EventsFailed = int(atomic.LoadInt64(&failed))

	logger.Get().Info(ctx, "event submission completed",
		logger.Int("successful", stats.EventsSuccessful),
		logger.Int("duplicate", stats.EventsDuplicate),
		logger.Int("failed", stats.EventsFailed))

	return nil
}

// submitSingleEvent submits a single event and returns the result.
func submitSingleEvent(ctx context.Context, client *HTTPClient, url string, event Event) string {
	resp, err := client.Post(ctx, url, event)
	if err != nil {
		return ResultFailed
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Get().Error(context.Background(), "failed to close response body", logger.Error(err))
		}
	}()

	// Read response body
	body, err := readResponseBody(resp)
	if err != nil {
		return ResultFailed
	}

	// Parse response based on status code
	switch resp.StatusCode {
	case StatusAccepted:
		// Accepted - new event
		var ack AckResponse
		if err := unmarshalJSON(body, &ack); err == nil && ack.Status == "accepted" {
			return "success"
		}
		return "success" // Assume success for 202 even if parsing fails
	case StatusOK:
		// OK - duplicate event
		var ack AckResponse
		if err := unmarshalJSON(body, &ack); err == nil && ack.Duplicate {
			return "duplicate"
		}
		return "duplicate" // Assume duplicate for 200 even if parsing fails
	default:
		// Error
		return ResultFailed
	}
}
