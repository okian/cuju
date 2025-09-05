package service_test

import (
	"context"
	"testing"
	"time"

	service "github.com/okian/cuju/internal/app"
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

func TestService_New(t *testing.T) {
	convey.Convey("Given a new service with default options", t, func() {
		svc := service.New()

		convey.Convey("Then it should have sensible defaults", func() {
			convey.So(svc, convey.ShouldNotBeNil)
		})
	})

	convey.Convey("Given a new service with custom options", t, func() {
		svc := service.New(
			service.WithWorkerCount(8),
			service.WithQueueSize(50_000),
			service.WithDedupeSize(25_000),
		)

		convey.Convey("Then it should be created successfully", func() {
			convey.So(svc, convey.ShouldNotBeNil)
		})
	})
}

func TestService_Start(t *testing.T) {
	convey.Convey("Given a new service", t, func() {
		svc := service.New()
		// Ensure service is stopped after test
		defer svc.Stop()

		convey.Convey("When starting the service", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := svc.Start(ctx)

			convey.Convey("Then it should start successfully", func() {
				convey.So(err, convey.ShouldBeNil)
			})

			convey.Convey("And it should be marked as started", func() {
				stats := svc.GetStats()
				convey.So(stats["started"], convey.ShouldEqual, true)
			})
		})
	})
}

func TestService_Stop(t *testing.T) {
	convey.Convey("Given a started service", t, func() {
		svc := service.New()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := svc.Start(ctx)
		convey.So(err, convey.ShouldBeNil)

		convey.Convey("When stopping the service", func() {
			svc.Stop()

			convey.Convey("Then it should be marked as stopped", func() {
				stats := svc.GetStats()
				convey.So(stats["started"], convey.ShouldEqual, false)
			})
		})
	})
}

func TestService_SeenAndRecord(t *testing.T) {
	convey.Convey("Given a started service", t, func() {
		svc := service.New()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := svc.Start(ctx)
		convey.So(err, convey.ShouldBeNil)
		// Ensure service is stopped after test
		defer svc.Stop()

		convey.Convey("When checking a new event ID", func() {
			eventID := "event-123"
			seen := svc.SeenAndRecord(ctx, eventID)

			convey.Convey("Then it should not have been seen before", func() {
				convey.So(seen, convey.ShouldBeFalse)
			})
		})

		convey.Convey("When checking the same event ID again", func() {
			eventID := "event-456"
			svc.SeenAndRecord(ctx, eventID)         // First time
			seen := svc.SeenAndRecord(ctx, eventID) // Second time

			convey.Convey("Then it should have been seen before", func() {
				convey.So(seen, convey.ShouldBeTrue)
			})
		})
	})
}

func TestService_Enqueue(t *testing.T) {
	convey.Convey("Given a started service", t, func() {
		svc := service.New()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := svc.Start(ctx)
		convey.So(err, convey.ShouldBeNil)
		// Ensure service is stopped after test
		defer svc.Stop()

		convey.Convey("When enqueueing a valid event", func() {
			event := struct {
				EventID   string
				TalentID  string
				RawMetric float64
				Skill     string
			}{
				EventID:   "event-123",
				TalentID:  "talent-456",
				RawMetric: 85.5,
				Skill:     "coding",
			}

			success := svc.Enqueue(ctx, event)

			convey.Convey("Then it should be enqueued successfully", func() {
				convey.So(success, convey.ShouldBeTrue)
			})
		})
	})
}

func TestService_GetStats(t *testing.T) {
	convey.Convey("Given a new service", t, func() {
		svc := service.New()

		convey.Convey("When getting stats before starting", func() {
			stats := svc.GetStats()

			convey.Convey("Then it should return basic stats", func() {
				convey.So(stats, convey.ShouldNotBeNil)
				convey.So(stats["started"], convey.ShouldEqual, false)
			})
		})
	})
}
