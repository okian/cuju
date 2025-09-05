package testevents

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/okian/cuju/pkg/logger"
)

// Constants for random number generation.
const (
	randomFloatDivisor = 1000000
	eventIDDivisor     = 10000
	skillTypeDivisor   = 8
)

// Constants for metric generation ranges.
const (
	avgPerformerMin     = 3.0
	avgPerformerRange   = 4.0
	highPerformerMin    = 7.0
	highPerformerRange  = 2.0
	lowPerformerMin     = 0.1
	lowPerformerRange   = 2.9
	elitePerformerMin   = 9.0
	elitePerformerMax   = 10.0
	elitePerformerRange = 1.0
	veryLowMin          = 0.1
	veryLowRange        = 0.9
	midPerformerMin     = 6.0
	midPerformerRange   = 2.0
	goodPerformerMin    = 2.0
	goodPerformerRange  = 2.0
	wideRangeMin        = 0.1
	wideRangeMax        = 10.0
	wideRange           = 9.9
)

// Constants for performance type cases.
const (
	caseAveragePerformer = 0
	caseHighPerformer    = 1
	caseLowPerformer     = 2
	caseElitePerformer   = 3
	caseVeryLowPerformer = 4
	caseMidHighPerformer = 5
	caseMidLowPerformer  = 6
	caseWideRange        = 7
)

// getRandomFloat returns a random float64 between 0.0 and 1.0 using crypto/rand.
func getRandomFloat() float64 {
	n, _ := rand.Int(rand.Reader, big.NewInt(randomFloatDivisor))
	return float64(n.Int64()) / float64(randomFloatDivisor)
}

// generateEvents creates the specified number of events with unique talent IDs.
func generateEvents(ctx context.Context, config *Config, stats *Stats) ([]Event, error) {
	logger.Get().Info(ctx, "generating events with unique talent IDs", logger.Int("numEvents", config.NumEvents))

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
	workerCount := minInt(config.Workers, config.NumEvents)
	eventsPerWorker := config.NumEvents / workerCount

	for worker := 0; worker < workerCount; worker++ {
		start := worker * eventsPerWorker
		end := start + eventsPerWorker
		if worker == workerCount-1 {
			end = config.NumEvents // Last worker gets remaining events
		}

		go func(_ int, start, end int) {
			for i := start; i < end; i++ {
				select {
				case <-ctx.Done():
					resultChan <- eventResult{index: i, err: ctx.Err()}
					return
				default:
					event := generateSingleEvent(i, talentIDs[i])
					resultChan <- eventResult{index: i, event: event, err: nil}
				}
			}
		}(worker, start, end)
	}

	// Collect results
	for i := 0; i < config.NumEvents; i++ {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled during event generation: %w", ctx.Err())
		case result := <-resultChan:
			if result.err != nil {
				return nil, fmt.Errorf("failed to generate event %d: %w", result.index, result.err)
			}
			events[result.index] = result.event
		}
	}

	stats.EventsGenerated = len(events)
	logger.Get().Info(ctx, "generated events successfully", logger.Int("count", len(events)))

	return events, nil
}

// generateSingleEvent creates a single event with the given index and talent ID.
func generateSingleEvent(index int, talentID string) Event {
	// Generate varied metric distribution (similar to shell script)
	rawMetric := generateVariedMetric()

	// Use "dribble" skill (matching shell script)
	skill := "dribble"

	// Current timestamp in RFC3339 format
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Generate unique event ID
	randNum, _ := rand.Int(rand.Reader, big.NewInt(eventIDDivisor))
	eventID := "event_" + strconv.FormatInt(int64(index), 10) + "_" + strconv.FormatInt(time.Now().Unix(), 10) + "_" + strconv.FormatInt(randNum.Int64(), 10)

	return Event{
		EventID:   eventID,
		TalentID:  talentID,
		RawMetric: rawMetric,
		Skill:     skill,
		TS:        timestamp,
	}
}

// generateVariedMetric creates a metric with varied distribution.
func generateVariedMetric() float64 {
	// Use the same distribution logic as the shell script
	randNum, _ := rand.Int(rand.Reader, big.NewInt(skillTypeDivisor))
	switch randNum.Int64() {
	case caseAveragePerformer:
		// Average performers (3.0 - 7.0) - most common
		return avgPerformerMin + getRandomFloat()*avgPerformerRange
	case caseHighPerformer:
		// High performers (7.0 - 9.0)
		return highPerformerMin + getRandomFloat()*highPerformerRange
	case caseLowPerformer:
		// Low performers (0.1 - 3.0)
		return lowPerformerMin + getRandomFloat()*lowPerformerRange
	case caseElitePerformer:
		// Elite performers (9.0 - 10.0) - rare
		return elitePerformerMin + getRandomFloat()*elitePerformerRange
	case caseVeryLowPerformer:
		// Very low performers (0.1 - 1.0) - rare
		return veryLowMin + getRandomFloat()*veryLowRange
	case caseMidHighPerformer:
		// Mid-high performers (6.0 - 8.0)
		return midPerformerMin + getRandomFloat()*midPerformerRange
	case caseMidLowPerformer:
		// Mid-low performers (2.0 - 4.0)
		return goodPerformerMin + getRandomFloat()*goodPerformerRange
	case caseWideRange:
		// Random across full range (0.1 - 10.0)
		return wideRangeMin + getRandomFloat()*wideRange
	default:
		return wideRangeMin + getRandomFloat()*wideRange
	}
}

// minInt returns the minimum of two integers.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
