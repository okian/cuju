// Package repository defines the ranking store interface and errors.
package repository

import "time"

// Option applies a configuration option to the TreapStore.
type Option func(*TreapStore)

// WithMetricsUpdateInterval sets the interval for background metrics updates.
func WithMetricsUpdateInterval(interval time.Duration) Option {
	return func(s *TreapStore) {
		if interval > 0 {
			s.metricsUpdateInterval = interval
		}
	}
}
