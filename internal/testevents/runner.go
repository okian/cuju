package testevents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/okian/cuju/pkg/logger"
)

// File permission constants.
const (
	directoryPermission = 0750
)

// Run executes the complete event test.
func Run(ctx context.Context, config *Config) error {
	stats := &Stats{
		StartTime: time.Now(),
	}

	logger.Get().Info(ctx, "starting cuju event test",
		logger.String("baseURL", config.BaseURL),
		logger.Int("events", config.NumEvents),
		logger.Int("workers", config.Workers),
		logger.String("timeout", config.Timeout.String()),
		logger.Int("topN", config.TopN),
		logger.String("logFile", config.LogFile),
		logger.Any("verbose", config.Verbose))

	// Step 1: Check service health
	if err := checkServiceHealth(ctx, config); err != nil {
		return fmt.Errorf("service health check failed: %w", err)
	}

	// Step 2: Generate events
	events, err := generateEvents(ctx, config, stats)
	if err != nil {
		return fmt.Errorf("event generation failed: %w", err)
	}

	// Step 3: Submit events concurrently
	if err := submitEvents(ctx, config, events, stats); err != nil {
		return fmt.Errorf("event submission failed: %w", err)
	}

	// Step 4: Wait for processing
	logger.Get().Info(ctx, "waiting for events to be processed")
	time.Sleep(HealthCheckDelay)

	// Step 5: Retrieve rankings concurrently
	rankings, err := retrieveRankings(ctx, config, events, stats)
	if err != nil {
		return fmt.Errorf("ranking retrieval failed: %w", err)
	}

	// Step 6: Get leaderboard
	leaderboard, err := getLeaderboard(ctx, config, stats)
	if err != nil {
		return fmt.Errorf("leaderboard retrieval failed: %w", err)
	}

	// Step 7: Verify results
	if err := verifyResults(ctx, config, rankings, leaderboard, stats); err != nil {
		return fmt.Errorf("result verification failed: %w", err)
	}

	// Step 8: Save events to file
	if err := saveEventsToFile(ctx, config, events); err != nil {
		logger.Get().Warn(ctx, "failed to save events to file", logger.Error(err))
	}

	// Final statistics
	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	displayFinalStats(stats)

	logger.Get().Info(ctx, "test completed successfully")
	return nil
}

// checkServiceHealth verifies the service is running.
func checkServiceHealth(ctx context.Context, config *Config) error {
	logger.Get().Info(ctx, "checking service health")

	client := newHTTPClient(config.Timeout)
	url := config.BaseURL + "/healthz"

	resp, err := client.Get(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to connect to service: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Get().Error(context.Background(), "failed to close response body", logger.Error(err))
		}
	}()

	// Accept any 200 response as healthy (the service returns Prometheus metrics)
	if resp.StatusCode != StatusOK {
		return fmt.Errorf("service health check failed with status: %d", resp.StatusCode)
	}

	logger.Get().Info(ctx, "service is healthy")
	return nil
}

// saveEventsToFile saves the generated events to a JSON file.
func saveEventsToFile(ctx context.Context, config *Config, events []Event) error {
	if len(events) == 0 {
		return fmt.Errorf("no events to save")
	}

	// Determine output filename
	filename := config.OutputFile
	if filename == "" {
		timestamp := time.Now().Format("20060102_150405")
		filename = "generated_events_" + timestamp + ".json"
	}

	// Ensure the directory exists
	dir := filepath.Dir(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, directoryPermission); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Write events to file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Get().Error(context.Background(), "failed to close file", logger.Error(err))
		}
	}()

	// Write JSON array
	if _, err := file.WriteString("[\n"); err != nil {
		return fmt.Errorf("failed to write opening bracket: %w", err)
	}

	for i, event := range events {
		jsonData, err := marshalJSON(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event %d: %w", i, err)
		}

		if _, err := file.Write(jsonData); err != nil {
			return fmt.Errorf("failed to write event %d: %w", i, err)
		}

		// Add comma except for last event
		if i < len(events)-1 {
			if _, err := file.WriteString(","); err != nil {
				return fmt.Errorf("failed to write comma: %w", err)
			}
		}
		_, _ = file.WriteString("\n")
	}

	if _, err := file.WriteString("]\n"); err != nil {
		return fmt.Errorf("failed to write closing bracket: %w", err)
	}

	logger.Get().Info(ctx, "events saved to file", logger.String("filename", filename))
	return nil
}

// displayFinalStats prints the final test statistics.
func displayFinalStats(stats *Stats) {
	var successRate, eventsPerSecond float64

	if stats.EventsSubmitted > 0 {
		successRate = float64(stats.EventsSuccessful) / float64(stats.EventsSubmitted) * PercentageMultiplier
	}

	if stats.Duration > 0 {
		eventsPerSecond = float64(stats.EventsSubmitted) / stats.Duration.Seconds()
	}

	logger.Get().Info(context.Background(), "final statistics",
		logger.Int("eventsGenerated", stats.EventsGenerated),
		logger.Int("eventsSubmitted", stats.EventsSubmitted),
		logger.Int("eventsSuccessful", stats.EventsSuccessful),
		logger.Int("eventsDuplicate", stats.EventsDuplicate),
		logger.Int("eventsFailed", stats.EventsFailed),
		logger.Int("rankingsRetrieved", stats.RankingsRetrieved),
		logger.Int("leaderboardEntries", stats.LeaderboardEntries),
		logger.String("duration", stats.Duration.String()),
		logger.Float64("successRate", successRate),
		logger.Float64("eventsPerSecond", eventsPerSecond))
}
