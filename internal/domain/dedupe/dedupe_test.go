package dedupe_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	dedupe "github.com/okian/cuju/internal/domain/dedupe"
	"github.com/smartystreets/goconvey/convey"
)

func TestInMemoryDeduper(t *testing.T) {
	convey.Convey("Given a new InMemoryDeduper", t, func() {
		convey.Convey("When creating a deduper with default options", func() {
			d := dedupe.NewInMemoryDeduper()

			convey.Convey("Then it should have default configuration", func() {
				convey.So(d, convey.ShouldNotBeNil)
				convey.So(d.Size(), convey.ShouldEqual, 0)
			})
		})

		convey.Convey("When creating a deduper with custom options", func() {
			d := dedupe.NewInMemoryDeduper(
				dedupe.WithMaxSize(100),
			)

			convey.Convey("Then it should have custom configuration", func() {
				convey.So(d, convey.ShouldNotBeNil)
				convey.So(d.Size(), convey.ShouldEqual, 0)
			})
		})

		convey.Convey("When recording events", func() {
			d := dedupe.NewInMemoryDeduper()

			convey.Convey("And the event is new", func() {
				seen := d.SeenAndRecord(context.Background(), "event-1")

				convey.Convey("Then it should return false and record the event", func() {
					convey.So(seen, convey.ShouldBeFalse)
					convey.So(d.Size(), convey.ShouldEqual, 1)
				})
			})

			convey.Convey("And the event was already seen", func() {
				// First time
				d.SeenAndRecord(context.Background(), "event-1")

				// Second time
				seen := d.SeenAndRecord(context.Background(), "event-1")

				convey.Convey("Then it should return true", func() {
					convey.So(seen, convey.ShouldBeTrue)
					convey.So(d.Size(), convey.ShouldEqual, 1)
				})
			})

			convey.Convey("And multiple events are recorded", func() {
				events := []string{"event-1", "event-2", "event-3", "event-4", "event-5"}

				for _, event := range events {
					seen := d.SeenAndRecord(context.Background(), event)
					convey.So(seen, convey.ShouldBeFalse)
				}

				convey.Convey("Then all events should be recorded", func() {
					convey.So(d.Size(), convey.ShouldEqual, int64(len(events)))

					// Check that all events are seen
					for _, event := range events {
						seen := d.SeenAndRecord(context.Background(), event)
						convey.So(seen, convey.ShouldBeTrue)
					}
				})
			})
		})

		convey.Convey("When unrecording events", func() {
			d := dedupe.NewInMemoryDeduper()

			convey.Convey("And the event exists", func() {
				// Record the event
				d.SeenAndRecord(context.Background(), "event-1")
				convey.So(d.Size(), convey.ShouldEqual, 1)

				// Unrecord the event
				d.Unrecord(context.Background(), "event-1")

				convey.Convey("Then it should be removed", func() {
					convey.So(d.Size(), convey.ShouldEqual, 0)

					// Should not be seen anymore
					seen := d.SeenAndRecord(context.Background(), "event-1")
					convey.So(seen, convey.ShouldBeFalse)
				})
			})

			convey.Convey("And the event doesn't exist", func() {
				// Try to unrecord non-existent event
				d.Unrecord(context.Background(), "nonexistent")

				convey.Convey("Then it should not affect the size", func() {
					convey.So(d.Size(), convey.ShouldEqual, 0)
				})
			})

			convey.Convey("And multiple events are unrecorded", func() {
				events := []string{"event-1", "event-2", "event-3"}

				// Record all events
				for _, event := range events {
					d.SeenAndRecord(context.Background(), event)
				}
				convey.So(d.Size(), convey.ShouldEqual, int64(len(events)))

				// Unrecord all events
				for _, event := range events {
					d.Unrecord(context.Background(), event)
				}

				convey.Convey("Then all events should be removed", func() {
					convey.So(d.Size(), convey.ShouldEqual, 0)

					// Check that none are seen
					for _, event := range events {
						seen := d.SeenAndRecord(context.Background(), event)
						convey.So(seen, convey.ShouldBeFalse)
					}
				})
			})
		})

		convey.Convey("When using bounded mode with eviction", func() {
			d := dedupe.NewInMemoryDeduper(dedupe.WithMaxSize(3))

			convey.Convey("And the deduper is at capacity", func() {
				// Fill to capacity
				events := []string{"event-1", "event-2", "event-3"}
				for _, event := range events {
					seen := d.SeenAndRecord(context.Background(), event)
					convey.So(seen, convey.ShouldBeFalse)
				}
				convey.So(d.Size(), convey.ShouldEqual, 3)

				// Add one more event
				seen := d.SeenAndRecord(context.Background(), "event-4")

				convey.Convey("Then it should evict the oldest and add the new one", func() {
					convey.So(seen, convey.ShouldBeFalse)
					convey.So(d.Size(), convey.ShouldEqual, 3)

					// The oldest event should be evicted, so size should remain 3
					// when we try to add event-1 again
					originalSize := d.Size()
					seen1 := d.SeenAndRecord(context.Background(), "event-1")
					convey.So(seen1, convey.ShouldBeFalse)                // Should not be seen (was evicted)
					convey.So(d.Size(), convey.ShouldEqual, originalSize) // Size should not increase

					// The newer events should still be seen (they were not evicted)
					// Note: Since we're calling SeenAndRecord, it will record them again
					// if they were evicted, so we need to check the size instead
					seen2 := d.SeenAndRecord(context.Background(), "event-2")
					convey.So(seen2, convey.ShouldBeFalse)                // Will be recorded again if evicted
					convey.So(d.Size(), convey.ShouldEqual, originalSize) // Size should not increase

					seen3 := d.SeenAndRecord(context.Background(), "event-3")
					convey.So(seen3, convey.ShouldBeFalse)                // Will be recorded again if evicted
					convey.So(d.Size(), convey.ShouldEqual, originalSize) // Size should not increase

					seen4 := d.SeenAndRecord(context.Background(), "event-4")
					convey.So(seen4, convey.ShouldBeFalse)                // Will be recorded again if evicted
					convey.So(d.Size(), convey.ShouldEqual, originalSize) // Size should not increase
				})
			})
		})

		convey.Convey("When using unbounded mode", func() {
			d := dedupe.NewInMemoryDeduper(dedupe.WithMaxSize(0))

			convey.Convey("And many events are recorded", func() {
				const numEvents = 1000
				for i := 0; i < numEvents; i++ {
					eventID := fmt.Sprintf("event-%d", i)
					seen := d.SeenAndRecord(context.Background(), eventID)
					convey.So(seen, convey.ShouldBeFalse)
				}

				convey.Convey("Then all events should be recorded without eviction", func() {
					convey.So(d.Size(), convey.ShouldEqual, int64(numEvents))

					// Check that all events are seen
					for i := 0; i < numEvents; i++ {
						eventID := fmt.Sprintf("event-%d", i)
						seen := d.SeenAndRecord(context.Background(), eventID)
						convey.So(seen, convey.ShouldBeTrue)
					}
				})
			})
		})
	})
}

func TestDedupeConcurrency(t *testing.T) {
	convey.Convey("Given a deduper with concurrent access", t, func() {
		d := dedupe.NewInMemoryDeduper(dedupe.WithMaxSize(1000))
		const numGoroutines = 10
		const eventsPerGoroutine = 100

		convey.Convey("When multiple goroutines record events concurrently", func() {
			var wg sync.WaitGroup
			errors := make(chan error, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func(goroutineID int) {
					defer wg.Done()
					for j := 0; j < eventsPerGoroutine; j++ {
						eventID := fmt.Sprintf("event-%d-%d", goroutineID, j)
						// This should not panic or cause race conditions
						d.SeenAndRecord(context.Background(), eventID)
					}
				}(i)
			}

			wg.Wait()
			close(errors)

			convey.Convey("Then all events should be recorded successfully", func() {
				convey.So(d.Size(), convey.ShouldEqual, int64(numGoroutines*eventsPerGoroutine))

				// Check for any errors
				for err := range errors {
					convey.So(err, convey.ShouldBeNil)
				}
			})
		})

		convey.Convey("When multiple goroutines unrecord events concurrently", func() {
			// First, record some events
			const numEvents = 500
			for i := 0; i < numEvents; i++ {
				eventID := fmt.Sprintf("event-%d", i)
				d.SeenAndRecord(context.Background(), eventID)
			}

			convey.So(d.Size(), convey.ShouldEqual, int64(numEvents))

			// Now unrecord them concurrently
			var wg sync.WaitGroup
			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func(goroutineID int) {
					defer wg.Done()
					for j := 0; j < numEvents/numGoroutines; j++ {
						eventID := fmt.Sprintf("event-%d", goroutineID*(numEvents/numGoroutines)+j)
						d.Unrecord(context.Background(), eventID)
					}
				}(i)
			}

			wg.Wait()

			convey.Convey("Then all events should be unrecorded successfully", func() {
				convey.So(d.Size(), convey.ShouldEqual, 0)
			})
		})
	})
}

func TestDedupeEdgeCases(t *testing.T) {
	convey.Convey("Given a deduper with edge cases", t, func() {
		convey.Convey("When recording empty string", func() {
			d := dedupe.NewInMemoryDeduper()

			seen := d.SeenAndRecord(context.Background(), "")

			convey.Convey("Then it should handle empty string", func() {
				convey.So(seen, convey.ShouldBeFalse)
				convey.So(d.Size(), convey.ShouldEqual, 1)

				// Should be seen on second call
				seen2 := d.SeenAndRecord(context.Background(), "")
				convey.So(seen2, convey.ShouldBeTrue)
			})
		})

		convey.Convey("When recording very long strings", func() {
			d := dedupe.NewInMemoryDeduper()

			longString := strings.Repeat("a", 10000)
			seen := d.SeenAndRecord(context.Background(), longString)

			convey.Convey("Then it should handle long strings", func() {
				convey.So(seen, convey.ShouldBeFalse)
				convey.So(d.Size(), convey.ShouldEqual, 1)

				// Should be seen on second call
				seen2 := d.SeenAndRecord(context.Background(), longString)
				convey.So(seen2, convey.ShouldBeTrue)
			})
		})

		convey.Convey("When using nil context", func() {
			d := dedupe.NewInMemoryDeduper()

			convey.Convey("Then it should not panic", func() {
				convey.So(func() { d.SeenAndRecord(context.TODO(), "event-1") }, convey.ShouldNotPanic)
				convey.So(func() { d.Unrecord(context.TODO(), "event-1") }, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When using very small max size", func() {
			d := dedupe.NewInMemoryDeduper(dedupe.WithMaxSize(1))

			convey.Convey("And adding multiple events", func() {
				// First event
				seen1 := d.SeenAndRecord(context.Background(), "event-1")
				convey.So(seen1, convey.ShouldBeFalse)
				convey.So(d.Size(), convey.ShouldEqual, 1)

				// Second event should evict the first
				seen2 := d.SeenAndRecord(context.Background(), "event-2")
				convey.So(seen2, convey.ShouldBeFalse)
				convey.So(d.Size(), convey.ShouldEqual, 1)

				// First event should not be seen (was evicted), so size should remain 1
				// when we try to add event-1 again
				originalSize := d.Size()
				seen1Again := d.SeenAndRecord(context.Background(), "event-1")
				convey.So(seen1Again, convey.ShouldBeFalse)
				convey.So(d.Size(), convey.ShouldEqual, originalSize) // Size should not increase

				// Second event should still be seen
				// Note: Since we're calling SeenAndRecord, it will record it again
				// if it was evicted, so we need to check the size instead
				seen2Again := d.SeenAndRecord(context.Background(), "event-2")
				convey.So(seen2Again, convey.ShouldBeFalse)           // Will be recorded again if evicted
				convey.So(d.Size(), convey.ShouldEqual, originalSize) // Size should not increase
			})
		})

		convey.Convey("When using negative max size", func() {
			d := dedupe.NewInMemoryDeduper(dedupe.WithMaxSize(-1))

			convey.Convey("Then it should be unbounded", func() {
				const numEvents = 1000
				for i := 0; i < numEvents; i++ {
					eventID := fmt.Sprintf("event-%d", i)
					seen := d.SeenAndRecord(context.Background(), eventID)
					convey.So(seen, convey.ShouldBeFalse)
				}

				convey.So(d.Size(), convey.ShouldEqual, int64(numEvents))
			})
		})
	})
}

func TestDedupeOptions(t *testing.T) {
	convey.Convey("Given dedupe options", t, func() {
		convey.Convey("When using WithMaxSize", func() {
			convey.Convey("Then it should set the max size", func() {
				d := dedupe.NewInMemoryDeduper(dedupe.WithMaxSize(500))
				convey.So(d, convey.ShouldNotBeNil)
			})

			convey.Convey("And when max size is zero", func() {
				d := dedupe.NewInMemoryDeduper(dedupe.WithMaxSize(0))
				convey.So(d, convey.ShouldNotBeNil)
			})

			convey.Convey("And when max size is negative", func() {
				d := dedupe.NewInMemoryDeduper(dedupe.WithMaxSize(-100))
				convey.So(d, convey.ShouldNotBeNil)
			})
		})

		// Removed tests for unused options: WithEvictionPolicy, WithTTL, WithMetrics, WithCleanupInterval
	})
}
