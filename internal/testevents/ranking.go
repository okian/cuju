package testevents

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/okian/cuju/pkg/logger"
)

// retrieveRankings retrieves rankings for all talents concurrently.
func retrieveRankings(ctx context.Context, config *Config, events []Event, stats *Stats) ([]Entry, error) { //nolint:gocognit,unparam // complex function required for concurrent ranking retrieval, error return for consistency
	logger.Get().Info(ctx, "retrieving rankings for talents", logger.Int("talents", len(events)), logger.Int("workers", config.Workers))

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
		go func(_ int) {
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
							logger.Get().Warn(ctx, "failed to get rank for talent", logger.String("talentID", talentID), logger.Error(err))
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
							logger.Get().Info(ctx, "ranking progress",
								logger.Int("total", int(total)),
								logger.Int("talents", len(talentIDs)),
								logger.Int("success", int(ret)),
								logger.Int("failed", int(fail)))
						} else {
							logger.Get().Info(ctx, "rankings progress",
								logger.Int("total", int(total)),
								logger.Int("talents", len(talentIDs)),
								logger.Int("success", int(ret)),
								logger.Int("failed", int(fail)))
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
		logger.Get().Info(ctx, "progress indicator complete")
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

	logger.Get().Info(ctx, "ranking retrieval completed",
		logger.Int("retrieved", len(validRankings)),
		logger.Int("failed", int(atomic.LoadInt64(&failed))))

	return validRankings, nil
}

// retrieveSingleRanking retrieves ranking for a single talent.
func retrieveSingleRanking(ctx context.Context, client *HTTPClient, baseURL, talentID string) (Entry, error) {
	url := baseURL + "/rank/" + talentID

	resp, err := client.Get(ctx, url)
	if err != nil {
		return Entry{}, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Get().Error(context.Background(), "failed to close response body", logger.Error(err))
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
	logger.Get().Info(ctx, "getting top leaderboard entries", logger.Int("topN", config.TopN))

	client := newHTTPClient(config.Timeout)
	url := config.BaseURL + "/leaderboard?limit=" + strconv.Itoa(config.TopN)

	resp, err := client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Get().Error(context.Background(), "failed to close response body", logger.Error(err))
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
	logger.Get().Info(ctx, "retrieved leaderboard entries", logger.Int("count", len(leaderboard)))

	return leaderboard, nil
}
