// Package repository defines the ranking store interface and errors.
package repository

import "time"

// Option applies a configuration option to the TreapStore.
type Option func(*TreapStore)

// WithShardCount sets the number of shards for the leaderboard store.
func WithShardCount(count int) Option {
	return func(s *TreapStore) {
		if count > 0 {
			s.shardCount = count
		}
	}
}

// WithSnapshotInterval sets the interval for publishing snapshots.
func WithSnapshotInterval(interval time.Duration) Option {
	return func(s *TreapStore) {
		if interval > 0 {
			s.snapshotInterval = interval
		}
	}
}

// WithTopCacheSize sets the size of the top-K cache.
func WithTopCacheSize(size int) Option {
	return func(s *TreapStore) {
		if size > 0 {
			s.topCacheSize = size
		}
	}
}
