// Package scoring defines the contract for computing scores from raw metrics.
package scoring

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// Option applies a configuration option to the InMemoryScorer
type Option func(*InMemoryScorer)

// WithSkillWeight sets a custom weight for a specific skill
func WithSkillWeight(skill string, weight float64) Option {
	return func(s *InMemoryScorer) {
		if weight > 0 {
			s.skillWeights[skill] = weight
		}
	}
}

// WithSkillWeights sets multiple custom weights for skills at once
func WithSkillWeights(weights map[string]float64) Option {
	return func(s *InMemoryScorer) {
		for skill, weight := range weights {
			if weight > 0 {
				s.skillWeights[skill] = weight
			}
		}
	}
}

// WithDefaultWeight sets the default weight for unknown skills
func WithDefaultWeight(weight float64) Option {
	return func(s *InMemoryScorer) {
		if weight > 0 {
			s.defaultWeight = weight
		}
	}
}

// WithLatencyRange sets the simulated latency range
func WithLatencyRange(min, max time.Duration) Option {
	return func(s *InMemoryScorer) {
		if min > 0 && max > min {
			s.minLatency = min
			s.maxLatency = max
		}
	}
}

// WithSkillWeightsFromConfig sets skill weights from a configuration map
func WithSkillWeightsFromConfig(weights map[string]float64, defaultWeight float64) Option {
	return func(s *InMemoryScorer) {
		// Copy the weights map to avoid external modifications
		s.skillWeights = make(map[string]float64)
		for skill, weight := range weights {
			if weight > 0 {
				s.skillWeights[skill] = weight
			}
		}
		if defaultWeight > 0 {
			s.defaultWeight = defaultWeight
		}
	}
}

// Input abstracts the event fields needed for scoring.
type Input struct {
	TalentID  string
	RawMetric float64
	Skill     string
}

// Result contains the computed score for a talent.
type Result struct {
	TalentID string
	Score    float64
}

// Scorer computes a score from an input. The implementation may simulate
// latency to model an external ML service.
type Scorer interface {
	// Score computes a score, honoring ctx for cancellation.
	Score(ctx context.Context, in Input) (Result, error)
}

// InMemoryScorer implements Scorer with simulated ML scoring
type InMemoryScorer struct {
	// Skill-specific scoring parameters
	skillWeights  map[string]float64
	defaultWeight float64
	// Simulated latency range
	minLatency time.Duration
	maxLatency time.Duration
	// Random seed for deterministic scoring
	rng *rand.Rand
}

// NewInMemoryScorer creates a new in-memory scorer with configuration options.
func NewInMemoryScorer(opts ...Option) *InMemoryScorer {
	s := &InMemoryScorer{
		skillWeights:  make(map[string]float64),     // Will be set by options
		defaultWeight: 100,                          // default weight for unknown skills
		minLatency:    80 * time.Millisecond,        // default min latency (docs requirement)
		maxLatency:    150 * time.Millisecond,       // default max latency (docs requirement)
		rng:           rand.New(rand.NewSource(42)), // deterministic for testing
	}

	// Apply all options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Score computes a score for the given input
func (s *InMemoryScorer) Score(ctx context.Context, in Input) (Result, error) {
	// Simulate ML service latency
	latency := s.minLatency + time.Duration(s.rng.Int63n(int64(s.maxLatency-s.minLatency)))
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case <-time.After(latency):
		// Continue with scoring
	}
	// Get skill weight
	weight, ok := s.skillWeights[in.Skill]
	if !ok {
		weight = s.defaultWeight
	}

	// Apply simple weight-based scoring
	score := in.RawMetric * weight

	// Normalize score to 0-100 range
	score = math.Max(0, math.Min(100, score))

	return Result{
		TalentID: in.TalentID,
		Score:    score,
	}, nil
}

// SetSkillWeight allows customization of skill-specific scoring
func (s *InMemoryScorer) SetSkillWeight(skill string, weight float64) {
	s.skillWeights[skill] = weight
}

// SetLatencyRange allows customization of simulated latency
func (s *InMemoryScorer) SetLatencyRange(min, max time.Duration) {
	s.minLatency = min
	s.maxLatency = max
}
