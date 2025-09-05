// Package worker defines worker contracts for asynchronous scoring and updates.
package worker

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"time"

	"github.com/okian/cuju/internal/adapters/mq/queue"
	"github.com/okian/cuju/internal/domain/model"
	"github.com/okian/cuju/pkg/logger"
	"github.com/okian/cuju/pkg/metrics"
)

// Default worker configuration constants.
const (
	defaultWorkerMultiplier = 20 // multiplier for runtime.NumCPU()
	metricsUpdateInterval   = 5 * time.Second
	workerShutdownTimeout   = 5 * time.Second
	poolShutdownTimeout     = 30 * time.Second
)

// Event abstracts what workers read off the queue.
// Using the model.Event type for consistency.
type Event = model.Event

// Updater updates the best score for a talent.
type Updater interface {
	UpdateBest(ctx context.Context, talentID string, score float64) (bool, error)
	// Optional extended method for metadata-aware repositories
	UpdateBestWithMeta(ctx context.Context, talentID string, score float64, eventID string, skill string, rawMetric float64) (bool, error)
}

// Scorer computes a score for an event.
type Scorer interface {
	Score(ctx context.Context, talentID string, rawMetric float64, skill string) (float64, error)
}

// Queue defines how workers receive events.
type Queue interface {
	Dequeue(ctx context.Context) <-chan Event
}

// Worker processes events and writes score updates using the provided interfaces.
type Worker interface {
	// Run starts the worker loop until ctx is canceled.
	Run(ctx context.Context)

	// Shutdown gracefully stops the worker.
	// It will process any remaining events before stopping.
	Shutdown(ctx context.Context) error
}

// InMemoryWorker implements Worker for processing events.
type InMemoryWorker struct {
	queue   Queue
	scorer  Scorer
	updater Updater
	name    string

	// Configuration
	// (removed unused timeout, retryCount, backoffStrategy, backoffDelay, metricsEnabled fields)

	// Shutdown control
	shutdown chan struct{}
	done     chan struct{}

	// Logging
	logger logger.Logger
}

// NewInMemoryWorker creates a new worker with configuration options.
func NewInMemoryWorker(queue Queue, scorer Scorer, updater Updater, opts ...Option) *InMemoryWorker {
	w := &InMemoryWorker{
		queue:    queue,
		scorer:   scorer,
		updater:  updater,
		name:     "worker", // default name
		shutdown: make(chan struct{}),
		done:     make(chan struct{}),
		logger:   logger.Get().Named("worker"), // will be updated by options
	}

	// Apply all options
	for _, opt := range opts {
		opt(w)
	}

	// Set up logger with worker name if not already set
	if w.name != "worker" {
		w.logger = w.logger.Named(w.name)
	}

	return w
}

// Run starts the worker loop.
func (w *InMemoryWorker) Run(ctx context.Context) {
	defer func() {
		close(w.done)
	}()

	eventChan := w.queue.Dequeue(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.shutdown:
			return
		case event, ok := <-eventChan:
			if !ok {
				// Channel closed, worker should stop
				return
			}

			// Process the event
			if err := w.processEvent(ctx, event); err != nil {
				w.logger.Error(ctx, "error processing event", logger.Error(err))
			}
		}
	}
}

// Shutdown gracefully stops the worker.
func (w *InMemoryWorker) Shutdown(ctx context.Context) error {
	// Signal shutdown
	close(w.shutdown)

	// Wait for worker to finish or context to timeout
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		w.logger.Warn(ctx, "shutdown timed out")
		return fmt.Errorf("shutdown timed out: %w", ctx.Err())
	}
}

// processEvent handles a single event.
func (w *InMemoryWorker) processEvent(ctx context.Context, event queue.Event) error { //nolint:gocritic // hugeParam: Event must be passed by value for channel semantics
	// Track overall processing latency
	start := time.Now()
	defer func() {
		latency := time.Since(start).Milliseconds()
		metrics.RecordWorkerProcessingLatency(float64(latency))
	}()

	// Track scoring latency
	scoreStart := time.Now()
	score, err := w.scorer.Score(ctx, event.TalentID, event.RawMetric, event.Skill)
	scoreLatency := time.Since(scoreStart).Milliseconds()

	// Record scoring latency metric
	metrics.RecordScoringLatency(float64(scoreLatency))

	if err != nil {
		// Record scoring error metric
		metrics.RecordScoringError()
		metrics.RecordWorkerError()
		metrics.RecordErrorByComponent("worker", "scoring_error")
		metrics.RecordErrorByType("scoring_error", "high")
		w.logger.Error(ctx, "scoring failed for event",
			logger.String("eventID", event.EventID),
			logger.Error(err),
		)
		return fmt.Errorf("failed to score event %s: %w", event.EventID, err)
	}

	// Update the leaderboard
	var updated bool
	if extended, ok := any(w.updater).(interface {
		UpdateBestWithMeta(ctx context.Context, talentID string, score float64, eventID string, skill string, rawMetric float64) (bool, error)
	}); ok {
		updated, err = extended.UpdateBestWithMeta(ctx, event.TalentID, score, event.EventID, event.Skill, event.RawMetric)
	} else {
		updated, err = w.updater.UpdateBest(ctx, event.TalentID, score)
	}
	if err != nil {
		// Record leaderboard error metric
		metrics.RecordLeaderboardError()
		metrics.RecordWorkerError()
		metrics.RecordErrorByComponent("worker", "leaderboard_error")
		metrics.RecordErrorByType("leaderboard_error", "high")
		w.logger.Error(ctx, "leaderboard update failed for event",
			logger.String("eventID", event.EventID),
			logger.Error(err),
		)
		return fmt.Errorf("leaderboard update failed: %w", err)
	}

	if updated {
		// Record leaderboard update metric
		metrics.RecordLeaderboardUpdate()
		metrics.RecordEventProcessed()
	}

	return nil
}

// Pool manages multiple workers.
type Pool struct {
	workers []*InMemoryWorker
	queue   Queue
	scorer  Scorer
	updater Updater

	// Shutdown control
	shutdown chan struct{}
	done     chan struct{}

	// Metrics tracking
	processedCount    int64
	lastProcessedTime time.Time

	// Logging
	logger logger.Logger
}

// NewPool creates a new worker pool.
func NewPool(workerCount int, queue Queue, scorer Scorer, updater Updater) *Pool {
	if workerCount < 1 {
		workerCount = runtime.NumCPU() * defaultWorkerMultiplier
	}

	pool := &Pool{
		workers:           make([]*InMemoryWorker, workerCount),
		queue:             queue,
		scorer:            scorer,
		updater:           updater,
		shutdown:          make(chan struct{}),
		done:              make(chan struct{}),
		lastProcessedTime: time.Now(),
		logger:            logger.Get().Named("worker-pool"),
	}

	for i := 0; i < workerCount; i++ {
		pool.workers[i] = NewInMemoryWorker(
			queue,
			scorer,
			updater,
			WithName("worker-"+strconv.Itoa(i)),
		)
	}

	// Initialize worker metrics
	metrics.UpdateWorkerActiveCount(workerCount)
	metrics.UpdateWorkerIdleCount(0)
	metrics.UpdateWorkerMessagesPerSecond(0.0)

	return pool
}

// Start starts all workers in the pool.
func (p *Pool) Start(ctx context.Context) {
	for _, worker := range p.workers {
		go worker.Run(ctx)
	}

	// Start metrics updater
	go p.startMetricsUpdater(ctx)
}

// startMetricsUpdater starts a background goroutine that updates worker metrics.
func (p *Pool) startMetricsUpdater(ctx context.Context) {
	ticker := time.NewTicker(metricsUpdateInterval) // Update metrics every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.shutdown:
			return
		case <-ticker.C:
			p.updateMetrics()
		}
	}
}

// updateMetrics updates worker-related metrics.
func (p *Pool) updateMetrics() {
	// Calculate messages per second
	now := time.Now()
	timeDiff := now.Sub(p.lastProcessedTime).Seconds()
	if timeDiff > 0 {
		messagesPerSecond := float64(p.processedCount) / timeDiff
		metrics.UpdateWorkerMessagesPerSecond(messagesPerSecond)
	}

	// Reset counters
	p.processedCount = 0
	p.lastProcessedTime = now
}

// RecordProcessedMessage increments the processed message count.
func (p *Pool) RecordProcessedMessage() {
	p.processedCount++
}

// Stop gracefully stops all workers.
func (p *Pool) Stop() {
	// Signal shutdown to all workers
	close(p.shutdown)

	// Wait for all workers to finish
	for _, worker := range p.workers {
		select {
		case <-worker.done:
			// Worker finished
		case <-time.After(workerShutdownTimeout):
			// Worker timeout
		}
	}
}

// Shutdown gracefully shuts down the entire worker pool.
func (p *Pool) Shutdown(ctx context.Context) error {
	// First close the queue to stop new events
	if closer, ok := p.queue.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			p.logger.Error(ctx, "error closing queue", logger.Error(err))
		}
	}

	// Signal shutdown to all workers
	close(p.shutdown)

	// Wait for all workers to finish or context to timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, poolShutdownTimeout)
	defer cancel()

	for i, worker := range p.workers {
		select {
		case <-worker.done:
		case <-shutdownCtx.Done():
			p.logger.Warn(ctx, "worker shutdown timed out", logger.Int("worker_id", i))
		}
	}

	return nil
}
