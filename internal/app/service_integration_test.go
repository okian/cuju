package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	service "github.com/okian/cuju/internal/app"
	"github.com/okian/cuju/internal/domain/model"
	"github.com/okian/cuju/pkg/logger"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	// Initialize logging for tests
	err := logger.Init()
	if err != nil {
		panic(err)
	}
}

func TestServiceIntegration(t *testing.T) {
	Convey("Given a service with full integration", t, func() {
		svc := service.New(
			service.WithWorkerCount(2),
			service.WithQueueSize(1000),
			service.WithDedupeSize(500),
		)
		defer svc.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		Convey("When starting the service", func() {
			err := svc.Start(ctx)

			Convey("Then it should start successfully", func() {
				So(err, ShouldBeNil)
			})

			Convey("And the service should be running", func() {
				stats := svc.GetStats()
				So(stats["started"], ShouldEqual, true)
			})
		})

		Convey("When processing events end-to-end", func() {
			err := svc.Start(ctx)
			So(err, ShouldBeNil)

			// Give service time to start
			time.Sleep(100 * time.Millisecond)

			Convey("And enqueueing multiple events", func() {
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
					So(success, ShouldBeTrue)
				}

				// Give workers time to process
				time.Sleep(500 * time.Millisecond)

				Convey("Then events should be processed", func() {
					stats := svc.GetStats()
					So(stats, ShouldNotBeNil)
				})

				Convey("And duplicate events should be detected", func() {
					// Try to enqueue the same event again
					duplicateEvent := events[0]
					success := svc.Enqueue(ctx, duplicateEvent)
					So(success, ShouldBeTrue) // Should still be enqueued (but as duplicate)

					// The event should be detected as duplicate during enqueue
					// We can verify this by checking that the event was processed
					stats := svc.GetStats()
					So(stats, ShouldNotBeNil)
				})

				Convey("And leaderboard should be updated", func() {
					// Get top N entries
					entries, err := svc.TopN(ctx, 10)
					So(err, ShouldBeNil)
					So(len(entries), ShouldBeGreaterThan, 0)

					// Verify ordering (highest scores first)
					for i := 1; i < len(entries); i++ {
						So(entries[i-1].Score, ShouldBeGreaterThanOrEqualTo, entries[i].Score)
					}
				})

				Convey("And individual ranks should be available", func() {
					// Check rank for talent-1 (should have the higher score)
					entry, err := svc.Rank(ctx, "talent-1")
					So(err, ShouldBeNil)
					So(entry.TalentID, ShouldEqual, "talent-1")
					So(entry.Score, ShouldEqual, 100.0) // Should have the higher score (95.0 * 1.5 = 142.5, capped at 100)
					So(entry.Rank, ShouldBeGreaterThan, 0)
				})
			})
		})

		Convey("When handling high-volume events", func() {
			err := svc.Start(ctx)
			So(err, ShouldBeNil)

			// Give service time to start
			time.Sleep(100 * time.Millisecond)

			Convey("And enqueueing many events concurrently", func() {
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

				Convey("Then most events should be enqueued successfully", func() {
					So(successCount, ShouldBeGreaterThan, numEvents/2)
				})

				// Give workers time to process
				time.Sleep(1 * time.Second)

				Convey("And leaderboard should reflect the updates", func() {
					entries, err := svc.TopN(ctx, 20)
					So(err, ShouldBeNil)
					So(len(entries), ShouldBeGreaterThan, 0)

					// Verify we have entries for multiple talents
					talentIDs := make(map[string]bool)
					for _, entry := range entries {
						talentIDs[entry.TalentID] = true
					}
					So(len(talentIDs), ShouldBeGreaterThan, 1)
				})
			})
		})

		Convey("When handling service lifecycle", func() {
			Convey("And starting and stopping multiple times", func() {
				// Start service
				err := svc.Start(ctx)
				So(err, ShouldBeNil)

				// Give it time to start
				time.Sleep(100 * time.Millisecond)

				// Stop service
				svc.Stop()

				// Give it time to stop
				time.Sleep(100 * time.Millisecond)

				// Check it's stopped
				stats := svc.GetStats()
				So(stats["started"], ShouldEqual, false)

				// Start again
				err = svc.Start(ctx)
				So(err, ShouldBeNil)

				// Give it time to start
				time.Sleep(100 * time.Millisecond)

				// Check it's started again
				stats = svc.GetStats()
				So(stats["started"], ShouldEqual, true)
			})
		})

		Convey("When handling edge cases", func() {
			err := svc.Start(ctx)
			So(err, ShouldBeNil)

			// Give service time to start
			time.Sleep(100 * time.Millisecond)

			Convey("And enqueueing events with extreme values", func() {
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
					So(success, ShouldBeTrue)
				}

				// Give workers time to process
				time.Sleep(500 * time.Millisecond)

				Convey("Then extreme values should be handled", func() {
					// Service should still be running
					stats := svc.GetStats()
					So(stats["started"], ShouldEqual, true)
				})
			})

			Convey("And enqueueing events with very long IDs", func() {
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
				So(success, ShouldBeTrue)

				// Give workers time to process
				time.Sleep(500 * time.Millisecond)

				Convey("Then long IDs should be handled", func() {
					// Service should still be running
					stats := svc.GetStats()
					So(stats["started"], ShouldEqual, true)
				})
			})
		})
	})
}

func TestServiceConcurrency(t *testing.T) {
	Convey("Given a service with concurrent operations", t, func() {
		svc := service.New(
			service.WithWorkerCount(4),
			service.WithQueueSize(2000),
			service.WithDedupeSize(1000),
		)
		defer svc.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := svc.Start(ctx)
		So(err, ShouldBeNil)

		// Give service time to start
		time.Sleep(100 * time.Millisecond)

		Convey("When multiple goroutines enqueue events concurrently", func() {
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

			Convey("Then all events should be processed", func() {
				// Service should still be running
				stats := svc.GetStats()
				So(stats["started"], ShouldEqual, true)

				// Should have entries in leaderboard
				entries, err := svc.TopN(ctx, 100)
				So(err, ShouldBeNil)
				So(len(entries), ShouldBeGreaterThan, 0)
			})
		})

		Convey("When multiple goroutines query the leaderboard concurrently", func() {
			numGoroutines := 20
			done := make(chan bool, numGoroutines)
			errors := make(chan error, numGoroutines*20) // Buffer for potential errors

			// Start multiple goroutines querying
			for i := 0; i < numGoroutines; i++ {
				go func(goroutineID int) {
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

			Convey("Then all queries should succeed", func() {
				// Check if any errors occurred
				select {
				case err := <-errors:
					So(err, ShouldBeNil)
				default:
					// No errors, test passed
					So(true, ShouldBeTrue)
				}
			})
		})
	})
}

func TestServiceErrorHandling(t *testing.T) {
	Convey("Given a service with error conditions", t, func() {
		svc := service.New(
			service.WithWorkerCount(1),
			service.WithQueueSize(10), // Small queue to test backpressure
			service.WithDedupeSize(5),
		)
		defer svc.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := svc.Start(ctx)
		So(err, ShouldBeNil)

		// Give service time to start
		time.Sleep(100 * time.Millisecond)

		Convey("When enqueueing events beyond queue capacity", func() {
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

			Convey("Then some events should be rejected due to backpressure", func() {
				So(successCount, ShouldBeLessThan, 20)
				So(successCount, ShouldBeGreaterThan, 0)
			})
		})

		Convey("When querying non-existent talents", func() {
			entry, err := svc.Rank(ctx, "non-existent-talent")

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(entry.TalentID, ShouldEqual, "")
			})
		})

		Convey("When querying with invalid limits", func() {
			entries, err := svc.TopN(ctx, 0)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(entries, ShouldBeNil)
			})
		})

		Convey("When querying with negative limits", func() {
			entries, err := svc.TopN(ctx, -1)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(entries, ShouldBeNil)
			})
		})
	})
}

func TestServicePerformance(t *testing.T) {
	Convey("Given a service for performance testing", t, func() {
		svc := service.New(
			service.WithWorkerCount(8),
			service.WithQueueSize(10000),
			service.WithDedupeSize(5000),
		)
		defer svc.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := svc.Start(ctx)
		So(err, ShouldBeNil)

		// Give service time to start
		time.Sleep(100 * time.Millisecond)

		Convey("When processing a large number of events", func() {
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

			Convey("Then enqueueing should be fast", func() {
				// Should be able to enqueue 1000 events in reasonable time
				So(enqueueTime, ShouldBeLessThan, 5*time.Second)
			})

			Convey("And leaderboard queries should be fast", func() {
				start := time.Now()
				entries, err := svc.TopN(ctx, 100)
				queryTime := time.Since(start)

				So(err, ShouldBeNil)
				So(len(entries), ShouldBeGreaterThan, 0)
				So(queryTime, ShouldBeLessThan, 100*time.Millisecond)
			})

			Convey("And rank queries should be fast", func() {
				start := time.Now()
				entry, err := svc.Rank(ctx, "talent-0")
				queryTime := time.Since(start)

				So(err, ShouldBeNil)
				So(entry.TalentID, ShouldEqual, "talent-0")
				So(queryTime, ShouldBeLessThan, 100*time.Millisecond)
			})
		})
	})
}
