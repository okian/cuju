package config_test

import (
	"runtime"
	"testing"

	"github.com/okian/cuju/internal/config"
	. "github.com/smartystreets/goconvey/convey"
)

func TestConfig_New(t *testing.T) {
	Convey("Given a new config with default options", t, func() {
		cfg := config.New()

		Convey("Then it should have sensible defaults", func() {
			So(cfg.Addr, ShouldEqual, ":9080")
			So(cfg.EventQueueSize, ShouldEqual, 200_000)
			So(cfg.WorkerCount, ShouldEqual, runtime.NumCPU()*10)
			So(cfg.DedupeSize, ShouldEqual, 500_000)
			So(cfg.ScoringLatencyMinMS, ShouldEqual, 80)
			So(cfg.ScoringLatencyMaxMS, ShouldEqual, 150)
		})
	})
}
