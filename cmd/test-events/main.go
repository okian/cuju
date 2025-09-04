package main

import (
	"context"
	"flag"
	"log"
	"runtime"
	"time"

	"github.com/okian/cuju/internal/testevents"
)

func main() {
	var (
		baseURL    = flag.String("url", "http://localhost:9080", "Base URL of the service")
		numEvents  = flag.Int("events", 100000, "Number of events to generate and submit")
		topN       = flag.Int("top", 50, "Number of top entries to fetch from leaderboard")
		workers    = flag.Int("workers", runtime.NumCPU()*2, "Number of concurrent workers")
		timeout    = flag.Duration("timeout", 30*time.Second, "HTTP request timeout")
		outputFile = flag.String("output", "", "Output file for generated events (default: generated_events_TIMESTAMP.json)")
		logFile    = flag.String("log", "", "Log file for test output (default: test_log_TIMESTAMP.log)")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
		help       = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help {
		testevents.ShowHelp()
		return
	}

	// Setup logging
	if err := testevents.SetupLogging(*logFile); err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Create test configuration
	config := &testevents.Config{
		BaseURL:    *baseURL,
		NumEvents:  *numEvents,
		TopN:       *topN,
		Workers:    *workers,
		Timeout:    *timeout,
		OutputFile: *outputFile,
		LogFile:    *logFile,
		Verbose:    *verbose,
	}

	// Run the test
	if err := testevents.Run(ctx, config); err != nil {
		log.Fatalf("Test failed: %v", err)
	}
}
