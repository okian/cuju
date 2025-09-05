package config_test

import (
	"runtime"
	"testing"

	"github.com/okian/cuju/internal/config"
	"github.com/smartystreets/goconvey/convey"
)

func TestConfig_New(t *testing.T) {
	convey.Convey("Given a new config with default options", t, func() {
		cfg := config.New()

		convey.Convey("Then it should have sensible defaults", func() {
			convey.So(cfg.Addr, convey.ShouldEqual, ":9080")
			convey.So(cfg.EventQueueSize, convey.ShouldEqual, 200_000)
			convey.So(cfg.WorkerCount, convey.ShouldEqual, runtime.NumCPU()*20)
			convey.So(cfg.DedupeSize, convey.ShouldEqual, 500_000)
			convey.So(cfg.ScoringLatencyMinMS, convey.ShouldEqual, 80)
			convey.So(cfg.ScoringLatencyMaxMS, convey.ShouldEqual, 150)
		})
	})
}
