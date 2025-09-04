// Package queue defines the contract for enqueuing and consuming events.
package queue

// Option applies a configuration option to the InMemoryQueue.
type Option func(*InMemoryQueue)

// WithCapacity sets the maximum capacity of the queue.
func WithCapacity(capacity int) Option {
	return func(q *InMemoryQueue) {
		if capacity > 0 {
			q.capacity = capacity
		}
	}
}

// WithBufferSize sets the buffer size for the events channel.
func WithBufferSize(size int) Option {
	return func(q *InMemoryQueue) {
		if size > 0 {
			q.bufferSize = size
		}
	}
}
