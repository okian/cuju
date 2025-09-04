package testevents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// HTTPClient wraps http.Client with timeout
type HTTPClient struct {
	client  *http.Client
	timeout time.Duration
}

// newHTTPClient creates a new HTTP client with timeout
func newHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// Get performs a GET request
func (c *HTTPClient) Get(url string) (*http.Response, error) {
	return c.client.Get(url)
}

// Post performs a POST request with JSON body
func (c *HTTPClient) Post(url string, body interface{}) (*http.Response, error) {
	jsonData, err := marshalJSON(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	return c.client.Do(req)
}

// marshalJSON marshals a struct to JSON
func marshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// unmarshalJSON unmarshals JSON to a struct
func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// readResponseBody reads and closes the response body
func readResponseBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// submitEvents submits events concurrently using worker pools
func submitEvents(ctx context.Context, config *Config, events []Event, stats *Stats) error {
	log.Printf("📤 Submitting %d events with %d workers...", len(events), config.Workers)

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
	eventChan := make(chan Event, config.Workers*2)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
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
					case "success":
						atomic.AddInt64(&successful, 1)
					case "duplicate":
						atomic.AddInt64(&duplicate, 1)
					case "failed":
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
							log.Printf("📊 Progress: %d/%d submitted (success: %d, duplicate: %d, failed: %d)",
								total, len(events), succ, dup, fail)
						} else {
							fmt.Printf("\r📤 Submitted: %d/%d (success: %d, duplicate: %d, failed: %d)",
								total, len(events), succ, dup, fail)
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
		fmt.Println() // New line after progress indicator
	}

	// Update stats
	stats.EventsSubmitted = int(atomic.LoadInt64(&submitted))
	stats.EventsSuccessful = int(atomic.LoadInt64(&successful))
	stats.EventsDuplicate = int(atomic.LoadInt64(&duplicate))
	stats.EventsFailed = int(atomic.LoadInt64(&failed))

	log.Printf(`✅ Event submission completed:
   Successful: %d
   Duplicate: %d
   Failed: %d
`, stats.EventsSuccessful, stats.EventsDuplicate, stats.EventsFailed)

	return nil
}

// submitSingleEvent submits a single event and returns the result
func submitSingleEvent(ctx context.Context, client *HTTPClient, url string, event Event) string {
	resp, err := client.Post(url, event)
	if err != nil {
		return "failed"
	}
	defer resp.Body.Close()

	// Read response body
	body, err := readResponseBody(resp)
	if err != nil {
		return "failed"
	}

	// Parse response based on status code
	switch resp.StatusCode {
	case 202:
		// Accepted - new event
		var ack AckResponse
		if err := unmarshalJSON(body, &ack); err == nil && ack.Status == "accepted" {
			return "success"
		}
		return "success" // Assume success for 202 even if parsing fails
	case 200:
		// OK - duplicate event
		var ack AckResponse
		if err := unmarshalJSON(body, &ack); err == nil && ack.Duplicate {
			return "duplicate"
		}
		return "duplicate" // Assume duplicate for 200 even if parsing fails
	default:
		// Error
		return "failed"
	}
}
