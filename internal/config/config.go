// Package config defines service configuration structures and loading hooks.
//
// Conventions:
// - Keep fields unexported where possible and use functional options.
// - Provide New(...Option) initializer to build a Config with defaults.
// - All future functions must accept context.Context as the first parameter.
// - External errors must be wrapped via this package's error helpers.
package config

import (
	"context"
	"runtime"
)

// Config contains process configuration. Extend as needed.
type Config struct {
	// LogLevel controls verbosity: debug, info, warn, error.
	LogLevel string `koanf:"log_level"`

	// Addr configures the HTTP listen address, e.g. ":8080".
	Addr string `koanf:"addr"`

	// EventQueueSize bounds the in-memory event queue.
	EventQueueSize int `koanf:"queue_size"`

	// WorkerCount sets the number of scoring workers.
	WorkerCount int `koanf:"worker_count"`

	// DedupeSize sets the size of the deduplication cache.
	DedupeSize int `koanf:"dedupe_size"`

	// ShardCount configures the number of shards in the leaderboard store.
	ShardCount int `koanf:"shard_count"`

	// MaxLeaderboardLimit caps GET /leaderboard?limit.
	MaxLeaderboardLimit int `koanf:"max_leaderboard_limit"`

	// ScoringLatencyMinMS and ScoringLatencyMaxMS simulate external ML latency bounds.
	ScoringLatencyMinMS int `koanf:"scoring_latency_min_ms"`
	ScoringLatencyMaxMS int `koanf:"scoring_latency_max_ms"`

	// SkillWeights maps skill names to their scoring weights.
	SkillWeights map[string]float64 `koanf:"skill_weights"`

	// DefaultSkillWeight is used for unknown skills.
	DefaultSkillWeight float64 `koanf:"default_skill_weight"`
}

// New creates a Config using provided options. Context is accepted first to
// satisfy the project-wide convention; it is reserved for future use (e.g.,
// loading from env/files) and is currently unused.
func New(_ context.Context) *Config {
	c := &Config{
		LogLevel:            "info",
		Addr:                ":9080",
		EventQueueSize:      100_000,
		WorkerCount:         runtime.NumCPU() * 10,
		DedupeSize:          500_000,
		ShardCount:          8,
		MaxLeaderboardLimit: 100,
		ScoringLatencyMinMS: 80,
		ScoringLatencyMaxMS: 150,
		SkillWeights: map[string]float64{
			"dribble": 3.0,
		},
		DefaultSkillWeight: 1.5,
	}
	return c
}
