package testevents

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/google/uuid"
)

// getRandomFloat returns a random float64 between 0.0 and 1.0 using crypto/rand.
func getRandomFloat() float64 {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return float64(n.Int64()) / 1000000.0
}

// generateEvents creates the specified number of events with unique talent IDs.
func generateEvents(ctx context.Context, config *Config, stats *Stats) ([]Event, error) {
	log.Printf("🎲 Generating %d events with unique talent IDs...", config.NumEvents)

	events := make([]Event, config.NumEvents)
	// rand.Seed is deprecated in Go 1.20+, using default source

	// Pre-allocate talent IDs to ensure uniqueness
	talentIDs := make([]string, config.NumEvents)
	for i := 0; i < config.NumEvents; i++ {
		talentIDs[i] = uuid.New().String()
	}

	// Generate events concurrently
	type eventResult struct {
		index int
		event Event
		err   error
	}

	resultChan := make(chan eventResult, config.NumEvents)

	// Use worker pool for event generation
	workerCount := min(config.Workers, config.NumEvents)
	eventsPerWorker := config.NumEvents / workerCount

	for worker := 0; worker < workerCount; worker++ {
		start := worker * eventsPerWorker
		end := start + eventsPerWorker
		if worker == workerCount-1 {
			end = config.NumEvents // Last worker gets remaining events
		}

		go func(workerID, start, end int) {
			for i := start; i < end; i++ {
				select {
				case <-ctx.Done():
					resultChan <- eventResult{index: i, err: ctx.Err()}
					return
				default:
					event, err := generateSingleEvent(i, talentIDs[i])
					resultChan <- eventResult{index: i, event: event, err: err}
				}
			}
		}(worker, start, end)
	}

	// Collect results
	for i := 0; i < config.NumEvents; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-resultChan:
			if result.err != nil {
				return nil, fmt.Errorf("failed to generate event %d: %w", result.index, result.err)
			}
			events[result.index] = result.event
		}
	}

	stats.EventsGenerated = len(events)
	log.Printf("✅ Generated %d events successfully", len(events))

	return events, nil
}

// generateSingleEvent creates a single event with the given index and talent ID.
func generateSingleEvent(index int, talentID string) (Event, error) {
	// Generate varied metric distribution (similar to shell script)
	rawMetric := generateVariedMetric()

	// Use "dribble" skill (matching shell script)
	skill := "dribble"

	// Current timestamp in RFC3339 format
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Generate unique event ID
	randNum, _ := rand.Int(rand.Reader, big.NewInt(10000))
	eventID := fmt.Sprintf("event_%04d_%d_%d", index, time.Now().Unix(), randNum.Int64())

	return Event{
		EventID:   eventID,
		TalentID:  talentID,
		RawMetric: rawMetric,
		Skill:     skill,
		TS:        timestamp,
	}, nil
}

// generateVariedMetric creates a metric with varied distribution.
func generateVariedMetric() float64 {
	// Use the same distribution logic as the shell script
	randNum, _ := rand.Int(rand.Reader, big.NewInt(8))
	switch randNum.Int64() {
	case 0:
		// Average performers (3.0 - 7.0) - most common
		return 3.0 + getRandomFloat()*4.0
	case 1:
		// High performers (7.0 - 9.0)
		return 7.0 + getRandomFloat()*2.0
	case 2:
		// Low performers (0.1 - 3.0)
		return 0.1 + getRandomFloat()*2.9
	case 3:
		// Elite performers (9.0 - 10.0) - rare
		return 9.0 + getRandomFloat()*1.0
	case 4:
		// Very low performers (0.1 - 1.0) - rare
		return 0.1 + getRandomFloat()*0.9
	case 5:
		// Mid-high performers (6.0 - 8.0)
		return 6.0 + getRandomFloat()*2.0
	case 6:
		// Mid-low performers (2.0 - 4.0)
		return 2.0 + getRandomFloat()*2.0
	case 7:
		// Random across full range (0.1 - 10.0)
		return 0.1 + getRandomFloat()*9.9
	default:
		return 0.1 + getRandomFloat()*9.9
	}
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
