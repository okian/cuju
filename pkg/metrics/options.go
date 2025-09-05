// Package metrics provides Prometheus metrics for the CUJU leaderboard service.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Option applies a configuration option to the Manager.
type Option func(*Manager)

// WithNamespace sets the namespace for all metrics.
func WithNamespace(namespace string) Option {
	return func(m *Manager) {
		if namespace != "" {
			m.namespace = namespace
		}
	}
}

// WithSubsystem sets the subsystem for all metrics.
func WithSubsystem(subsystem string) Option {
	return func(m *Manager) {
		if subsystem != "" {
			m.subsystem = subsystem
		}
	}
}

// WithHistogramBuckets sets custom histogram buckets for latency metrics.
func WithHistogramBuckets(buckets []float64) Option {
	return func(m *Manager) {
		if len(buckets) > 0 {
			m.histogramBuckets = buckets
		}
	}
}

// WithMetricsEnabled enables or disables metrics collection.
func WithMetricsEnabled(enabled bool) Option {
	return func(m *Manager) {
		m.enabled = enabled
	}
}

// WithRefreshInterval sets the interval for updating gauge metrics.
func WithRefreshInterval(interval time.Duration) Option {
	return func(m *Manager) {
		if interval > 0 {
			m.refreshInterval = interval
		}
	}
}

// WithCustomLabels adds custom labels to all metrics.
func WithCustomLabels(labels map[string]string) Option {
	return func(m *Manager) {
		if labels != nil {
			m.customLabels = labels
		}
	}
}

// WithMetricPrefix sets a custom prefix for metric names.
func WithMetricPrefix(prefix string) Option {
	return func(m *Manager) {
		if prefix != "" {
			m.metricPrefix = prefix
		}
	}
}

// WithPrometheusRegistry sets a custom Prometheus registry.
func WithPrometheusRegistry(registry prometheus.Registerer) Option {
	return func(m *Manager) {
		if registry != nil {
			m.registry = registry
		}
	}
}
