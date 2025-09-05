package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/smartystreets/goconvey/convey"
)

func TestMetricsOptions(t *testing.T) {
	convey.Convey("Given metrics options", t, func() {
		convey.Convey("When creating options", func() {
			namespaceOpt := WithNamespace("test-namespace")
			subsystemOpt := WithSubsystem("test-subsystem")
			metricPrefixOpt := WithMetricPrefix("test-prefix")
			histogramBucketsOpt := WithHistogramBuckets([]float64{0.1, 0.5, 1.0})
			metricsEnabledOpt := WithMetricsEnabled(true)
			refreshIntervalOpt := WithRefreshInterval(5 * time.Second)
			customLabelsOpt := WithCustomLabels(map[string]string{"env": "test"})

			convey.Convey("Then they should be valid functions", func() {
				convey.So(namespaceOpt, convey.ShouldNotBeNil)
				convey.So(subsystemOpt, convey.ShouldNotBeNil)
				convey.So(metricPrefixOpt, convey.ShouldNotBeNil)
				convey.So(histogramBucketsOpt, convey.ShouldNotBeNil)
				convey.So(metricsEnabledOpt, convey.ShouldNotBeNil)
				convey.So(refreshIntervalOpt, convey.ShouldNotBeNil)
				convey.So(customLabelsOpt, convey.ShouldNotBeNil)
			})
		})
	})
}

func TestMetricsManagerCreation(t *testing.T) {
	convey.Convey("Given metrics manager creation", t, func() {
		convey.Convey("When creating with default options", func() {
			registry := prometheus.NewRegistry()
			manager := NewManager(WithPrometheusRegistry(registry))

			convey.Convey("Then it should be created successfully", func() {
				convey.So(manager, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When creating with custom options", func() {
			registry := prometheus.NewRegistry()
			manager := NewManager(
				WithNamespace("test-namespace"),
				WithSubsystem("test-subsystem"),
				WithMetricPrefix("test-prefix"),
				WithHistogramBuckets([]float64{0.1, 0.5, 1.0}),
				WithMetricsEnabled(true),
				WithRefreshInterval(10*time.Second),
				WithCustomLabels(map[string]string{"env": "test", "version": "1.0"}),
				WithPrometheusRegistry(registry),
			)

			convey.Convey("Then it should be created successfully", func() {
				convey.So(manager, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When creating with custom registry", func() {
			registry := prometheus.NewRegistry()
			manager := NewManager(WithPrometheusRegistry(registry))

			convey.Convey("Then it should be created successfully", func() {
				convey.So(manager, convey.ShouldNotBeNil)
			})
		})
	})
}

func TestMetricsRecording(t *testing.T) {
	convey.Convey("Given metrics recording", t, func() {
		convey.Convey("When recording event metrics", func() {
			convey.Convey("Then it should record processed events", func() {
				convey.So(func() {
					RecordEventProcessed()
					RecordEventProcessed()
					RecordEventProcessed()
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record duplicate events", func() {
				convey.So(func() {
					RecordEventDuplicate()
					RecordEventDuplicate()
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record scoring latency", func() {
				convey.So(func() {
					RecordScoringLatency(100.0)
					RecordScoringLatency(150.0)
					RecordScoringLatency(200.0)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record leaderboard updates", func() {
				convey.So(func() {
					RecordLeaderboardUpdate()
					RecordLeaderboardUpdate()
				}, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When recording operational metrics", func() {
			convey.Convey("Then it should update queue size", func() {
				convey.So(func() {
					UpdateQueueSize(1000)
					UpdateQueueSize(2000)
					UpdateQueueSize(500)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should update worker count", func() {
				convey.So(func() {
					UpdateWorkerCount(8)
					UpdateWorkerCount(16)
					UpdateWorkerCount(4)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should update total talents", func() {
				convey.So(func() {
					UpdateTotalTalents(10000)
					UpdateTotalTalents(15000)
					UpdateTotalTalents(20000)
				}, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When recording HTTP metrics", func() {
			convey.Convey("Then it should record HTTP requests", func() {
				convey.So(func() {
					RecordHTTPRequest("/healthz", "GET", "200")
					RecordHTTPRequest("/events", "POST", "202")
					RecordHTTPRequest("/leaderboard", "GET", "200")
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record HTTP request duration", func() {
				convey.So(func() {
					RecordHTTPRequestDuration("/healthz", "GET", "200", 5.0)
					RecordHTTPRequestDuration("/events", "POST", "202", 10.0)
					RecordHTTPRequestDuration("/leaderboard", "GET", "200", 15.0)
				}, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When recording error metrics", func() {
			convey.Convey("Then it should record scoring errors", func() {
				convey.So(func() {
					RecordScoringError()
					RecordScoringError()
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record leaderboard errors", func() {
				convey.So(func() {
					RecordLeaderboardError()
					RecordLeaderboardError()
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record errors by component", func() {
				convey.So(func() {
					RecordErrorByComponent("scoring", "timeout")
					RecordErrorByComponent("repository", "connection_failed")
					RecordErrorByComponent("queue", "full")
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record errors by type", func() {
				convey.So(func() {
					RecordErrorByType("timeout", "error")
					RecordErrorByType("connection_failed", "error")
					RecordErrorByType("validation_error", "warning")
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record errors by endpoint", func() {
				convey.So(func() {
					RecordErrorByEndpoint("/events", "POST", "timeout")
					RecordErrorByEndpoint("/leaderboard", "GET", "not_found")
					RecordErrorByEndpoint("/rank", "GET", "validation_error")
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record error latency", func() {
				convey.So(func() {
					RecordErrorLatency("scoring", "timeout", 100.0)
					RecordErrorLatency("repository", "connection_failed", 200.0)
					RecordErrorLatency("queue", "full", 50.0)
				}, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When recording repository metrics", func() {
			convey.Convey("Then it should update repository shard count", func() {
				convey.So(func() {
					UpdateRepositoryShardCount(4)
					UpdateRepositoryShardCount(8)
					UpdateRepositoryShardCount(16)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should update repository records total", func() {
				convey.So(func() {
					UpdateRepositoryRecordsTotal(100000)
					UpdateRepositoryRecordsTotal(200000)
					UpdateRepositoryRecordsTotal(500000)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should update repository records per shard", func() {
				convey.So(func() {
					UpdateRepositoryRecordsPerShard("shard-0", 25000)
					UpdateRepositoryRecordsPerShard("shard-1", 30000)
					UpdateRepositoryRecordsPerShard("shard-2", 20000)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should update repository shard utilization", func() {
				convey.So(func() {
					UpdateRepositoryShardUtilization("shard-0", 0.75)
					UpdateRepositoryShardUtilization("shard-1", 0.85)
					UpdateRepositoryShardUtilization("shard-2", 0.65)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record repository update latency", func() {
				convey.So(func() {
					RecordRepositoryUpdateLatency(5.0)
					RecordRepositoryUpdateLatency(10.0)
					RecordRepositoryUpdateLatency(15.0)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record repository query latency", func() {
				convey.So(func() {
					RecordRepositoryQueryLatency(2.0)
					RecordRepositoryQueryLatency(5.0)
					RecordRepositoryQueryLatency(8.0)
				}, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When recording queue metrics", func() {
			convey.Convey("Then it should update queue capacity", func() {
				convey.So(func() {
					UpdateQueueCapacity(10000)
					UpdateQueueCapacity(20000)
					UpdateQueueCapacity(50000)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should update queue utilization", func() {
				convey.So(func() {
					UpdateQueueUtilization(0.5)
					UpdateQueueUtilization(0.75)
					UpdateQueueUtilization(0.9)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record queue enqueue", func() {
				convey.So(func() {
					RecordQueueEnqueue()
					RecordQueueEnqueue()
					RecordQueueEnqueue()
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record queue dequeue", func() {
				convey.So(func() {
					RecordQueueDequeue()
					RecordQueueDequeue()
					RecordQueueDequeue()
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record queue enqueue errors", func() {
				convey.So(func() {
					RecordQueueEnqueueError()
					RecordQueueEnqueueError()
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record queue processing latency", func() {
				convey.So(func() {
					RecordQueueProcessingLatency(20.0)
					RecordQueueProcessingLatency(30.0)
					RecordQueueProcessingLatency(40.0)
				}, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When recording worker metrics", func() {
			convey.Convey("Then it should update worker active count", func() {
				convey.So(func() {
					UpdateWorkerActiveCount(4)
					UpdateWorkerActiveCount(8)
					UpdateWorkerActiveCount(12)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should update worker idle count", func() {
				convey.So(func() {
					UpdateWorkerIdleCount(2)
					UpdateWorkerIdleCount(4)
					UpdateWorkerIdleCount(6)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should update worker messages per second", func() {
				convey.So(func() {
					UpdateWorkerMessagesPerSecond(100.0)
					UpdateWorkerMessagesPerSecond(200.0)
					UpdateWorkerMessagesPerSecond(300.0)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record worker processing latency", func() {
				convey.So(func() {
					RecordWorkerProcessingLatency(50.0)
					RecordWorkerProcessingLatency(75.0)
					RecordWorkerProcessingLatency(100.0)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record worker errors", func() {
				convey.So(func() {
					RecordWorkerError()
					RecordWorkerError()
				}, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When recording system metrics", func() {
			convey.Convey("Then it should update system memory usage", func() {
				convey.So(func() {
					UpdateSystemMemoryUsage(1024 * 1024 * 100) // 100MB
					UpdateSystemMemoryUsage(1024 * 1024 * 200) // 200MB
					UpdateSystemMemoryUsage(1024 * 1024 * 300) // 300MB
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should update system goroutine count", func() {
				convey.So(func() {
					UpdateSystemGoroutineCount(100)
					UpdateSystemGoroutineCount(200)
					UpdateSystemGoroutineCount(300)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should record system GC pause time", func() {
				convey.So(func() {
					RecordSystemGCPauseTime(1.0)
					RecordSystemGCPauseTime(2.0)
					RecordSystemGCPauseTime(3.0)
				}, convey.ShouldNotPanic)
			})
		})
	})
}

func TestMetricsEdgeCases(t *testing.T) {
	convey.Convey("Given metrics edge cases", t, func() {
		convey.Convey("When recording metrics with edge values", func() {
			convey.Convey("And using zero values", func() {
				convey.So(func() {
					UpdateQueueSize(0)
					UpdateWorkerCount(0)
					UpdateTotalTalents(0)
					RecordScoringLatency(0.0)
					RecordHTTPRequestDuration("/test", "GET", "200", 0.0)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And using negative values", func() {
				convey.So(func() {
					UpdateQueueSize(-100)
					UpdateWorkerCount(-10)
					UpdateTotalTalents(-1000)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And using very large values", func() {
				convey.So(func() {
					UpdateQueueSize(1000000)
					UpdateWorkerCount(10000)
					UpdateTotalTalents(10000000)
					RecordScoringLatency(10000.0)
					RecordHTTPRequestDuration("/test", "GET", "200", 30000.0)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And using empty strings", func() {
				convey.So(func() {
					RecordHTTPRequest("", "", "200")
					RecordHTTPRequestDuration("", "", "200", 10.0)
					RecordErrorByComponent("", "")
					RecordErrorByType("", "")
					RecordErrorByEndpoint("", "", "")
					RecordErrorLatency("", "", 10.0)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And using special characters in labels", func() {
				convey.So(func() {
					RecordHTTPRequest("/test?param=value&other=123", "GET", "200")
					RecordErrorByComponent("component-with-dash", "error_with_underscore")
					RecordErrorByType("error.with.dots", "error")
					RecordErrorByEndpoint("/api/v1/events", "POST", "timeout")
				}, convey.ShouldNotPanic)
			})
		})
	})
}

func TestMetricsConcurrency(t *testing.T) {
	convey.Convey("Given metrics concurrency", t, func() {
		convey.Convey("When recording metrics concurrently", func() {
			done := make(chan bool, 10)

			// Start multiple goroutines recording metrics
			for i := 0; i < 10; i++ {
				go func(_ int) {
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

			convey.Convey("Then it should handle concurrent access without panics", func() {
				convey.So(true, convey.ShouldBeTrue) // If we get here, no panics occurred
			})
		})
	})
}

func TestMetricsOptionsValidation(t *testing.T) {
	convey.Convey("Given metrics options validation", t, func() {
		convey.Convey("When creating with empty namespace", func() {
			registry := prometheus.NewRegistry()
			manager := NewManager(WithNamespace(""), WithPrometheusRegistry(registry))

			convey.Convey("Then it should be created successfully", func() {
				convey.So(manager, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When creating with empty subsystem", func() {
			registry := prometheus.NewRegistry()
			manager := NewManager(WithSubsystem(""), WithPrometheusRegistry(registry))

			convey.Convey("Then it should be created successfully", func() {
				convey.So(manager, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When creating with empty metric prefix", func() {
			registry := prometheus.NewRegistry()
			manager := NewManager(WithMetricPrefix(""), WithPrometheusRegistry(registry))

			convey.Convey("Then it should be created successfully", func() {
				convey.So(manager, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When creating with nil histogram buckets", func() {
			registry := prometheus.NewRegistry()
			manager := NewManager(WithHistogramBuckets(nil), WithPrometheusRegistry(registry))

			convey.Convey("Then it should be created successfully", func() {
				convey.So(manager, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When creating with empty histogram buckets", func() {
			registry := prometheus.NewRegistry()
			manager := NewManager(WithHistogramBuckets([]float64{}), WithPrometheusRegistry(registry))

			convey.Convey("Then it should be created successfully", func() {
				convey.So(manager, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When creating with nil custom labels", func() {
			registry := prometheus.NewRegistry()
			manager := NewManager(WithCustomLabels(nil), WithPrometheusRegistry(registry))

			convey.Convey("Then it should be created successfully", func() {
				convey.So(manager, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When creating with empty custom labels", func() {
			registry := prometheus.NewRegistry()
			manager := NewManager(WithCustomLabels(map[string]string{}), WithPrometheusRegistry(registry))

			convey.Convey("Then it should be created successfully", func() {
				convey.So(manager, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When creating with zero refresh interval", func() {
			registry := prometheus.NewRegistry()
			manager := NewManager(WithRefreshInterval(0), WithPrometheusRegistry(registry))

			convey.Convey("Then it should be created successfully", func() {
				convey.So(manager, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When creating with negative refresh interval", func() {
			registry := prometheus.NewRegistry()
			manager := NewManager(WithRefreshInterval(-1*time.Second), WithPrometheusRegistry(registry))

			convey.Convey("Then it should be created successfully", func() {
				convey.So(manager, convey.ShouldNotBeNil)
			})
		})
	})
}
