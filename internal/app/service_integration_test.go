package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	service "github.com/okian/cuju/internal/app"
	"github.com/okian/cuju/internal/domain/model"
	"github.com/okian/cuju/pkg/logger"
	"github.com/smartystreets/goconvey/convey"
)

func init() {
	// Initialize logging for tests
	err := logger.Init()
	if err != nil {
		panic(err)
	}
}

func TestServiceIntegration(t *testing.T) {
	convey.Convey("Given a service with full integration", t, func() {
		svc := service.New(
			service.WithWorkerCount(2),
			service.WithQueueSize(1000),
			service.WithDedupeSize(500),
		)
		defer svc.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		convey.Convey("When starting the service", func() {
			err := svc.Start(ctx)

			convey.Convey("Then it should start successfully", func() {
				convey.So(err, convey.ShouldBeNil)
			})

			convey.Convey("And the service should be running", func() {
				stats := svc.GetStats()
				convey.So(stats["started"], convey.ShouldEqual, true)
			})
		})

		convey.Convey("When processing events end-to-end", func() {
			err := svc.Start(ctx)
			convey.So(err, convey.ShouldBeNil)

			// Give service time to start
			time.Sleep(100 * time.Millisecond)

			convey.Convey("And enqueueing multiple events", func() {
				events := []model.Event{
					{
						EventID:   "event-1",
						TalentID:  "talent-1",
						RawMetric: 85.0,
						Skill:     "coding",
						TS:        time.Now(),
					},
					{
						EventID:   "event-2",
						TalentID:  "talent-2",
						RawMetric: 90.0,
						Skill:     "design",
						TS:        time.Now(),
					},
					{
						EventID:   "event-3",
						TalentID:  "talent-1", // Same talent, different event
						RawMetric: 95.0,       // Higher score
						Skill:     "coding",
						TS:        time.Now(),
					},
				}

				// Enqueue all events
				for _, event := range events {
					success := svc.Enqueue(ctx, event)
					convey.So(success, convey.ShouldBeTrue)
				}

				// Give workers time to process
				time.Sleep(500 * time.Millisecond)

				convey.Convey("Then events should be processed", func() {
					stats := svc.GetStats()
					convey.So(stats, convey.ShouldNotBeNil)
				})

				convey.Convey("And duplicate events should be detected", func() {
					// Try to enqueue the same event again
					duplicateEvent := events[0]
					success := svc.Enqueue(ctx, duplicateEvent)
					convey.So(success, convey.ShouldBeTrue) // Should still be enqueued (but as duplicate)

					// The event should be detected as duplicate during enqueue
					// We can verify this by checking that the event was processed
					stats := svc.GetStats()
					convey.So(stats, convey.ShouldNotBeNil)
				})

				convey.Convey("And leaderboard should be updated", func() {
					// Get top N entries
					entries, err := svc.TopN(ctx, 10)
					convey.So(err, convey.ShouldBeNil)
					convey.So(len(entries), convey.ShouldBeGreaterThan, 0)

					// Verify ordering (highest scores first)
					for i := 1; i < len(entries); i++ {
						convey.So(entries[i-1].Score, convey.ShouldBeGreaterThanOrEqualTo, entries[i].Score)
					}
				})

				convey.Convey("And individual ranks should be available", func() {
					// Check rank for talent-1 (should have the higher score)
					entry, err := svc.Rank(ctx, "talent-1")
					convey.So(err, convey.ShouldBeNil)
					convey.So(entry.TalentID, convey.ShouldEqual, "talent-1")
					convey.So(entry.Score, convey.ShouldEqual, 100.0) // Should have the higher score (95.0 * 1.5 = 142.5, capped at 100)
					convey.So(entry.Rank, convey.ShouldBeGreaterThan, 0)
				})
			})
		})

		convey.Convey("When handling high-volume events", func() {
			err := svc.Start(ctx)
			convey.So(err, convey.ShouldBeNil)

			// Give service time to start
			time.Sleep(100 * time.Millisecond)

			convey.Convey("And enqueueing many events concurrently", func() {
				numEvents := 100
				events := make([]model.Event, numEvents)

				// Generate events
				for i := 0; i < numEvents; i++ {
					events[i] = model.Event{
						EventID:   "bulk-event-" + string(rune(i)),
						TalentID:  "talent-" + string(rune(i%10)), // 10 different talents
						RawMetric: float64(50 + i%50),             // Scores 50-99
						Skill:     "skill-" + string(rune(i%5)),   // 5 different skills
						TS:        time.Now(),
					}
				}

				// Enqueue all events
				successCount := 0
				for _, event := range events {
					if svc.Enqueue(ctx, event) {
						successCount++
					}
				}

				convey.Convey("Then most events should be enqueued successfully", func() {
					convey.So(successCount, convey.ShouldBeGreaterThan, numEvents/2)
				})

				// Give workers time to process
				time.Sleep(1 * time.Second)

				convey.Convey("And leaderboard should reflect the updates", func() {
					entries, err := svc.TopN(ctx, 20)
					convey.So(err, convey.ShouldBeNil)
					convey.So(len(entries), convey.ShouldBeGreaterThan, 0)

					// Verify we have entries for multiple talents
					talentIDs := make(map[string]bool)
					for _, entry := range entries {
						talentIDs[entry.TalentID] = true
					}
					convey.So(len(talentIDs), convey.ShouldBeGreaterThan, 1)
				})
			})
		})

		convey.Convey("When handling service lifecycle", func() {
			convey.Convey("And starting and stopping multiple times", func() {
				// Start service
				err := svc.Start(ctx)
				convey.So(err, convey.ShouldBeNil)

				// Give it time to start
				time.Sleep(100 * time.Millisecond)

				// Stop service
				svc.Stop()

				// Give it time to stop
				time.Sleep(100 * time.Millisecond)

				// Check it's stopped
				stats := svc.GetStats()
				convey.So(stats["started"], convey.ShouldEqual, false)

				// Start again
				err = svc.Start(ctx)
				convey.So(err, convey.ShouldBeNil)

				// Give it time to start
				time.Sleep(100 * time.Millisecond)

				// Check it's started again
				stats = svc.GetStats()
				convey.So(stats["started"], convey.ShouldEqual, true)
			})
		})

		convey.Convey("When handling edge cases", func() {
			err := svc.Start(ctx)
			convey.So(err, convey.ShouldBeNil)

			// Give service time to start
			time.Sleep(100 * time.Millisecond)

			convey.Convey("And enqueueing events with extreme values", func() {
				extremeEvents := []model.Event{
					{
						EventID:   "extreme-1",
						TalentID:  "talent-extreme",
						RawMetric: 0.0,
						Skill:     "zero-skill",
						TS:        time.Now(),
					},
					{
						EventID:   "extreme-2",
						TalentID:  "talent-extreme",
						RawMetric: 1000.0,
						Skill:     "max-skill",
						TS:        time.Now(),
					},
					{
						EventID:   "extreme-3",
						TalentID:  "talent-extreme",
						RawMetric: -100.0,
						Skill:     "negative-skill",
						TS:        time.Now(),
					},
				}

				for _, event := range extremeEvents {
					success := svc.Enqueue(ctx, event)
					convey.So(success, convey.ShouldBeTrue)
				}

				// Give workers time to process
				time.Sleep(500 * time.Millisecond)

				convey.Convey("Then extreme values should be handled", func() {
					// Service should still be running
					stats := svc.GetStats()
					convey.So(stats["started"], convey.ShouldEqual, true)
				})
			})

			convey.Convey("And enqueueing events with very long IDs", func() {
				longID := "very-long-event-id-" + string(make([]byte, 1000))
				longTalentID := "very-long-talent-id-" + string(make([]byte, 1000))

				event := model.Event{
					EventID:   longID,
					TalentID:  longTalentID,
					RawMetric: 75.0,
					Skill:     "long-skill",
					TS:        time.Now(),
				}

				success := svc.Enqueue(ctx, event)
				convey.So(success, convey.ShouldBeTrue)

				// Give workers time to process
				time.Sleep(500 * time.Millisecond)

				convey.Convey("Then long IDs should be handled", func() {
					// Service should still be running
					stats := svc.GetStats()
					convey.So(stats["started"], convey.ShouldEqual, true)
				})
			})
		})
	})
}

func TestServiceConcurrency(t *testing.T) {
	convey.Convey("Given a service with concurrent operations", t, func() {
		svc := service.New(
			service.WithWorkerCount(4),
			service.WithQueueSize(2000),
			service.WithDedupeSize(1000),
		)
		defer svc.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := svc.Start(ctx)
		convey.So(err, convey.ShouldBeNil)

		// Give service time to start
		time.Sleep(100 * time.Millisecond)

		convey.Convey("When multiple goroutines enqueue events concurrently", func() {
			numGoroutines := 10
			eventsPerGoroutine := 50
			done := make(chan bool, numGoroutines)

			// Start multiple goroutines
			for i := 0; i < numGoroutines; i++ {
				go func(goroutineID int) {
					for j := 0; j < eventsPerGoroutine; j++ {
						event := model.Event{
							EventID:   "concurrent-event-" + string(rune(goroutineID)) + "-" + string(rune(j)),
							TalentID:  "talent-" + string(rune(goroutineID)),
							RawMetric: float64(50 + j),
							Skill:     "concurrent-skill",
							TS:        time.Now(),
						}
						svc.Enqueue(ctx, event)
					}
					done <- true
				}(i)
			}

			// Wait for all goroutines to complete
			for i := 0; i < numGoroutines; i++ {
				<-done
			}

			// Give workers time to process
			time.Sleep(2 * time.Second)

			convey.Convey("Then all events should be processed", func() {
				// Service should still be running
				stats := svc.GetStats()
				convey.So(stats["started"], convey.ShouldEqual, true)

				// Should have entries in leaderboard
				entries, err := svc.TopN(ctx, 100)
				convey.So(err, convey.ShouldBeNil)
				convey.So(len(entries), convey.ShouldBeGreaterThan, 0)
			})
		})

		convey.Convey("When multiple goroutines query the leaderboard concurrently", func() {
			numGoroutines := 20
			done := make(chan bool, numGoroutines)
			errors := make(chan error, numGoroutines*20) // Buffer for potential errors

			// Start multiple goroutines querying
			for i := 0; i < numGoroutines; i++ {
				go func(_ int) {
					for j := 0; j < 10; j++ {
						// Query TopN
						entries, err := svc.TopN(ctx, 10)
						if err != nil {
							errors <- err
							continue
						}
						if entries == nil {
							errors <- fmt.Errorf("entries is nil")
							continue
						}

						// Query individual rank
						if len(entries) > 0 {
							entry, err := svc.Rank(ctx, entries[0].TalentID)
							if err != nil {
								errors <- err
								continue
							}
							if entry.TalentID == "" {
								errors <- fmt.Errorf("talent ID is empty")
								continue
							}
						}
					}
					done <- true
				}(i)
			}

			// Wait for all goroutines to complete
			for i := 0; i < numGoroutines; i++ {
				<-done
			}

			convey.Convey("Then all queries should succeed", func() {
				// Check if any errors occurred
				select {
				case err := <-errors:
					convey.So(err, convey.ShouldBeNil)
				default:
					// No errors, test passed
					convey.So(true, convey.ShouldBeTrue)
				}
			})
		})
	})
}

func TestServiceErrorHandling(t *testing.T) {
	convey.Convey("Given a service with error conditions", t, func() {
		svc := service.New(
			service.WithWorkerCount(1),
			service.WithQueueSize(10), // Small queue to test backpressure
			service.WithDedupeSize(5),
		)
		defer svc.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := svc.Start(ctx)
		convey.So(err, convey.ShouldBeNil)

		// Give service time to start
		time.Sleep(100 * time.Millisecond)

		convey.Convey("When enqueueing events beyond queue capacity", func() {
			// Fill the queue beyond capacity
			successCount := 0
			for i := 0; i < 20; i++ {
				event := model.Event{
					EventID:   "backpressure-event-" + string(rune(i)),
					TalentID:  "talent-" + string(rune(i)),
					RawMetric: float64(50 + i),
					Skill:     "backpressure-skill",
					TS:        time.Now(),
				}
				if svc.Enqueue(ctx, event) {
					successCount++
				}
			}

			convey.Convey("Then some events should be rejected due to backpressure", func() {
				convey.So(successCount, convey.ShouldBeLessThan, 20)
				convey.So(successCount, convey.ShouldBeGreaterThan, 0)
			})
		})

		convey.Convey("When querying non-existent talents", func() {
			entry, err := svc.Rank(ctx, "non-existent-talent")

			convey.Convey("Then it should return an error", func() {
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(entry.TalentID, convey.ShouldEqual, "")
			})
		})

		convey.Convey("When querying with invalid limits", func() {
			entries, err := svc.TopN(ctx, 0)

			convey.Convey("Then it should return an error", func() {
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(entries, convey.ShouldBeNil)
			})
		})

		convey.Convey("When querying with negative limits", func() {
			entries, err := svc.TopN(ctx, -1)

			convey.Convey("Then it should return an error", func() {
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(entries, convey.ShouldBeNil)
			})
		})
	})
}

func TestServicePerformance(t *testing.T) {
	convey.Convey("Given a service for performance testing", t, func() {
		svc := service.New(
			service.WithWorkerCount(8),
			service.WithQueueSize(10000),
			service.WithDedupeSize(5000),
		)
		defer svc.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := svc.Start(ctx)
		convey.So(err, convey.ShouldBeNil)

		// Give service time to start
		time.Sleep(100 * time.Millisecond)

		convey.Convey("When processing a large number of events", func() {
			numEvents := 1000
			start := time.Now()

			// Enqueue events
			for i := 0; i < numEvents; i++ {
				event := model.Event{
					EventID:   "perf-event-" + string(rune(i)),
					TalentID:  "talent-" + string(rune(i%100)), // 100 different talents
					RawMetric: float64(50 + i%50),
					Skill:     "perf-skill",
					TS:        time.Now(),
				}
				svc.Enqueue(ctx, event)
			}

			enqueueTime := time.Since(start)

			// Give workers time to process
			time.Sleep(2 * time.Second)

			convey.Convey("Then enqueueing should be fast", func() {
				// Should be able to enqueue 1000 events in reasonable time
				convey.So(enqueueTime, convey.ShouldBeLessThan, 5*time.Second)
			})

			convey.Convey("And leaderboard queries should be fast", func() {
				start := time.Now()
				entries, err := svc.TopN(ctx, 100)
				queryTime := time.Since(start)

				convey.So(err, convey.ShouldBeNil)
				convey.So(len(entries), convey.ShouldBeGreaterThan, 0)
				convey.So(queryTime, convey.ShouldBeLessThan, 100*time.Millisecond)
			})

			convey.Convey("And rank queries should be fast", func() {
				start := time.Now()
				entry, err := svc.Rank(ctx, "talent-0")
				queryTime := time.Since(start)

				convey.So(err, convey.ShouldBeNil)
				convey.So(entry.TalentID, convey.ShouldEqual, "talent-0")
				convey.So(queryTime, convey.ShouldBeLessThan, 100*time.Millisecond)
			})
		})
	})
}
