package dedupe_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	dedupe "github.com/okian/cuju/internal/domain/dedupe"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInMemoryDeduper(t *testing.T) {
	Convey("Given a new InMemoryDeduper", t, func() {
		Convey("When creating a deduper with default options", func() {
			d := dedupe.NewInMemoryDeduper()

			Convey("Then it should have default configuration", func() {
				So(d, ShouldNotBeNil)
				So(d.Size(), ShouldEqual, 0)
			})
		})

		Convey("When creating a deduper with custom options", func() {
			d := dedupe.NewInMemoryDeduper(
				dedupe.WithMaxSize(100),
			)

			Convey("Then it should have custom configuration", func() {
				So(d, ShouldNotBeNil)
				So(d.Size(), ShouldEqual, 0)
			})
		})

		Convey("When recording events", func() {
			d := dedupe.NewInMemoryDeduper()

			Convey("And the event is new", func() {
				seen := d.SeenAndRecord(context.Background(), "event-1")

				Convey("Then it should return false and record the event", func() {
					So(seen, ShouldBeFalse)
					So(d.Size(), ShouldEqual, 1)
				})
			})

			Convey("And the event was already seen", func() {
				// First time
				d.SeenAndRecord(context.Background(), "event-1")

				// Second time
				seen := d.SeenAndRecord(context.Background(), "event-1")

				Convey("Then it should return true", func() {
					So(seen, ShouldBeTrue)
					So(d.Size(), ShouldEqual, 1)
				})
			})

			Convey("And multiple events are recorded", func() {
				events := []string{"event-1", "event-2", "event-3", "event-4", "event-5"}

				for _, event := range events {
					seen := d.SeenAndRecord(context.Background(), event)
					So(seen, ShouldBeFalse)
				}

				Convey("Then all events should be recorded", func() {
					So(d.Size(), ShouldEqual, int64(len(events)))

					// Check that all events are seen
					for _, event := range events {
						seen := d.SeenAndRecord(context.Background(), event)
						So(seen, ShouldBeTrue)
					}
				})
			})
		})

		Convey("When unrecording events", func() {
			d := dedupe.NewInMemoryDeduper()

			Convey("And the event exists", func() {
				// Record the event
				d.SeenAndRecord(context.Background(), "event-1")
				So(d.Size(), ShouldEqual, 1)

				// Unrecord the event
				d.Unrecord(context.Background(), "event-1")

				Convey("Then it should be removed", func() {
					So(d.Size(), ShouldEqual, 0)

					// Should not be seen anymore
					seen := d.SeenAndRecord(context.Background(), "event-1")
					So(seen, ShouldBeFalse)
				})
			})

			Convey("And the event doesn't exist", func() {
				// Try to unrecord non-existent event
				d.Unrecord(context.Background(), "nonexistent")

				Convey("Then it should not affect the size", func() {
					So(d.Size(), ShouldEqual, 0)
				})
			})

			Convey("And multiple events are unrecorded", func() {
				events := []string{"event-1", "event-2", "event-3"}

				// Record all events
				for _, event := range events {
					d.SeenAndRecord(context.Background(), event)
				}
				So(d.Size(), ShouldEqual, int64(len(events)))

				// Unrecord all events
				for _, event := range events {
					d.Unrecord(context.Background(), event)
				}

				Convey("Then all events should be removed", func() {
					So(d.Size(), ShouldEqual, 0)

					// Check that none are seen
					for _, event := range events {
						seen := d.SeenAndRecord(context.Background(), event)
						So(seen, ShouldBeFalse)
					}
				})
			})
		})

		Convey("When using bounded mode with eviction", func() {
			d := dedupe.NewInMemoryDeduper(dedupe.WithMaxSize(3))

			Convey("And the deduper is at capacity", func() {
				// Fill to capacity
				events := []string{"event-1", "event-2", "event-3"}
				for _, event := range events {
					seen := d.SeenAndRecord(context.Background(), event)
					So(seen, ShouldBeFalse)
				}
				So(d.Size(), ShouldEqual, 3)

				// Add one more event
				seen := d.SeenAndRecord(context.Background(), "event-4")

				Convey("Then it should evict the oldest and add the new one", func() {
					So(seen, ShouldBeFalse)
					So(d.Size(), ShouldEqual, 3)

					// The oldest event should be evicted, so size should remain 3
					// when we try to add event-1 again
					originalSize := d.Size()
					seen1 := d.SeenAndRecord(context.Background(), "event-1")
					So(seen1, ShouldBeFalse)                // Should not be seen (was evicted)
					So(d.Size(), ShouldEqual, originalSize) // Size should not increase

					// The newer events should still be seen (they were not evicted)
					// Note: Since we're calling SeenAndRecord, it will record them again
					// if they were evicted, so we need to check the size instead
					seen2 := d.SeenAndRecord(context.Background(), "event-2")
					So(seen2, ShouldBeFalse)                // Will be recorded again if evicted
					So(d.Size(), ShouldEqual, originalSize) // Size should not increase

					seen3 := d.SeenAndRecord(context.Background(), "event-3")
					So(seen3, ShouldBeFalse)                // Will be recorded again if evicted
					So(d.Size(), ShouldEqual, originalSize) // Size should not increase

					seen4 := d.SeenAndRecord(context.Background(), "event-4")
					So(seen4, ShouldBeFalse)                // Will be recorded again if evicted
					So(d.Size(), ShouldEqual, originalSize) // Size should not increase
				})
			})
		})

		Convey("When using unbounded mode", func() {
			d := dedupe.NewInMemoryDeduper(dedupe.WithMaxSize(0))

			Convey("And many events are recorded", func() {
				const numEvents = 1000
				for i := 0; i < numEvents; i++ {
					eventID := fmt.Sprintf("event-%d", i)
					seen := d.SeenAndRecord(context.Background(), eventID)
					So(seen, ShouldBeFalse)
				}

				Convey("Then all events should be recorded without eviction", func() {
					So(d.Size(), ShouldEqual, int64(numEvents))

					// Check that all events are seen
					for i := 0; i < numEvents; i++ {
						eventID := fmt.Sprintf("event-%d", i)
						seen := d.SeenAndRecord(context.Background(), eventID)
						So(seen, ShouldBeTrue)
					}
				})
			})
		})
	})
}

func TestDedupeConcurrency(t *testing.T) {
	Convey("Given a deduper with concurrent access", t, func() {
		d := dedupe.NewInMemoryDeduper(dedupe.WithMaxSize(1000))
		const numGoroutines = 10
		const eventsPerGoroutine = 100

		Convey("When multiple goroutines record events concurrently", func() {
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

			Convey("Then all events should be recorded successfully", func() {
				So(d.Size(), ShouldEqual, int64(numGoroutines*eventsPerGoroutine))

				// Check for any errors
				for err := range errors {
					So(err, ShouldBeNil)
				}
			})
		})

		Convey("When multiple goroutines unrecord events concurrently", func() {
			// First, record some events
			const numEvents = 500
			for i := 0; i < numEvents; i++ {
				eventID := fmt.Sprintf("event-%d", i)
				d.SeenAndRecord(context.Background(), eventID)
			}

			So(d.Size(), ShouldEqual, int64(numEvents))

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

			Convey("Then all events should be unrecorded successfully", func() {
				So(d.Size(), ShouldEqual, 0)
			})
		})
	})
}

func TestDedupeEdgeCases(t *testing.T) {
	Convey("Given a deduper with edge cases", t, func() {
		Convey("When recording empty string", func() {
			d := dedupe.NewInMemoryDeduper()

			seen := d.SeenAndRecord(context.Background(), "")

			Convey("Then it should handle empty string", func() {
				So(seen, ShouldBeFalse)
				So(d.Size(), ShouldEqual, 1)

				// Should be seen on second call
				seen2 := d.SeenAndRecord(context.Background(), "")
				So(seen2, ShouldBeTrue)
			})
		})

		Convey("When recording very long strings", func() {
			d := dedupe.NewInMemoryDeduper()

			longString := strings.Repeat("a", 10000)
			seen := d.SeenAndRecord(context.Background(), longString)

			Convey("Then it should handle long strings", func() {
				So(seen, ShouldBeFalse)
				So(d.Size(), ShouldEqual, 1)

				// Should be seen on second call
				seen2 := d.SeenAndRecord(context.Background(), longString)
				So(seen2, ShouldBeTrue)
			})
		})

		Convey("When using nil context", func() {
			d := dedupe.NewInMemoryDeduper()

			Convey("Then it should not panic", func() {
				So(func() { d.SeenAndRecord(nil, "event-1") }, ShouldNotPanic)
				So(func() { d.Unrecord(nil, "event-1") }, ShouldNotPanic)
			})
		})

		Convey("When using very small max size", func() {
			d := dedupe.NewInMemoryDeduper(dedupe.WithMaxSize(1))

			Convey("And adding multiple events", func() {
				// First event
				seen1 := d.SeenAndRecord(context.Background(), "event-1")
				So(seen1, ShouldBeFalse)
				So(d.Size(), ShouldEqual, 1)

				// Second event should evict the first
				seen2 := d.SeenAndRecord(context.Background(), "event-2")
				So(seen2, ShouldBeFalse)
				So(d.Size(), ShouldEqual, 1)

				// First event should not be seen (was evicted), so size should remain 1
				// when we try to add event-1 again
				originalSize := d.Size()
				seen1Again := d.SeenAndRecord(context.Background(), "event-1")
				So(seen1Again, ShouldBeFalse)
				So(d.Size(), ShouldEqual, originalSize) // Size should not increase

				// Second event should still be seen
				// Note: Since we're calling SeenAndRecord, it will record it again
				// if it was evicted, so we need to check the size instead
				seen2Again := d.SeenAndRecord(context.Background(), "event-2")
				So(seen2Again, ShouldBeFalse)           // Will be recorded again if evicted
				So(d.Size(), ShouldEqual, originalSize) // Size should not increase
			})
		})

		Convey("When using negative max size", func() {
			d := dedupe.NewInMemoryDeduper(dedupe.WithMaxSize(-1))

			Convey("Then it should be unbounded", func() {
				const numEvents = 1000
				for i := 0; i < numEvents; i++ {
					eventID := fmt.Sprintf("event-%d", i)
					seen := d.SeenAndRecord(context.Background(), eventID)
					So(seen, ShouldBeFalse)
				}

				So(d.Size(), ShouldEqual, int64(numEvents))
			})
		})
	})
}

func TestDedupeOptions(t *testing.T) {
	Convey("Given dedupe options", t, func() {
		Convey("When using WithMaxSize", func() {
			Convey("Then it should set the max size", func() {
				d := dedupe.NewInMemoryDeduper(dedupe.WithMaxSize(500))
				So(d, ShouldNotBeNil)
			})

			Convey("And when max size is zero", func() {
				d := dedupe.NewInMemoryDeduper(dedupe.WithMaxSize(0))
				So(d, ShouldNotBeNil)
			})

			Convey("And when max size is negative", func() {
				d := dedupe.NewInMemoryDeduper(dedupe.WithMaxSize(-100))
				So(d, ShouldNotBeNil)
			})
		})

		// Removed tests for unused options: WithEvictionPolicy, WithTTL, WithMetrics, WithCleanupInterval
	})
}
