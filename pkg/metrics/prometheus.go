// Package metrics provides Prometheus metrics for the CUJU leaderboard service.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Default metrics configuration constants.
const (
	defaultRefreshInterval = 10 * time.Second
)

// Manager manages all Prometheus metrics for the CUJU service.
type Manager struct {
	namespace        string
	subsystem        string
	histogramBuckets []float64
	enabled          bool
	refreshInterval  time.Duration
	customLabels     map[string]string
	metricPrefix     string
	registry         prometheus.Registerer

	// Core Business Metrics - What really matters for a leaderboard
	eventsProcessed    prometheus.Counter
	eventsDuplicate    prometheus.Counter
	scoringLatency     prometheus.Histogram
	leaderboardUpdates prometheus.Counter

	// Operational Health Metrics
	queueSize    prometheus.Gauge
	workerCount  prometheus.Gauge
	totalTalents prometheus.Gauge

	// Snapshot Metrics - Repository snapshot timings
	repositorySnapshotRebuildDuration prometheus.Histogram
	repositorySnapshotLastUnix        prometheus.Gauge
	repositorySnapshotCount           prometheus.Counter
	repositorySnapshotLastDurationMs  prometheus.Gauge

	// HTTP Performance Metrics
	httpRequests        *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec

	// Business Quality Metrics
	scoringErrors     prometheus.Counter
	leaderboardErrors prometheus.Counter

	// Repository Metrics - Shard and record management
	repositoryShardCount       prometheus.Gauge
	repositoryRecordsTotal     prometheus.Gauge
	repositoryRecordsPerShard  *prometheus.GaugeVec
	repositoryShardUtilization *prometheus.GaugeVec
	repositoryUpdateLatency    prometheus.Histogram
	repositoryQueryLatency     prometheus.Histogram

	// Queue Metrics - Message queue performance
	queueCapacity          prometheus.Gauge
	queueUtilization       prometheus.Gauge
	queueEnqueueRate       prometheus.Counter
	queueDequeueRate       prometheus.Counter
	queueEnqueueErrors     prometheus.Counter
	queueDequeueErrors     prometheus.Counter
	queueProcessingLatency prometheus.Histogram

	// Worker Metrics - Processing performance
	workerActiveCount       prometheus.Gauge
	workerIdleCount         prometheus.Gauge
	workerMessagesPerSecond prometheus.Gauge
	workerProcessingLatency prometheus.Histogram
	workerErrorRate         prometheus.Counter
	workerRetryCount        prometheus.Counter

	// Enhanced Error Metrics - Detailed error tracking
	errorRateByComponent *prometheus.CounterVec
	errorRateByType      *prometheus.CounterVec
	errorRateByEndpoint  *prometheus.CounterVec
	errorLatency         *prometheus.HistogramVec

	// System Performance Metrics
	systemMemoryUsage    prometheus.Gauge
	systemGoroutineCount prometheus.Gauge
	systemGCPauseTime    prometheus.Histogram
}

// Global metrics manager instance.
var globalManager *Manager //nolint:gochecknoglobals // intentional global for singleton metrics manager

// Custom registry to avoid default Go metrics.
var customRegistry = prometheus.NewRegistry() //nolint:gochecknoglobals // intentional global for metrics registry

// Initialize global metrics.
func init() { //nolint:gochecknoinits // intentional init for global metrics setup
	globalManager = NewManager(WithPrometheusRegistry(customRegistry))
}

// NewManager creates a new metrics manager with default configuration.
func NewManager(opts ...Option) *Manager {
	m := &Manager{
		namespace:        "cuju",
		subsystem:        "leaderboard",
		histogramBuckets: prometheus.DefBuckets,
		enabled:          true,
		refreshInterval:  defaultRefreshInterval,
		customLabels:     make(map[string]string),
		metricPrefix:     "",
		registry:         prometheus.DefaultRegisterer,
	}

	// Apply all options
	for _, opt := range opts {
		opt(m)
	}

	// Initialize metrics
	m.initializeMetrics()

	return m
}

// initializeMetrics creates all the Prometheus metrics.
func (m *Manager) initializeMetrics() { //nolint:funlen // long function required for comprehensive metrics initialization
	// Ensure metrics are registered on the configured registry (custom by default)
	auto := promauto.With(m.registry)
	// Core Business Metrics - Focus on what drives business value
	m.eventsProcessed = auto.NewCounter(prometheus.CounterOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "events_processed_total",
		Help:      "Total number of events successfully processed",
	})

	m.eventsDuplicate = auto.NewCounter(prometheus.CounterOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "events_duplicate_total",
		Help:      "Total number of duplicate events detected (indicates data quality)",
	})

	m.scoringLatency = auto.NewHistogram(prometheus.HistogramOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "scoring_latency_milliseconds",
		Help:      "Histogram of scoring latency in milliseconds (core performance metric)",
		Buckets:   m.histogramBuckets,
	})

	m.leaderboardUpdates = auto.NewCounter(prometheus.CounterOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "leaderboard_updates_total",
		Help:      "Total number of leaderboard updates (indicates active competition)",
	})

	// Operational Health Metrics - System stability indicators
	m.queueSize = auto.NewGauge(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "queue_size",
		Help:      "Current size of the event queue (backlog indicator)",
	})

	m.workerCount = auto.NewGauge(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "worker_count",
		Help:      "Current number of active workers (processing capacity)",
	})

	m.totalTalents = auto.NewGauge(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "total_talents",
		Help:      "Total number of talents in the leaderboard (business scale)",
	})

	// HTTP Performance Metrics - User experience indicators
	m.httpRequests = auto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.namespace,
			Subsystem: m.subsystem,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests by endpoint and method",
		},
		[]string{"endpoint", "method", "status_code"},
	)

	m.httpRequestDuration = auto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: m.namespace,
			Subsystem: m.subsystem,
			Name:      "http_request_duration_milliseconds",
			Help:      "HTTP request duration in milliseconds (user experience)",
			Buckets:   m.histogramBuckets,
		},
		[]string{"endpoint", "method", "status_code"},
	)

	// Business Quality Metrics - Error tracking for business impact
	m.scoringErrors = auto.NewCounter(prometheus.CounterOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "scoring_errors_total",
		Help:      "Total number of scoring errors (business impact)",
	})

	m.leaderboardErrors = auto.NewCounter(prometheus.CounterOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "leaderboard_errors_total",
		Help:      "Total number of leaderboard update errors (business impact)",
	})

	// Repository Metrics - Shard and record management
	m.repositoryShardCount = auto.NewGauge(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "repository_shard_count",
		Help:      "Total number of repository shards",
	})

	m.repositoryRecordsTotal = auto.NewGauge(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "repository_records_total",
		Help:      "Total number of records across all shards",
	})

	m.repositoryRecordsPerShard = auto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: m.namespace,
			Subsystem: m.subsystem,
			Name:      "repository_records_per_shard",
			Help:      "Number of records per shard",
		},
		[]string{"shard_id"},
	)

	m.repositoryShardUtilization = auto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: m.namespace,
			Subsystem: m.subsystem,
			Name:      "repository_shard_utilization_ratio",
			Help:      "Shard utilization ratio (records / capacity)",
		},
		[]string{"shard_id"},
	)

	m.repositoryUpdateLatency = auto.NewHistogram(prometheus.HistogramOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "repository_update_latency_milliseconds",
		Help:      "Repository update operation latency in milliseconds",
		Buckets:   m.histogramBuckets,
	})

	m.repositoryQueryLatency = auto.NewHistogram(prometheus.HistogramOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "repository_query_latency_milliseconds",
		Help:      "Repository query operation latency in milliseconds",
		Buckets:   m.histogramBuckets,
	})

	// Snapshot Metrics - Timing and frequency of repository snapshots
	m.repositorySnapshotRebuildDuration = auto.NewHistogram(prometheus.HistogramOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "repository_snapshot_rebuild_duration_milliseconds",
		Help:      "Repository snapshot rebuild duration in milliseconds",
		Buckets:   m.histogramBuckets,
	})

	m.repositorySnapshotLastUnix = auto.NewGauge(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "repository_snapshot_last_unix",
		Help:      "Unix timestamp of the last repository snapshot publish",
	})

	m.repositorySnapshotCount = auto.NewCounter(prometheus.CounterOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "repository_snapshot_count_total",
		Help:      "Total number of repository snapshots published",
	})

	m.repositorySnapshotLastDurationMs = auto.NewGauge(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "repository_snapshot_last_duration_milliseconds",
		Help:      "Last repository snapshot rebuild duration in milliseconds",
	})

	// Queue Metrics - Message queue performance
	m.queueCapacity = auto.NewGauge(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "queue_capacity",
		Help:      "Maximum queue capacity",
	})

	m.queueUtilization = auto.NewGauge(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "queue_utilization_ratio",
		Help:      "Queue utilization ratio (current size / capacity)",
	})

	m.queueEnqueueRate = auto.NewCounter(prometheus.CounterOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "queue_enqueue_total",
		Help:      "Total number of messages enqueued",
	})

	m.queueDequeueRate = auto.NewCounter(prometheus.CounterOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "queue_dequeue_total",
		Help:      "Total number of messages dequeued",
	})

	m.queueEnqueueErrors = auto.NewCounter(prometheus.CounterOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "queue_enqueue_errors_total",
		Help:      "Total number of enqueue errors",
	})

	m.queueDequeueErrors = auto.NewCounter(prometheus.CounterOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "queue_dequeue_errors_total",
		Help:      "Total number of dequeue errors",
	})

	m.queueProcessingLatency = auto.NewHistogram(prometheus.HistogramOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "queue_processing_latency_milliseconds",
		Help:      "Queue processing latency in milliseconds",
		Buckets:   m.histogramBuckets,
	})

	// Worker Metrics - Processing performance
	m.workerActiveCount = auto.NewGauge(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "worker_active_count",
		Help:      "Number of active workers",
	})

	m.workerIdleCount = auto.NewGauge(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "worker_idle_count",
		Help:      "Number of idle workers",
	})

	m.workerMessagesPerSecond = auto.NewGauge(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "worker_messages_per_second",
		Help:      "Average messages processed per second by workers",
	})

	m.workerProcessingLatency = auto.NewHistogram(prometheus.HistogramOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "worker_processing_latency_milliseconds",
		Help:      "Worker processing latency in milliseconds",
		Buckets:   m.histogramBuckets,
	})

	m.workerErrorRate = auto.NewCounter(prometheus.CounterOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "worker_errors_total",
		Help:      "Total number of worker errors",
	})

	m.workerRetryCount = auto.NewCounter(prometheus.CounterOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "worker_retries_total",
		Help:      "Total number of worker retries",
	})

	// Enhanced Error Metrics - Detailed error tracking
	m.errorRateByComponent = auto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.namespace,
			Subsystem: m.subsystem,
			Name:      "errors_by_component_total",
			Help:      "Total number of errors by component",
		},
		[]string{"component", "error_type"},
	)

	m.errorRateByType = auto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.namespace,
			Subsystem: m.subsystem,
			Name:      "errors_by_type_total",
			Help:      "Total number of errors by type",
		},
		[]string{"error_type", "severity"},
	)

	m.errorRateByEndpoint = auto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: m.namespace,
			Subsystem: m.subsystem,
			Name:      "errors_by_endpoint_total",
			Help:      "Total number of errors by endpoint",
		},
		[]string{"endpoint", "method", "error_type"},
	)

	m.errorLatency = auto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: m.namespace,
			Subsystem: m.subsystem,
			Name:      "error_latency_milliseconds",
			Help:      "Latency of operations that resulted in errors",
			Buckets:   m.histogramBuckets,
		},
		[]string{"component", "error_type"},
	)

	// System Performance Metrics
	m.systemMemoryUsage = auto.NewGauge(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "system_memory_usage_bytes",
		Help:      "System memory usage in bytes",
	})

	m.systemGoroutineCount = auto.NewGauge(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "system_goroutine_count",
		Help:      "Number of goroutines",
	})

	m.systemGCPauseTime = auto.NewHistogram(prometheus.HistogramOpts{
		Namespace: m.namespace,
		Subsystem: m.subsystem,
		Name:      "system_gc_pause_time_milliseconds",
		Help:      "GC pause time in milliseconds",
		Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 25, 50, 100, 250, 500, 1000},
	})
}

// RecordEventProcessed increments the events processed counter.
func RecordEventProcessed() {
	globalManager.eventsProcessed.Inc()
}

// RecordEventDuplicate increments the duplicate events counter.
func RecordEventDuplicate() {
	globalManager.eventsDuplicate.Inc()
}

// RecordScoringLatency records scoring latency in milliseconds.
func RecordScoringLatency(latencyMs float64) {
	globalManager.scoringLatency.Observe(latencyMs)
}

// RecordLeaderboardUpdate increments the leaderboard updates counter.
func RecordLeaderboardUpdate() {
	globalManager.leaderboardUpdates.Inc()
}

// UpdateQueueSize sets the current queue size.
func UpdateQueueSize(size int) {
	globalManager.queueSize.Set(float64(size))
}

// UpdateWorkerCount sets the current worker count.
func UpdateWorkerCount(count int) {
	globalManager.workerCount.Set(float64(count))
}

// UpdateTotalTalents sets the total talents count.
func UpdateTotalTalents(count int) {
	globalManager.totalTalents.Set(float64(count))
}

// RecordHTTPRequest records an HTTP request.
func RecordHTTPRequest(endpoint, method, statusCode string) {
	globalManager.httpRequests.WithLabelValues(endpoint, method, statusCode).Inc()
}

// RecordHTTPRequestDuration records HTTP request duration.
func RecordHTTPRequestDuration(endpoint, method, statusCode string, duration float64) {
	globalManager.httpRequestDuration.WithLabelValues(endpoint, method, statusCode).Observe(duration)
}

// RecordScoringError increments the scoring errors counter.
func RecordScoringError() {
	globalManager.scoringErrors.Inc()
}

// RecordLeaderboardError increments the leaderboard errors counter.
func RecordLeaderboardError() {
	globalManager.leaderboardErrors.Inc()
}

// Repository Metrics Functions.

// UpdateRepositoryShardCount sets the total number of repository shards.
func UpdateRepositoryShardCount(count int) {
	globalManager.repositoryShardCount.Set(float64(count))
}

// UpdateRepositoryRecordsTotal sets the total number of records across all shards.
func UpdateRepositoryRecordsTotal(count int) {
	globalManager.repositoryRecordsTotal.Set(float64(count))
}

// UpdateRepositoryRecordsPerShard sets the number of records for a specific shard.
func UpdateRepositoryRecordsPerShard(shardID string, count int) {
	globalManager.repositoryRecordsPerShard.WithLabelValues(shardID).Set(float64(count))
}

// UpdateRepositoryShardUtilization sets the utilization ratio for a specific shard.
func UpdateRepositoryShardUtilization(shardID string, utilization float64) {
	globalManager.repositoryShardUtilization.WithLabelValues(shardID).Set(utilization)
}

// RecordRepositoryUpdateLatency records repository update operation latency.
func RecordRepositoryUpdateLatency(latencyMs float64) {
	globalManager.repositoryUpdateLatency.Observe(latencyMs)
}

// RecordRepositoryQueryLatency records repository query operation latency.
func RecordRepositoryQueryLatency(latencyMs float64) {
	globalManager.repositoryQueryLatency.Observe(latencyMs)
}

// Snapshot Metrics Functions.

// Queue Metrics Functions.

// UpdateQueueCapacity sets the maximum queue capacity.
func UpdateQueueCapacity(capacity int) {
	globalManager.queueCapacity.Set(float64(capacity))
}

// UpdateQueueUtilization sets the queue utilization ratio.
func UpdateQueueUtilization(utilization float64) {
	globalManager.queueUtilization.Set(utilization)
}

// RecordQueueEnqueue increments the enqueue counter.
func RecordQueueEnqueue() {
	globalManager.queueEnqueueRate.Inc()
}

// RecordQueueDequeue increments the dequeue counter.
func RecordQueueDequeue() {
	globalManager.queueDequeueRate.Inc()
}

// RecordQueueEnqueueError increments the enqueue error counter.
func RecordQueueEnqueueError() {
	globalManager.queueEnqueueErrors.Inc()
}

// RecordQueueProcessingLatency records queue processing latency.
func RecordQueueProcessingLatency(latencyMs float64) {
	globalManager.queueProcessingLatency.Observe(latencyMs)
}

// Worker Metrics Functions.

// UpdateWorkerActiveCount sets the number of active workers.
func UpdateWorkerActiveCount(count int) {
	globalManager.workerActiveCount.Set(float64(count))
}

// UpdateWorkerIdleCount sets the number of idle workers.
func UpdateWorkerIdleCount(count int) {
	globalManager.workerIdleCount.Set(float64(count))
}

// UpdateWorkerMessagesPerSecond sets the average messages processed per second.
func UpdateWorkerMessagesPerSecond(rate float64) {
	globalManager.workerMessagesPerSecond.Set(rate)
}

// RecordWorkerProcessingLatency records worker processing latency.
func RecordWorkerProcessingLatency(latencyMs float64) {
	globalManager.workerProcessingLatency.Observe(latencyMs)
}

// RecordWorkerError increments the worker error counter.
func RecordWorkerError() {
	globalManager.workerErrorRate.Inc()
}

// Enhanced Error Metrics Functions.

// RecordErrorByComponent records an error with component and type labels.
func RecordErrorByComponent(component, errorType string) {
	globalManager.errorRateByComponent.WithLabelValues(component, errorType).Inc()
}

// RecordErrorByType records an error with type and severity labels.
func RecordErrorByType(errorType, severity string) {
	globalManager.errorRateByType.WithLabelValues(errorType, severity).Inc()
}

// RecordErrorByEndpoint records an error with endpoint, method, and error type labels.
func RecordErrorByEndpoint(endpoint, method, errorType string) {
	globalManager.errorRateByEndpoint.WithLabelValues(endpoint, method, errorType).Inc()
}

// RecordErrorLatency records the latency of an operation that resulted in an error.
func RecordErrorLatency(component, errorType string, latencyMs float64) {
	globalManager.errorLatency.WithLabelValues(component, errorType).Observe(latencyMs)
}

// System Performance Metrics Functions.

// UpdateSystemMemoryUsage sets the system memory usage in bytes.
func UpdateSystemMemoryUsage(bytes uint64) {
	globalManager.systemMemoryUsage.Set(float64(bytes))
}

// UpdateSystemGoroutineCount sets the number of goroutines.
func UpdateSystemGoroutineCount(count int) {
	globalManager.systemGoroutineCount.Set(float64(count))
}

// RecordSystemGCPauseTime records GC pause time in milliseconds.
func RecordSystemGCPauseTime(pauseMs float64) {
	globalManager.systemGCPauseTime.Observe(pauseMs)
}

// GetRegistry returns the custom Prometheus registry used by our metrics.
func GetRegistry() *prometheus.Registry {
	return customRegistry
}
