package service_test

import (
	"context"
	"testing"
	"time"

	service "github.com/okian/cuju/internal/app"
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

func TestService_New(t *testing.T) {
	Convey("Given a new service with default options", t, func() {
		svc := service.New()

		Convey("Then it should have sensible defaults", func() {
			So(svc, ShouldNotBeNil)
		})
	})

	Convey("Given a new service with custom options", t, func() {
		svc := service.New(
			service.WithWorkerCount(8),
			service.WithQueueSize(50_000),
			service.WithDedupeSize(25_000),
			service.WithShardCount(2),
		)

		Convey("Then it should be created successfully", func() {
			So(svc, ShouldNotBeNil)
		})
	})
}

func TestService_Start(t *testing.T) {
	Convey("Given a new service", t, func() {
		svc := service.New()
		// Ensure service is stopped after test
		defer svc.Stop()

		Convey("When starting the service", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := svc.Start(ctx)

			Convey("Then it should start successfully", func() {
				So(err, ShouldBeNil)
			})

			Convey("And it should be marked as started", func() {
				stats := svc.GetStats()
				So(stats["started"], ShouldEqual, true)
			})
		})
	})
}

func TestService_Stop(t *testing.T) {
	Convey("Given a started service", t, func() {
		svc := service.New()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := svc.Start(ctx)
		So(err, ShouldBeNil)

		Convey("When stopping the service", func() {
			svc.Stop()

			Convey("Then it should be marked as stopped", func() {
				stats := svc.GetStats()
				So(stats["started"], ShouldEqual, false)
			})
		})
	})
}

func TestService_SeenAndRecord(t *testing.T) {
	Convey("Given a started service", t, func() {
		svc := service.New()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := svc.Start(ctx)
		So(err, ShouldBeNil)
		// Ensure service is stopped after test
		defer svc.Stop()

		Convey("When checking a new event ID", func() {
			eventID := "event-123"
			seen := svc.SeenAndRecord(ctx, eventID)

			Convey("Then it should not have been seen before", func() {
				So(seen, ShouldBeFalse)
			})
		})

		Convey("When checking the same event ID again", func() {
			eventID := "event-456"
			svc.SeenAndRecord(ctx, eventID)         // First time
			seen := svc.SeenAndRecord(ctx, eventID) // Second time

			Convey("Then it should have been seen before", func() {
				So(seen, ShouldBeTrue)
			})
		})
	})
}

func TestService_Enqueue(t *testing.T) {
	Convey("Given a started service", t, func() {
		svc := service.New()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := svc.Start(ctx)
		So(err, ShouldBeNil)
		// Ensure service is stopped after test
		defer svc.Stop()

		Convey("When enqueueing a valid event", func() {
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

			Convey("Then it should be enqueued successfully", func() {
				So(success, ShouldBeTrue)
			})
		})
	})
}

func TestService_GetStats(t *testing.T) {
	Convey("Given a new service", t, func() {
		svc := service.New()

		Convey("When getting stats before starting", func() {
			stats := svc.GetStats()

			Convey("Then it should return basic stats", func() {
				So(stats, ShouldNotBeNil)
				So(stats["started"], ShouldEqual, false)
			})
		})
	})
}
