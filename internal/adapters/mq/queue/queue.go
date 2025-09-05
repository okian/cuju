// Package queue defines the contract for enqueuing and consuming events.
//
// Implementations may use channels or more advanced structures. The MVP
// will start with an in-memory bounded queue.
package queue

import (
	"context"
	"sync"
	"time"

	"github.com/okian/cuju/internal/domain/model"
	"github.com/okian/cuju/pkg/metrics"
)

// Default queue configuration constants.
const (
	defaultQueueCapacity = 100000
	defaultBufferSize    = 100000
)

// Event represents the payload type flowing through the queue.
// Using the model.Event type for type safety.
type Event = model.Event

// Queue provides non-blocking enqueue and channel-based dequeue semantics.
type Queue interface {
	// Enqueue adds an event to the queue.
	// Returns false if the queue is full and the event was not enqueued.
	Enqueue(ctx context.Context, e Event) bool

	// Dequeue returns a channel that will receive events as they become available.
	// The channel will be closed when the queue is closed.
	Dequeue(ctx context.Context) <-chan Event

	// Len returns the current number of queued events.
	Len(ctx context.Context) int

	// Close gracefully shuts down the queue.
	// After closing, no new events can be enqueued and the dequeue channel will be closed.
	Close() error

	// IsClosed returns true if the queue has been closed.
	IsClosed() bool
}

// InMemoryQueue implements Queue using a buffered channel.
type InMemoryQueue struct {
	events     chan Event
	capacity   int
	bufferSize int
	// (removed unused enqueueTimeout field)
	mu     sync.RWMutex
	closed bool
}

// NewInMemoryQueue creates a new in-memory queue with configuration options.
func NewInMemoryQueue(opts ...Option) *InMemoryQueue {
	q := &InMemoryQueue{
		capacity:   defaultQueueCapacity, // default capacity
		bufferSize: defaultBufferSize,    // default buffer size
	}

	// Apply all options
	for _, opt := range opts {
		opt(q)
	}

	// Initialize the events channel with the configured buffer size
	q.events = make(chan Event, q.bufferSize)

	// Initialize metrics
	metrics.UpdateQueueCapacity(q.capacity)
	metrics.UpdateQueueSize(0)
	metrics.UpdateQueueUtilization(0.0)

	return q
}

// Enqueue adds an event to the queue.
func (q *InMemoryQueue) Enqueue(ctx context.Context, e Event) bool { //nolint:gocritic // hugeParam: Event must be passed by value for channel semantics
	start := time.Now()
	defer func() {
		latency := time.Since(start).Milliseconds()
		metrics.RecordQueueProcessingLatency(float64(latency))
	}()

	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		metrics.RecordQueueEnqueueError()
		metrics.RecordErrorByComponent("queue", "closed")
		return false
	}

	// Check if we're at capacity
	if len(q.events) >= q.capacity {
		metrics.RecordQueueEnqueueError()
		metrics.RecordErrorByComponent("queue", "capacity_exceeded")
		return false
	}

	select {
	case q.events <- e:
		metrics.RecordQueueEnqueue()
		// Update queue size and utilization
		currentSize := len(q.events)
		metrics.UpdateQueueSize(currentSize)
		utilization := float64(currentSize) / float64(q.capacity)
		metrics.UpdateQueueUtilization(utilization)
		return true
	case <-ctx.Done():
		metrics.RecordQueueEnqueueError()
		metrics.RecordErrorByComponent("queue", "context_cancelled")
		return false // context cancelled
	default:
		metrics.RecordQueueEnqueueError()
		metrics.RecordErrorByComponent("queue", "queue_full")
		return false // queue is full
	}
}

// Dequeue returns a channel that will receive events as they become available.
func (q *InMemoryQueue) Dequeue(ctx context.Context) <-chan Event {
	// Wrap the channel to track dequeue metrics
	dequeueChan := make(chan Event)
	go func() {
		defer close(dequeueChan)
		for event := range q.events {
			select {
			case dequeueChan <- event:
				metrics.RecordQueueDequeue()
				// Update queue size and utilization after dequeue
				currentSize := len(q.events)
				metrics.UpdateQueueSize(currentSize)
				utilization := float64(currentSize) / float64(q.capacity)
				metrics.UpdateQueueUtilization(utilization)
			case <-ctx.Done():
				return
			}
		}
	}()
	return dequeueChan
}

// Len returns the current number of queued events.
func (q *InMemoryQueue) Len(ctx context.Context) int {
	size := len(q.events)
	metrics.UpdateQueueSize(size)
	utilization := float64(size) / float64(q.capacity)
	metrics.UpdateQueueUtilization(utilization)
	return size
}

// Close gracefully shuts down the queue.
func (q *InMemoryQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil // already closed
	}

	// Close the events channel to signal consumers to stop
	close(q.events)
	q.closed = true

	return nil
}

// IsClosed returns true if the queue has been closed.
func (q *InMemoryQueue) IsClosed() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.closed
}
