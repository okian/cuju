package main

import (
	"context"
	"flag"
	"os"
	"runtime"
	"time"

	"github.com/okian/cuju/internal/testevents"
)

// Default configuration constants.
const (
	defaultNumEvents   = 10000
	defaultTopN        = 50
	defaultWorkers     = 2 // multiplier for runtime.NumCPU()
	defaultTimeout     = 30 * time.Second
	defaultTestTimeout = 10 * time.Minute
)

func main() {
	var (
		baseURL    = flag.String("url", "http://localhost:9080", "Base URL of the service")
		numEvents  = flag.Int("events", defaultNumEvents, "Number of events to generate and submit")
		topN       = flag.Int("top", defaultTopN, "Number of top entries to fetch from leaderboard")
		workers    = flag.Int("workers", runtime.NumCPU()*defaultWorkers, "Number of concurrent workers")
		timeout    = flag.Duration("timeout", defaultTimeout, "HTTP request timeout")
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
		os.Stderr.WriteString("Failed to setup logging: " + err.Error() + "\n")
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
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
		os.Stderr.WriteString("Test failed: " + err.Error() + "\n")
		return
	}
}
