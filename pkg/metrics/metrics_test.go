package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
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
			refreshIntervalOpt := WithRefreshInterval(5 * time.Second)
			customLabelsOpt := WithCustomLabels(map[string]string{"env": "test"})

			Convey("Then they should be valid functions", func() {
				So(namespaceOpt, ShouldNotBeNil)
				So(subsystemOpt, ShouldNotBeNil)
				So(metricPrefixOpt, ShouldNotBeNil)
				So(histogramBucketsOpt, ShouldNotBeNil)
				So(metricsEnabledOpt, ShouldNotBeNil)
				So(refreshIntervalOpt, ShouldNotBeNil)
				So(customLabelsOpt, ShouldNotBeNil)
			})
		})
	})
}

func TestMetricsManagerCreation(t *testing.T) {
	Convey("Given metrics manager creation", t, func() {
		Convey("When creating with default options", func() {
			registry := prometheus.NewRegistry()
			manager := NewMetricsManager(WithPrometheusRegistry(registry))

			Convey("Then it should be created successfully", func() {
				So(manager, ShouldNotBeNil)
			})
		})

		Convey("When creating with custom options", func() {
			registry := prometheus.NewRegistry()
			manager := NewMetricsManager(
				WithNamespace("test-namespace"),
				WithSubsystem("test-subsystem"),
				WithMetricPrefix("test-prefix"),
				WithHistogramBuckets([]float64{0.1, 0.5, 1.0}),
				WithMetricsEnabled(true),
				WithRefreshInterval(10*time.Second),
				WithCustomLabels(map[string]string{"env": "test", "version": "1.0"}),
				WithPrometheusRegistry(registry),
			)

			Convey("Then it should be created successfully", func() {
				So(manager, ShouldNotBeNil)
			})
		})

		Convey("When creating with custom registry", func() {
			registry := prometheus.NewRegistry()
			manager := NewMetricsManager(WithPrometheusRegistry(registry))

			Convey("Then it should be created successfully", func() {
				So(manager, ShouldNotBeNil)
			})
		})
	})
}

func TestMetricsRecording(t *testing.T) {
	Convey("Given metrics recording", t, func() {
		Convey("When recording event metrics", func() {
			Convey("Then it should record processed events", func() {
				So(func() {
					RecordEventProcessed()
					RecordEventProcessed()
					RecordEventProcessed()
				}, ShouldNotPanic)
			})

			Convey("And it should record duplicate events", func() {
				So(func() {
					RecordEventDuplicate()
					RecordEventDuplicate()
				}, ShouldNotPanic)
			})

			Convey("And it should record scoring latency", func() {
				So(func() {
					RecordScoringLatency(100.0)
					RecordScoringLatency(150.0)
					RecordScoringLatency(200.0)
				}, ShouldNotPanic)
			})

			Convey("And it should record leaderboard updates", func() {
				So(func() {
					RecordLeaderboardUpdate()
					RecordLeaderboardUpdate()
				}, ShouldNotPanic)
			})
		})

		Convey("When recording operational metrics", func() {
			Convey("Then it should update queue size", func() {
				So(func() {
					UpdateQueueSize(1000)
					UpdateQueueSize(2000)
					UpdateQueueSize(500)
				}, ShouldNotPanic)
			})

			Convey("And it should update worker count", func() {
				So(func() {
					UpdateWorkerCount(8)
					UpdateWorkerCount(16)
					UpdateWorkerCount(4)
				}, ShouldNotPanic)
			})

			Convey("And it should update total talents", func() {
				So(func() {
					UpdateTotalTalents(10000)
					UpdateTotalTalents(15000)
					UpdateTotalTalents(20000)
				}, ShouldNotPanic)
			})
		})

		Convey("When recording HTTP metrics", func() {
			Convey("Then it should record HTTP requests", func() {
				So(func() {
					RecordHTTPRequest("/healthz", "GET", "200")
					RecordHTTPRequest("/events", "POST", "202")
					RecordHTTPRequest("/leaderboard", "GET", "200")
				}, ShouldNotPanic)
			})

			Convey("And it should record HTTP request duration", func() {
				So(func() {
					RecordHTTPRequestDuration("/healthz", "GET", "200", 5.0)
					RecordHTTPRequestDuration("/events", "POST", "202", 10.0)
					RecordHTTPRequestDuration("/leaderboard", "GET", "200", 15.0)
				}, ShouldNotPanic)
			})
		})

		Convey("When recording error metrics", func() {
			Convey("Then it should record scoring errors", func() {
				So(func() {
					RecordScoringError()
					RecordScoringError()
				}, ShouldNotPanic)
			})

			Convey("And it should record leaderboard errors", func() {
				So(func() {
					RecordLeaderboardError()
					RecordLeaderboardError()
				}, ShouldNotPanic)
			})

			Convey("And it should record errors by component", func() {
				So(func() {
					RecordErrorByComponent("scoring", "timeout")
					RecordErrorByComponent("repository", "connection_failed")
					RecordErrorByComponent("queue", "full")
				}, ShouldNotPanic)
			})

			Convey("And it should record errors by type", func() {
				So(func() {
					RecordErrorByType("timeout", "error")
					RecordErrorByType("connection_failed", "error")
					RecordErrorByType("validation_error", "warning")
				}, ShouldNotPanic)
			})

			Convey("And it should record errors by endpoint", func() {
				So(func() {
					RecordErrorByEndpoint("/events", "POST", "timeout")
					RecordErrorByEndpoint("/leaderboard", "GET", "not_found")
					RecordErrorByEndpoint("/rank", "GET", "validation_error")
				}, ShouldNotPanic)
			})

			Convey("And it should record error latency", func() {
				So(func() {
					RecordErrorLatency("scoring", "timeout", 100.0)
					RecordErrorLatency("repository", "connection_failed", 200.0)
					RecordErrorLatency("queue", "full", 50.0)
				}, ShouldNotPanic)
			})
		})

		Convey("When recording repository metrics", func() {
			Convey("Then it should update repository shard count", func() {
				So(func() {
					UpdateRepositoryShardCount(4)
					UpdateRepositoryShardCount(8)
					UpdateRepositoryShardCount(16)
				}, ShouldNotPanic)
			})

			Convey("And it should update repository records total", func() {
				So(func() {
					UpdateRepositoryRecordsTotal(100000)
					UpdateRepositoryRecordsTotal(200000)
					UpdateRepositoryRecordsTotal(500000)
				}, ShouldNotPanic)
			})

			Convey("And it should update repository records per shard", func() {
				So(func() {
					UpdateRepositoryRecordsPerShard("shard-0", 25000)
					UpdateRepositoryRecordsPerShard("shard-1", 30000)
					UpdateRepositoryRecordsPerShard("shard-2", 20000)
				}, ShouldNotPanic)
			})

			Convey("And it should update repository shard utilization", func() {
				So(func() {
					UpdateRepositoryShardUtilization("shard-0", 0.75)
					UpdateRepositoryShardUtilization("shard-1", 0.85)
					UpdateRepositoryShardUtilization("shard-2", 0.65)
				}, ShouldNotPanic)
			})

			Convey("And it should record repository update latency", func() {
				So(func() {
					RecordRepositoryUpdateLatency(5.0)
					RecordRepositoryUpdateLatency(10.0)
					RecordRepositoryUpdateLatency(15.0)
				}, ShouldNotPanic)
			})

			Convey("And it should record repository query latency", func() {
				So(func() {
					RecordRepositoryQueryLatency(2.0)
					RecordRepositoryQueryLatency(5.0)
					RecordRepositoryQueryLatency(8.0)
				}, ShouldNotPanic)
			})
		})

		Convey("When recording queue metrics", func() {
			Convey("Then it should update queue capacity", func() {
				So(func() {
					UpdateQueueCapacity(10000)
					UpdateQueueCapacity(20000)
					UpdateQueueCapacity(50000)
				}, ShouldNotPanic)
			})

			Convey("And it should update queue utilization", func() {
				So(func() {
					UpdateQueueUtilization(0.5)
					UpdateQueueUtilization(0.75)
					UpdateQueueUtilization(0.9)
				}, ShouldNotPanic)
			})

			Convey("And it should record queue enqueue", func() {
				So(func() {
					RecordQueueEnqueue()
					RecordQueueEnqueue()
					RecordQueueEnqueue()
				}, ShouldNotPanic)
			})

			Convey("And it should record queue dequeue", func() {
				So(func() {
					RecordQueueDequeue()
					RecordQueueDequeue()
					RecordQueueDequeue()
				}, ShouldNotPanic)
			})

			Convey("And it should record queue enqueue errors", func() {
				So(func() {
					RecordQueueEnqueueError()
					RecordQueueEnqueueError()
				}, ShouldNotPanic)
			})

			Convey("And it should record queue processing latency", func() {
				So(func() {
					RecordQueueProcessingLatency(20.0)
					RecordQueueProcessingLatency(30.0)
					RecordQueueProcessingLatency(40.0)
				}, ShouldNotPanic)
			})
		})

		Convey("When recording worker metrics", func() {
			Convey("Then it should update worker active count", func() {
				So(func() {
					UpdateWorkerActiveCount(4)
					UpdateWorkerActiveCount(8)
					UpdateWorkerActiveCount(12)
				}, ShouldNotPanic)
			})

			Convey("And it should update worker idle count", func() {
				So(func() {
					UpdateWorkerIdleCount(2)
					UpdateWorkerIdleCount(4)
					UpdateWorkerIdleCount(6)
				}, ShouldNotPanic)
			})

			Convey("And it should update worker messages per second", func() {
				So(func() {
					UpdateWorkerMessagesPerSecond(100.0)
					UpdateWorkerMessagesPerSecond(200.0)
					UpdateWorkerMessagesPerSecond(300.0)
				}, ShouldNotPanic)
			})

			Convey("And it should record worker processing latency", func() {
				So(func() {
					RecordWorkerProcessingLatency(50.0)
					RecordWorkerProcessingLatency(75.0)
					RecordWorkerProcessingLatency(100.0)
				}, ShouldNotPanic)
			})

			Convey("And it should record worker errors", func() {
				So(func() {
					RecordWorkerError()
					RecordWorkerError()
				}, ShouldNotPanic)
			})
		})

		Convey("When recording system metrics", func() {
			Convey("Then it should update system memory usage", func() {
				So(func() {
					UpdateSystemMemoryUsage(1024 * 1024 * 100) // 100MB
					UpdateSystemMemoryUsage(1024 * 1024 * 200) // 200MB
					UpdateSystemMemoryUsage(1024 * 1024 * 300) // 300MB
				}, ShouldNotPanic)
			})

			Convey("And it should update system goroutine count", func() {
				So(func() {
					UpdateSystemGoroutineCount(100)
					UpdateSystemGoroutineCount(200)
					UpdateSystemGoroutineCount(300)
				}, ShouldNotPanic)
			})

			Convey("And it should record system GC pause time", func() {
				So(func() {
					RecordSystemGCPauseTime(1.0)
					RecordSystemGCPauseTime(2.0)
					RecordSystemGCPauseTime(3.0)
				}, ShouldNotPanic)
			})
		})
	})
}

func TestMetricsEdgeCases(t *testing.T) {
	Convey("Given metrics edge cases", t, func() {
		Convey("When recording metrics with edge values", func() {
			Convey("And using zero values", func() {
				So(func() {
					UpdateQueueSize(0)
					UpdateWorkerCount(0)
					UpdateTotalTalents(0)
					RecordScoringLatency(0.0)
					RecordHTTPRequestDuration("/test", "GET", "200", 0.0)
				}, ShouldNotPanic)
			})

			Convey("And using negative values", func() {
				So(func() {
					UpdateQueueSize(-100)
					UpdateWorkerCount(-10)
					UpdateTotalTalents(-1000)
				}, ShouldNotPanic)
			})

			Convey("And using very large values", func() {
				So(func() {
					UpdateQueueSize(1000000)
					UpdateWorkerCount(10000)
					UpdateTotalTalents(10000000)
					RecordScoringLatency(10000.0)
					RecordHTTPRequestDuration("/test", "GET", "200", 30000.0)
				}, ShouldNotPanic)
			})

			Convey("And using empty strings", func() {
				So(func() {
					RecordHTTPRequest("", "", "200")
					RecordHTTPRequestDuration("", "", "200", 10.0)
					RecordErrorByComponent("", "")
					RecordErrorByType("", "")
					RecordErrorByEndpoint("", "", "")
					RecordErrorLatency("", "", 10.0)
				}, ShouldNotPanic)
			})

			Convey("And using special characters in labels", func() {
				So(func() {
					RecordHTTPRequest("/test?param=value&other=123", "GET", "200")
					RecordErrorByComponent("component-with-dash", "error_with_underscore")
					RecordErrorByType("error.with.dots", "error")
					RecordErrorByEndpoint("/api/v1/events", "POST", "timeout")
				}, ShouldNotPanic)
			})
		})
	})
}

func TestMetricsConcurrency(t *testing.T) {
	Convey("Given metrics concurrency", t, func() {
		Convey("When recording metrics concurrently", func() {
			done := make(chan bool, 10)

			// Start multiple goroutines recording metrics
			for i := 0; i < 10; i++ {
				go func(id int) {
					for j := 0; j < 100; j++ {
						RecordEventProcessed()
						UpdateQueueSize(1000 + j)
						RecordScoringLatency(float64(j))
						RecordHTTPRequest("/test", "GET", "200")
					}
					done <- true
				}(i)
			}

			// Wait for all goroutines to complete
			for i := 0; i < 10; i++ {
				<-done
			}

			Convey("Then it should handle concurrent access without panics", func() {
				So(true, ShouldBeTrue) // If we get here, no panics occurred
			})
		})
	})
}

func TestMetricsOptionsValidation(t *testing.T) {
	Convey("Given metrics options validation", t, func() {
		Convey("When creating with empty namespace", func() {
			registry := prometheus.NewRegistry()
			manager := NewMetricsManager(WithNamespace(""), WithPrometheusRegistry(registry))

			Convey("Then it should be created successfully", func() {
				So(manager, ShouldNotBeNil)
			})
		})

		Convey("When creating with empty subsystem", func() {
			registry := prometheus.NewRegistry()
			manager := NewMetricsManager(WithSubsystem(""), WithPrometheusRegistry(registry))

			Convey("Then it should be created successfully", func() {
				So(manager, ShouldNotBeNil)
			})
		})

		Convey("When creating with empty metric prefix", func() {
			registry := prometheus.NewRegistry()
			manager := NewMetricsManager(WithMetricPrefix(""), WithPrometheusRegistry(registry))

			Convey("Then it should be created successfully", func() {
				So(manager, ShouldNotBeNil)
			})
		})

		Convey("When creating with nil histogram buckets", func() {
			registry := prometheus.NewRegistry()
			manager := NewMetricsManager(WithHistogramBuckets(nil), WithPrometheusRegistry(registry))

			Convey("Then it should be created successfully", func() {
				So(manager, ShouldNotBeNil)
			})
		})

		Convey("When creating with empty histogram buckets", func() {
			registry := prometheus.NewRegistry()
			manager := NewMetricsManager(WithHistogramBuckets([]float64{}), WithPrometheusRegistry(registry))

			Convey("Then it should be created successfully", func() {
				So(manager, ShouldNotBeNil)
			})
		})

		Convey("When creating with nil custom labels", func() {
			registry := prometheus.NewRegistry()
			manager := NewMetricsManager(WithCustomLabels(nil), WithPrometheusRegistry(registry))

			Convey("Then it should be created successfully", func() {
				So(manager, ShouldNotBeNil)
			})
		})

		Convey("When creating with empty custom labels", func() {
			registry := prometheus.NewRegistry()
			manager := NewMetricsManager(WithCustomLabels(map[string]string{}), WithPrometheusRegistry(registry))

			Convey("Then it should be created successfully", func() {
				So(manager, ShouldNotBeNil)
			})
		})

		Convey("When creating with zero refresh interval", func() {
			registry := prometheus.NewRegistry()
			manager := NewMetricsManager(WithRefreshInterval(0), WithPrometheusRegistry(registry))

			Convey("Then it should be created successfully", func() {
				So(manager, ShouldNotBeNil)
			})
		})

		Convey("When creating with negative refresh interval", func() {
			registry := prometheus.NewRegistry()
			manager := NewMetricsManager(WithRefreshInterval(-1*time.Second), WithPrometheusRegistry(registry))

			Convey("Then it should be created successfully", func() {
				So(manager, ShouldNotBeNil)
			})
		})
	})
}
