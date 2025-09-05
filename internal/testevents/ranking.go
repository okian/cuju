package testevents

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// retrieveRankings retrieves rankings for all talents concurrently.
func retrieveRankings(ctx context.Context, config *Config, events []Event, stats *Stats) ([]Entry, error) {
	log.Printf("🏆 Retrieving rankings for %d talents with %d workers...", len(events), config.Workers)

	client := newHTTPClient(config.Timeout)

	// Extract unique talent IDs
	talentIDs := make([]string, len(events))
	for i, event := range events {
		talentIDs[i] = event.TalentID
	}

	// Results storage
	rankings := make([]Entry, len(talentIDs))
	var (
		retrieved int64
		failed    int64
	)

	// Progress reporting
	var lastReport time.Time
	reportInterval := 1 * time.Second

	// Create worker pool
	talentChan := make(chan int, config.Workers*WorkerChannelMultiplier) // Send indices instead of IDs
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for index := range talentChan {
				select {
				case <-ctx.Done():
					return
				default:
					talentID := talentIDs[index]
					entry, err := retrieveSingleRanking(ctx, client, config.BaseURL, talentID)

					if err != nil {
						atomic.AddInt64(&failed, 1)
						if config.Verbose {
							log.Printf("⚠️  Failed to get rank for %s: %v", talentID, err)
						}
					} else {
						rankings[index] = entry
						atomic.AddInt64(&retrieved, 1)
					}

					// Progress reporting
					if time.Since(lastReport) >= reportInterval {
						lastReport = time.Now()
						total := atomic.LoadInt64(&retrieved) + atomic.LoadInt64(&failed)
						ret := atomic.LoadInt64(&retrieved)
						fail := atomic.LoadInt64(&failed)

						if config.Verbose {
							log.Printf("📊 Ranking progress: %d/%d retrieved (success: %d, failed: %d)",
								total, len(talentIDs), ret, fail)
						} else {
							log.Printf("\r🏆 Rankings: %d/%d retrieved (success: %d, failed: %d)",
								total, len(talentIDs), ret, fail)
						}
					}
				}
			}
		}(i)
	}

	// Send talent indices to workers
	go func() {
		defer close(talentChan)
		for i := range talentIDs {
			select {
			case <-ctx.Done():
				return
			case talentChan <- i:
			}
		}
	}()

	// Wait for all workers to complete
	wg.Wait()

	// Final progress report
	if !config.Verbose {
		log.Println() // New line after progress indicator
	}

	// Filter out empty entries (failed retrievals)
	validRankings := make([]Entry, 0, len(rankings))
	for _, entry := range rankings {
		if entry.TalentID != "" { // Empty TalentID indicates failed retrieval
			validRankings = append(validRankings, entry)
		}
	}

	// Update stats
	stats.RankingsRetrieved = len(validRankings)

	log.Printf(`✅ Ranking retrieval completed:
   Retrieved: %d
   Failed: %d
`, len(validRankings), int(atomic.LoadInt64(&failed)))

	return validRankings, nil
}

// retrieveSingleRanking retrieves ranking for a single talent.
func retrieveSingleRanking(ctx context.Context, client *HTTPClient, baseURL, talentID string) (Entry, error) {
	url := fmt.Sprintf("%s/rank/%s", baseURL, talentID)

	resp, err := client.Get(url)
	if err != nil {
		return Entry{}, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != StatusOK {
		body, _ := readResponseBody(resp)
		return Entry{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := readResponseBody(resp)
	if err != nil {
		return Entry{}, fmt.Errorf("failed to read response: %w", err)
	}

	var entry Entry
	if err := unmarshalJSON(body, &entry); err != nil {
		return Entry{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return entry, nil
}

// getLeaderboard retrieves the top N leaderboard entries.
func getLeaderboard(ctx context.Context, config *Config, stats *Stats) ([]Entry, error) {
	log.Printf("🥇 Getting top %d leaderboard entries...", config.TopN)

	client := newHTTPClient(config.Timeout)
	url := fmt.Sprintf("%s/leaderboard?limit=%d", config.BaseURL, config.TopN)

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != StatusOK {
		body, _ := readResponseBody(resp)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := readResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var leaderboard []Entry
	if err := unmarshalJSON(body, &leaderboard); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	stats.LeaderboardEntries = len(leaderboard)
	log.Printf("✅ Retrieved %d leaderboard entries", len(leaderboard))

	return leaderboard, nil
}
