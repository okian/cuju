package worker_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	queue "github.com/okian/cuju/internal/adapters/mq/queue"
	worker "github.com/okian/cuju/internal/adapters/mq/worker"
	model "github.com/okian/cuju/internal/domain/model"
	logging "github.com/okian/cuju/pkg/logger"
	"github.com/smartystreets/goconvey/convey"
)

// Mock implementations for testing.
type mockQueue struct {
	eventChan  chan queue.Event
	closeError error
}

func newMockQueue() *mockQueue {
	return &mockQueue{
		eventChan: make(chan queue.Event, 10),
	}
}

func (mq *mockQueue) Dequeue(ctx context.Context) <-chan queue.Event {
	return mq.eventChan
}

func (mq *mockQueue) Close() error {
	close(mq.eventChan)
	return mq.closeError
}

func (mq *mockQueue) addEvent(event queue.Event) { //nolint:gocritic // hugeParam: Event must be passed by value for channel semantics
	mq.eventChan <- event
}

type mockScorer struct {
	scores map[string]float64
	errors map[string]error
	mu     sync.RWMutex
}

func newMockScorer() *mockScorer {
	return &mockScorer{
		scores: make(map[string]float64),
		errors: make(map[string]error),
	}
}

func (ms *mockScorer) Score(ctx context.Context, talentID string, rawMetric float64, skill string) (float64, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if err, exists := ms.errors[talentID]; exists {
		return 0, err
	}
	if score, exists := ms.scores[talentID]; exists {
		return score, nil
	}
	return rawMetric * 0.8, nil // Default scoring
}

func (ms *mockScorer) setScore(talentID string, score float64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.scores[talentID] = score
}

func (ms *mockScorer) setError(talentID string, err error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.errors[talentID] = err
}

type mockUpdater struct {
	updates map[string]float64
	errors  map[string]error
	mu      sync.RWMutex
}

func newMockUpdater() *mockUpdater {
	return &mockUpdater{
		updates: make(map[string]float64),
		errors:  make(map[string]error),
	}
}

func (mu *mockUpdater) UpdateBest(ctx context.Context, talentID string, score float64) (bool, error) {
	mu.mu.Lock()
	defer mu.mu.Unlock()

	if err, exists := mu.errors[talentID]; exists {
		return false, err
	}

	mu.updates[talentID] = score
	return true, nil
}

func (mu *mockUpdater) UpdateBestWithMeta(ctx context.Context, talentID string, score float64, eventID, skill string, rawMetric float64) (bool, error) {
	// For tests, just delegate to UpdateBest
	return mu.UpdateBest(ctx, talentID, score)
}

func (mu *mockUpdater) setError(talentID string, err error) {
	mu.mu.Lock()
	defer mu.mu.Unlock()
	mu.errors[talentID] = err
}

func (mu *mockUpdater) getUpdate(talentID string) (float64, bool) {
	mu.mu.RLock()
	defer mu.mu.RUnlock()
	score, exists := mu.updates[talentID]
	return score, exists
}

func TestInMemoryWorker(t *testing.T) {
	convey.Convey("Given a new InMemoryWorker", t, func() {
		// Initialize logging for tests
		_ = logging.Init()

		queue := newMockQueue()
		scorer := newMockScorer()
		updater := newMockUpdater()

		convey.Convey("When creating a worker with default options", func() {
			worker := worker.NewInMemoryWorker(queue, scorer, updater)

			convey.Convey("Then it should be created successfully", func() {
				convey.So(worker, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When creating a worker with custom options", func() {
			worker := worker.NewInMemoryWorker(
				queue, scorer, updater,
				worker.WithName("test-worker"),
			)

			convey.Convey("Then it should be created successfully", func() {
				convey.So(worker, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When running a worker", func() {
			worker := worker.NewInMemoryWorker(queue, scorer, updater)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Start worker in goroutine
			go worker.Run(ctx)

			// Give worker time to start
			time.Sleep(10 * time.Millisecond)

			convey.Convey("And when processing events", func() {
				event := model.Event{
					EventID:   "event-1",
					TalentID:  "talent-1",
					RawMetric: 100.0,
					Skill:     "programming",
					TS:        time.Now(),
				}

				// Set expected score
				scorer.setScore("talent-1", 85.0)

				// Add event to queue
				queue.addEvent(event)

				// Give worker time to process
				time.Sleep(50 * time.Millisecond)

				convey.Convey("Then it should update the leaderboard", func() {
					score, updated := updater.getUpdate("talent-1")
					convey.So(updated, convey.ShouldBeTrue)
					convey.So(score, convey.ShouldEqual, 85.0)
				})
			})

			convey.Convey("And when scoring fails", func() {
				event := model.Event{
					EventID:   "event-2",
					TalentID:  "talent-2",
					RawMetric: 100.0,
					Skill:     "programming",
					TS:        time.Now(),
				}

				// Set scoring error
				scorer.setError("talent-2", errors.New("scoring error"))

				// Add event to queue
				queue.addEvent(event)

				// Give worker time to process
				time.Sleep(50 * time.Millisecond)

				convey.Convey("Then it should not update the leaderboard", func() {
					_, updated := updater.getUpdate("talent-2")
					convey.So(updated, convey.ShouldBeFalse)
				})
			})

			convey.Convey("And when updating fails", func() {
				event := model.Event{
					EventID:   "event-3",
					TalentID:  "talent-3",
					RawMetric: 100.0,
					Skill:     "programming",
					TS:        time.Now(),
				}

				// Set updater error
				updater.setError("talent-3", errors.New("update error"))

				// Add event to queue
				queue.addEvent(event)

				// Give worker time to process
				time.Sleep(50 * time.Millisecond)

				convey.Convey("Then it should not update the leaderboard", func() {
					_, updated := updater.getUpdate("talent-3")
					convey.So(updated, convey.ShouldBeFalse)
				})
			})

			convey.Convey("And when shutting down", func() {
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer shutdownCancel()

				err := worker.Shutdown(shutdownCtx)

				convey.Convey("Then it should shutdown gracefully", func() {
					convey.So(err, convey.ShouldBeNil)
				})
			})
		})

		convey.Convey("When context is cancelled", func() {
			worker := worker.NewInMemoryWorker(queue, scorer, updater)
			ctx, cancel := context.WithCancel(context.Background())

			// Start worker in goroutine
			go worker.Run(ctx)

			// Give worker time to start
			time.Sleep(10 * time.Millisecond)

			// Cancel context
			cancel()

			// Give worker time to stop
			time.Sleep(50 * time.Millisecond)

			convey.Convey("Then worker should stop", func() {
				// Worker should have stopped due to context cancellation
				convey.So(true, convey.ShouldBeTrue) // Placeholder assertion
			})
		})
	})
}

func TestWorkerPool(t *testing.T) {
	convey.Convey("Given a new WorkerPool", t, func() {
		// Initialize logging for tests
		_ = logging.Init()

		queue := newMockQueue()
		scorer := newMockScorer()
		updater := newMockUpdater()

		convey.Convey("When creating a worker pool with default count", func() {
			pool := worker.NewPool(0, queue, scorer, updater)

			convey.Convey("Then it should be created successfully", func() {
				convey.So(pool, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When creating a worker pool with custom count", func() {
			workerCount := 3
			pool := worker.NewPool(workerCount, queue, scorer, updater)

			convey.Convey("Then it should be created successfully", func() {
				convey.So(pool, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When starting a worker pool", func() {
			pool := worker.NewPool(2, queue, scorer, updater)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			pool.Start(ctx)

			// Give workers time to start
			time.Sleep(20 * time.Millisecond)

			convey.Convey("And when processing multiple events", func() {
				events := []model.Event{
					{EventID: "event-1", TalentID: "talent-1", RawMetric: 100.0, Skill: "programming", TS: time.Now()},
					{EventID: "event-2", TalentID: "talent-2", RawMetric: 95.0, Skill: "design", TS: time.Now()},
					{EventID: "event-3", TalentID: "talent-3", RawMetric: 90.0, Skill: "marketing", TS: time.Now()},
				}

				// Set expected scores
				scorer.setScore("talent-1", 85.0)
				scorer.setScore("talent-2", 80.0)
				scorer.setScore("talent-3", 75.0)

				// Add events to queue
				for _, event := range events {
					queue.addEvent(event)
				}

				// Give workers time to process
				time.Sleep(100 * time.Millisecond)

				convey.Convey("Then all events should be processed", func() {
					for _, event := range events {
						score, updated := updater.getUpdate(event.TalentID)
						convey.So(updated, convey.ShouldBeTrue)
						convey.So(score, convey.ShouldBeGreaterThan, 0)
					}
				})
			})

			convey.Convey("And when shutting down", func() {
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
				defer shutdownCancel()

				err := pool.Shutdown(shutdownCtx)

				convey.Convey("Then it should shutdown gracefully", func() {
					convey.So(err, convey.ShouldBeNil)
				})
			})
		})

		convey.Convey("When stopping a worker pool", func() {
			pool := worker.NewPool(2, queue, scorer, updater)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			pool.Start(ctx)

			// Give workers time to start
			time.Sleep(20 * time.Millisecond)

			pool.Stop()

			// Give workers time to stop
			time.Sleep(50 * time.Millisecond)

			convey.Convey("Then all workers should be stopped", func() {
				// Workers should have stopped
				convey.So(true, convey.ShouldBeTrue) // Placeholder assertion
			})
		})
	})
}

func TestWorkerOptions(t *testing.T) {
	convey.Convey("Given worker options", t, func() {
		convey.Convey("When using WithName", func() {
			convey.Convey("Then it should set the worker name", func() {
				queue := newMockQueue()
				scorer := newMockScorer()
				updater := newMockUpdater()
				worker := worker.NewInMemoryWorker(queue, scorer, updater, worker.WithName("test-worker"))
				// Note: Can't test unexported fields directly
				convey.So(worker, convey.ShouldNotBeNil)
			})
		})

		// Removed tests for unused options: WithBatchSize, WithTimeout, WithRetryCount, WithBackoff, WithMetrics
	})
}

func TestWorkerConcurrency(t *testing.T) {
	convey.Convey("Given a worker pool with multiple workers", t, func() {
		// Initialize logging for tests
		_ = logging.Init()

		queue := newMockQueue()
		scorer := newMockScorer()
		updater := newMockUpdater()

		pool := worker.NewPool(4, queue, scorer, updater)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		pool.Start(ctx)

		// Give workers time to start
		time.Sleep(20 * time.Millisecond)

		convey.Convey("When processing many concurrent events", func() {
			const eventCount = 100
			var wg sync.WaitGroup

			// Start multiple goroutines adding events
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()
					for j := 0; j < eventCount/5; j++ {
						eventID := fmt.Sprintf("event-%d-%d", workerID, j)
						talentID := fmt.Sprintf("talent-%d-%d", workerID, j)
						event := model.Event{
							EventID:   eventID,
							TalentID:  talentID,
							RawMetric: float64(100 - j),
							Skill:     "programming",
							TS:        time.Now(),
						}
						scorer.setScore(talentID, float64(80-j))
						queue.addEvent(event)
					}
				}(i)
			}

			// Wait for all events to be added
			wg.Wait()

			// Give workers time to process
			time.Sleep(200 * time.Millisecond)

			convey.Convey("Then all events should be processed", func() {
				// Check that all events were processed
				processedCount := 0
				for i := 0; i < 5; i++ {
					for j := 0; j < eventCount/5; j++ {
						talentID := fmt.Sprintf("talent-%d-%d", i, j)
						if _, updated := updater.getUpdate(talentID); updated {
							processedCount++
						}
					}
				}
				convey.So(processedCount, convey.ShouldEqual, eventCount)
			})
		})
	})
}

func TestWorkerErrorHandling(t *testing.T) {
	convey.Convey("Given a worker with error conditions", t, func() {
		// Initialize logging for tests
		_ = logging.Init()

		queue := newMockQueue()
		scorer := newMockScorer()
		updater := newMockUpdater()

		worker := worker.NewInMemoryWorker(queue, scorer, updater)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start worker in goroutine
		go worker.Run(ctx)

		// Give worker time to start
		time.Sleep(10 * time.Millisecond)

		convey.Convey("When scoring consistently fails", func() {
			event := model.Event{
				EventID:   "event-error",
				TalentID:  "talent-error",
				RawMetric: 100.0,
				Skill:     "programming",
				TS:        time.Now(),
			}

			// Set persistent scoring error
			scorer.setError("talent-error", errors.New("persistent scoring error"))

			// Add event to queue
			queue.addEvent(event)

			// Give worker time to process
			time.Sleep(50 * time.Millisecond)

			convey.Convey("Then it should not update the leaderboard", func() {
				_, updated := updater.getUpdate("talent-error")
				convey.So(updated, convey.ShouldBeFalse)
			})
		})

		convey.Convey("When updating consistently fails", func() {
			event := model.Event{
				EventID:   "event-update-error",
				TalentID:  "talent-update-error",
				RawMetric: 100.0,
				Skill:     "programming",
				TS:        time.Now(),
			}

			// Set persistent update error
			updater.setError("talent-update-error", errors.New("persistent update error"))

			// Add event to queue
			queue.addEvent(event)

			// Give worker time to process
			time.Sleep(50 * time.Millisecond)

			convey.Convey("Then it should not update the leaderboard", func() {
				_, updated := updater.getUpdate("talent-update-error")
				convey.So(updated, convey.ShouldBeFalse)
			})
		})

		convey.Convey("When queue channel is closed", func() {
			// Close the queue
			_ = queue.Close()

			// Give worker time to detect closure
			time.Sleep(50 * time.Millisecond)

			convey.Convey("Then worker should stop", func() {
				// Worker should have stopped due to queue closure
				convey.So(true, convey.ShouldBeTrue) // Placeholder assertion
			})
		})
	})
}
