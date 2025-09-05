package testevents

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Run executes the complete event test.
func Run(ctx context.Context, config *Config) error {
	stats := &Stats{
		StartTime: time.Now(),
	}

	log.Printf(`🚀 Starting Cuju Event Test
📊 Configuration:
   Base URL: %s
   Events: %d
   Workers: %d
   Timeout: %v
   Top N: %d
   Log File: %s
   Verbose: %t

`, config.BaseURL, config.NumEvents, config.Workers, config.Timeout, config.TopN, config.LogFile, config.Verbose)

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
	log.Println("⏳ Waiting for events to be processed...")
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
		log.Printf("⚠️  Warning: Failed to save events to file: %v", err)
	}

	// Final statistics
	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	printFinalStats(stats)

	log.Println("✅ Test completed successfully!")
	return nil
}

// checkServiceHealth verifies the service is running.
func checkServiceHealth(ctx context.Context, config *Config) error {
	log.Println("🔍 Checking service health...")

	client := newHTTPClient(config.Timeout)
	url := config.BaseURL + "/healthz"

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to service: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	// Accept any 200 response as healthy (the service returns Prometheus metrics)
	if resp.StatusCode != StatusOK {
		return fmt.Errorf("service health check failed with status: %d", resp.StatusCode)
	}

	log.Println("✅ Service is healthy")
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
		filename = fmt.Sprintf("generated_events_%s.json", timestamp)
	}

	// Ensure the directory exists
	dir := filepath.Dir(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, 0750); err != nil { //nolint:gosec // directory permissions
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Write events to file
	file, err := os.Create(filename) //nolint:gosec // file creation for test output
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("failed to close file: %v", err)
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

	log.Printf("💾 Events saved to: %s", filename)
	return nil
}

// printFinalStats prints the final test statistics.
func printFinalStats(stats *Stats) {
	var successRate, eventsPerSecond float64
	var successRateStr, eventsPerSecondStr string

	if stats.EventsSubmitted > 0 {
		successRate = float64(stats.EventsSuccessful) / float64(stats.EventsSubmitted) * PercentageMultiplier
		successRateStr = fmt.Sprintf("   Success Rate: %.2f%%\n", successRate)
	}

	if stats.Duration > 0 {
		eventsPerSecond = float64(stats.EventsSubmitted) / stats.Duration.Seconds()
		eventsPerSecondStr = fmt.Sprintf("   Events/Second: %.2f\n", eventsPerSecond)
	}

	log.Printf(`
📈 === FINAL STATISTICS ===
   Events Generated: %d
   Events Submitted: %d
   Events Successful: %d
   Events Duplicate: %d
   Events Failed: %d
   Rankings Retrieved: %d
   Leaderboard Entries: %d
   Total Duration: %v
%s%s
`, stats.EventsGenerated, stats.EventsSubmitted, stats.EventsSuccessful,
		stats.EventsDuplicate, stats.EventsFailed, stats.RankingsRetrieved,
		stats.LeaderboardEntries, stats.Duration, successRateStr, eventsPerSecondStr)
}
