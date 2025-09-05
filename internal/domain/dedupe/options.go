// Package dedupe defines the interface for idempotency tracking.
package dedupe

// Option applies a configuration option to the InMemoryDeduper.
type Option func(*inMemoryDeduper)

// WithMaxSize sets the maximum number of IDs to keep in memory.
// If maxSize > 0: bounded mode with LIFO eviction.
// If maxSize <= 0: unbounded mode (no eviction, no size limit).
func WithMaxSize(maxSize int) Option {
	return func(d *inMemoryDeduper) {
		d.maxSize = maxSize
	}
}
