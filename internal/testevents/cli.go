package testevents

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// SetupLogging configures logging to both console and file.
// If logFile is empty, a timestamped filename is generated.
func SetupLogging(logFile string) error {
	if logFile == "" {
		timestamp := time.Now().Format("20060102_150405")
		logFile = fmt.Sprintf("test_log_%s.log", timestamp)
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600) //nolint:gosec // log file permissions
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	multiWriter := io.MultiWriter(os.Stdout, file)
	log.SetOutput(multiWriter)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("📝 Logging to file: %s", logFile)
	return nil
}

// ShowHelp prints usage information for the test events tool.
func ShowHelp() {
	log.Print(`Cuju Event Test Tool
===================

A high-performance concurrent tool for testing the Cuju event processing system.

Usage:
  go run cmd/test-events/main.go [options]

Options:
  -url string
        Base URL of the service (default "http://localhost:9080")
  -events int
        Number of events to generate and submit (default 10000)
  -top int
        Number of top entries to fetch from leaderboard (default 50)
  -workers int
        Number of concurrent workers (default CPU cores * 2)
  -timeout duration
        HTTP request timeout (default 30s)
  -output string
        Output file for generated events (default: generated_events_TIMESTAMP.json)
  -log string
        Log file for test output (default: test_log_TIMESTAMP.log)
  -verbose
        Enable verbose logging
  -help
        Show this help message

Examples:
  # Test with default settings
  go run cmd/test-events/main.go

  # Test with custom parameters
  go run cmd/test-events/main.go -events 50000 -workers 16 -url http://localhost:8080

  # Test with verbose output
  go run cmd/test-events/main.go -verbose -events 10000

  # Test with custom log file
  go run cmd/test-events/main.go -events 50000 -log my_test.log
`)
}
