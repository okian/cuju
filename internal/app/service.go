// Package service provides the core business service that implements
// the dependencies required by the HTTP API.
package service

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"time"

	eventqueue "github.com/okian/cuju/internal/adapters/mq/queue"
	workerpool "github.com/okian/cuju/internal/adapters/mq/worker"
	repository "github.com/okian/cuju/internal/adapters/repository"
	"github.com/okian/cuju/internal/domain/dedupe"
	"github.com/okian/cuju/internal/domain/model"
	"github.com/okian/cuju/internal/domain/scoring"
	"github.com/okian/cuju/internal/domain/types"
	"github.com/okian/cuju/pkg/logger"
	"github.com/okian/cuju/pkg/metrics"
)

// scoringAdapter adapts the scoring.Scorer interface to worker.Scorer
type scoringAdapter struct {
	scorer scoring.Scorer
}

func (a *scoringAdapter) Score(ctx context.Context, talentID string, rawMetric float64, skill string) (float64, error) {
	input := scoring.Input{
		TalentID:  talentID,
		RawMetric: rawMetric,
		Skill:     skill,
	}

	result, err := a.scorer.Score(ctx, input)
	if err != nil {
		return 0, err
	}

	return result.Score, nil
}

// Service implements the API dependencies for the leaderboard system.
type Service struct {
	mu sync.RWMutex

	// Core components
	leaderboard repository.Store
	deduper     dedupe.Deduper
	eventQueue  eventqueue.Queue
	scorer      scoring.Scorer
	workerPool  *workerpool.WorkerPool

	// Configuration
	workerCount   int
	queueSize     int
	dedupeSize    int
	skillWeights  map[string]float64
	defaultWeight float64
	// Scoring latency configuration
	scoringMinLatency time.Duration
	scoringMaxLatency time.Duration

	// State
	started bool
	stopCh  chan struct{}

	// Logging
	logger logger.Logger
}

// Option applies a configuration option to the Service.
type Option func(*Service)

// WithWorkerCount sets the number of worker goroutines.
func WithWorkerCount(count int) Option {
	return func(s *Service) {
		if count > 0 {
			s.workerCount = count
		}
	}
}

// WithQueueSize sets the maximum size of the event queue.
func WithQueueSize(size int) Option {
	return func(s *Service) {
		if size > 0 {
			s.queueSize = size
		}
	}
}

// WithDedupeSize sets the size of the deduplication cache.
func WithDedupeSize(size int) Option {
	return func(s *Service) {
		if size > 0 {
			s.dedupeSize = size
		}
	}
}

// WithLogger sets a custom logger for the service.
func WithLogger(logger logger.Logger) Option {
	return func(s *Service) {
		if logger != nil {
			s.logger = logger
		}
	}
}

// WithSkillWeights sets the skill weights for scoring.
func WithSkillWeights(weights map[string]float64) Option {
	return func(s *Service) {
		s.skillWeights = weights
	}
}

// WithDefaultSkillWeight sets the default weight for unknown skills.
func WithDefaultSkillWeight(weight float64) Option {
	return func(s *Service) {
		s.defaultWeight = weight
	}
}

// WithScoringLatencyRange sets the simulated scoring latency range.
func WithScoringLatencyRange(min, max time.Duration) Option {
	return func(s *Service) {
		if min > 0 && max > min {
			s.scoringMinLatency = min
			s.scoringMaxLatency = max
		}
	}
}

// New constructs a new Service with default configuration.
func New(opts ...Option) *Service {
	s := &Service{
		workerCount: runtime.NumCPU() * 2, // Default to 2x CPU cores
		queueSize:   100000,               // Default queue size
		dedupeSize:  50000,                // Default dedupe cache size
		skillWeights: map[string]float64{
			"coding": 1.0,
		},
		defaultWeight:     0.5,
		stopCh:            make(chan struct{}),
		logger:            nil, // Will be replaced when service starts
		scoringMinLatency: 80 * time.Millisecond,
		scoringMaxLatency: 150 * time.Millisecond,
	}

	// Apply all options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Start initializes and starts the service components.
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}

	// Initialize logger if not already set
	if s.logger == nil {
		s.logger = logger.Get()
	}

	s.logger.Info(ctx, "starting leaderboard service...")

	// Initialize components
	s.leaderboard = repository.NewTreapStore(ctx)
	s.logger.Info(ctx, "using treap store")
	s.deduper = dedupe.NewInMemoryDeduper(
		dedupe.WithMaxSize(s.dedupeSize),
	)
	s.eventQueue = eventqueue.NewInMemoryQueue(
		eventqueue.WithCapacity(s.queueSize),
		eventqueue.WithBufferSize(s.queueSize),
	)
	s.scorer = scoring.NewInMemoryScorer(
		scoring.WithSkillWeightsFromConfig(s.skillWeights, s.defaultWeight),
		scoring.WithLatencyRange(s.scoringMinLatency, s.scoringMaxLatency),
	)

	// Create and start worker pool with scoring adapter
	scoringAdapter := &scoringAdapter{scorer: s.scorer}
	s.workerPool = workerpool.NewWorkerPool(s.workerCount, s.eventQueue, scoringAdapter, s.leaderboard)
	s.workerPool.Start(ctx)

	s.started = true
	s.logger.Info(ctx, "leaderboard service started",
		logger.Int("workers", s.workerCount),
		logger.Int("queueSize", s.queueSize),
		logger.Int("dedupeSize", s.dedupeSize),
	)

	return nil
}

// Stop gracefully shuts down the service.
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return
	}

	s.logger.Info(context.Background(), "stopping leaderboard service...")

	// Stop worker pool
	if s.workerPool != nil {
		s.workerPool.Stop()
	}

	// Close leaderboard store
	if s.leaderboard != nil {
		if closer, ok := s.leaderboard.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}

	// Close queue
	if q, ok := s.eventQueue.(*eventqueue.InMemoryQueue); ok {
		_ = q.Close()
	}

	// Signal cleanup loop to stop
	select {
	case <-s.stopCh:
		// Channel already closed
	default:
		close(s.stopCh)
	}

	s.started = false
	s.logger.Info(context.Background(), "leaderboard service stopped")
}

// SeenAndRecord atomically checks if an event id was seen and records it if not.
// Returns true if the event was already seen, false if it was newly recorded.
// This is the ONLY method for deduplication - thread-safe and atomic.
func (s *Service) SeenAndRecord(ctx context.Context, id string) bool {
	seen := s.deduper.SeenAndRecord(ctx, id)
	if seen {
		metrics.RecordEventDuplicate()
	}
	return seen
}

// Unrecord removes an event ID from the seen list, allowing it to be retried.
func (s *Service) Unrecord(ctx context.Context, id string) {
	s.deduper.Unrecord(ctx, id)
}

// Enqueue submits an event for asynchronous processing.
func (s *Service) Enqueue(ctx context.Context, e any) bool {
	// Log what we receive
	s.logger.Debug(ctx, "received event",
		logger.String("type", reflect.TypeOf(e).String()),
		logger.Any("event", e),
	)

	// Try to extract fields using reflection
	v := reflect.ValueOf(e)
	if v.Kind() == reflect.Struct {
		talentID := v.FieldByName("TalentID").String()
		rawMetric := v.FieldByName("RawMetric").Float()
		skill := v.FieldByName("Skill").String()

		s.logger.Debug(ctx, "extracted event fields",
			logger.String("talentID", talentID),
			logger.Float64("rawMetric", rawMetric),
			logger.String("skill", skill),
		)

		if talentID != "" && skill != "" {
			// Try to get EventID from the original event, or generate a deterministic one
			eventID := v.FieldByName("EventID").String()
			if eventID == "" {
				// Generate a deterministic event ID based on content for deduplication
				eventID = fmt.Sprintf("%s_%s_%f", talentID, skill, rawMetric)
			}

			// Check for duplicates before enqueueing
			if s.SeenAndRecord(ctx, eventID) {
				s.logger.Debug(ctx, "duplicate event detected, skipping",
					logger.String("eventID", eventID),
					logger.String("talentID", talentID),
				)
				return true // Return true to indicate "processed" (as duplicate)
			}

			workerEvent := model.Event{
				EventID:   eventID,
				TalentID:  talentID,
				RawMetric: rawMetric,
				Skill:     skill,
			}
			s.logger.Debug(ctx, "enqueueing worker event",
				logger.String("eventID", workerEvent.EventID),
				logger.String("talentID", workerEvent.TalentID),
				logger.Float64("rawMetric", workerEvent.RawMetric),
				logger.String("skill", workerEvent.Skill),
			)
			success := s.eventQueue.Enqueue(ctx, workerEvent)
			if success {
				metrics.RecordEventProcessed()
				// Update queue size metric
				metrics.UpdateQueueSize(s.eventQueue.Len(ctx))
			}
			return success
		}
	}

	s.logger.Warn(ctx, "failed to convert event type",
		logger.String("type", reflect.TypeOf(e).String()),
	)
	return false
}

// TopN returns the top N leaderboard entries.
func (s *Service) TopN(ctx context.Context, n int) ([]types.Entry, error) {
	entries, err := s.leaderboard.TopN(ctx, n)
	if err != nil {
		return nil, err
	}

	// Convert to API format
	apiEntries := make([]types.Entry, len(entries))
	for i, entry := range entries {
		apiEntries[i] = types.Entry{
			Rank:     entry.Rank,
			TalentID: entry.TalentID,
			Score:    entry.Score,
		}
	}

	return apiEntries, nil
}

// Rank returns the rank and score for a given talent id.
func (s *Service) Rank(ctx context.Context, talentID string) (types.Entry, error) {
	entry, err := s.leaderboard.Rank(ctx, talentID)
	if err != nil {
		return types.Entry{}, err
	}

	return types.Entry{
		Rank:     entry.Rank,
		TalentID: entry.TalentID,
		Score:    entry.Score,
	}, nil
}

// GetStats returns service statistics for monitoring.
func (s *Service) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := context.Background()
	stats := map[string]interface{}{
		"started":     s.started,
		"workerCount": s.workerCount,
		"queueSize":   s.queueSize,
		"dedupeSize":  s.dedupeSize,
	}

	if s.started {
		queueLen := s.eventQueue.Len(ctx)
		totalTalents := s.leaderboard.Count(ctx)

		stats["queueLength"] = queueLen
		stats["totalTalents"] = totalTalents

		// Update metrics
		metrics.UpdateQueueSize(queueLen)
		metrics.UpdateTotalTalents(totalTalents)
		metrics.UpdateWorkerCount(s.workerCount)
	}

	return stats
}

// Size returns the current number of entries in the deduper.
func (s *Service) Size() int64 {
	if s.deduper == nil {
		return 0
	}
	return s.deduper.Size()
}
