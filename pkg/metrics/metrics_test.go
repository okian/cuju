package metrics

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMetricsOptions(t *testing.T) {
	Convey("Given metrics options", t, func() {
		Convey("When creating options", func() {
			namespaceOpt := WithNamespace("test-namespace")
			subsystemOpt := WithSubsystem("test-subsystem")
			metricPrefixOpt := WithMetricPrefix("test-prefix")
			histogramBucketsOpt := WithHistogramBuckets([]float64{0.1, 0.5, 1.0})
			metricsEnabledOpt := WithMetricsEnabled(true)

			Convey("Then they should be valid functions", func() {
				So(namespaceOpt, ShouldNotBeNil)
				So(subsystemOpt, ShouldNotBeNil)
				So(metricPrefixOpt, ShouldNotBeNil)
				So(histogramBucketsOpt, ShouldNotBeNil)
				So(metricsEnabledOpt, ShouldNotBeNil)
			})
		})
	})
}

func TestMetricsManagerCreation(t *testing.T) {
	Convey("Given metrics manager creation", t, func() {
		Convey("When creating with custom options", func() {
			// Use a single instance to avoid registry conflicts
			manager := NewMetricsManager(
				WithNamespace("test-namespace"),
				WithSubsystem("test-subsystem"),
				WithMetricPrefix("test-prefix"),
				WithHistogramBuckets([]float64{0.1, 0.5, 1.0}),
			)

			Convey("Then it should be created successfully", func() {
				So(manager, ShouldNotBeNil)
			})
		})
	})
}
