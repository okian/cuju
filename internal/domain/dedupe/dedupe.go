// Package dedupe defines the interface for idempotency tracking.
package dedupe

import (
	"context"
	"sync"
	"sync/atomic"
)

// Deduper records seen event IDs to ensure at-most-once processing.
type Deduper interface {
	// SeenAndRecord atomically checks if id was seen and records it if not.
	// Returns true if id was already seen, false if it was newly recorded.
	// This is the ONLY method for deduplication - thread-safe and atomic.
	SeenAndRecord(ctx context.Context, id string) bool

	// Unrecord removes an ID from the seen list, allowing it to be retried.
	// This should only be used when an event was marked as seen but failed
	// to be processed (e.g., queue backpressure).
	Unrecord(ctx context.Context, id string)

	Size() int64
}

// node represents a single entry in the linked list
type node struct {
	id   string
	next *node
}

// reset clears the node state for reuse
func (n *node) reset() {
	n.id = ""
	n.next = nil
}

// inMemoryDeduper implements Deduper using an in-memory linked list with LIFO eviction.
// For bounded mode (maxSize > 0): uses linked list with LIFO eviction and sync.Pool for nodes
// For unbounded mode (maxSize <= 0): uses simple map (no eviction, no size limit)
type inMemoryDeduper struct {
	mu       sync.RWMutex
	seen     map[string]*node // id -> node pointer for bounded mode, nil for unbounded
	head     *node            // head of linked list (most recently added)
	maxSize  int              // maximum number of IDs to keep in memory (0 or negative = UNBOUNDED)
	size     atomic.Int64     // current number of entries (atomic)
	nodePool sync.Pool        // pool for reusing node objects
	// (removed unused evictionPolicy, ttl, cleanupInterval, metricsEnabled fields)
}

// NewInMemoryDeduper creates a new in-memory deduper with configuration options.
func NewInMemoryDeduper(opts ...Option) Deduper {
	d := &inMemoryDeduper{
		maxSize: 50000, // default max size
	}

	// Apply all options
	for _, opt := range opts {
		opt(d)
	}

	// Initialize the seen map
	d.seen = make(map[string]*node)

	// Initialize sync.Pool for node reuse in bounded mode
	if d.maxSize > 0 {
		d.nodePool = sync.Pool{
			New: func() interface{} {
				return &node{}
			},
		}
	}

	return d
}

// SeenAndRecord atomically checks if id was seen and records it if not.
// This is the ONLY method for deduplication.
// Returns true if id was already seen, false if it was newly recorded.
func (d *inMemoryDeduper) SeenAndRecord(ctx context.Context, id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if already seen
	if _, exists := d.seen[id]; exists {
		return true // Already seen
	}

	if d.maxSize > 0 {
		// BOUNDED MODE: Use linked list with LIFO eviction
		// Check if we need to evict before adding the new entry
		if len(d.seen) >= d.maxSize {
			d.evictLIFO()
		}

		// Create new node from pool
		n := d.nodePool.Get().(*node)
		n.id = id
		n.next = d.head

		// Update head and map
		d.head = n
		d.seen[id] = n
	} else {
		// UNBOUNDED MODE: Just use map
		d.seen[id] = nil
	}
	d.size.Add(1)
	return false // Newly recorded
}

// Unrecord removes an ID from the seen list, allowing it to be retried.
func (d *inMemoryDeduper) Unrecord(ctx context.Context, id string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.maxSize > 0 {
		// BOUNDED MODE: Remove from linked list and map
		if node, exists := d.seen[id]; exists {
			// Remove from map
			delete(d.seen, id)

			// Remove from linked list
			if d.head == node {
				// Node is at head
				d.head = node.next
			} else {
				// Find and remove node from middle/tail
				current := d.head
				for current != nil && current.next != node {
					current = current.next
				}
				if current != nil {
					current.next = node.next
				}
			}

			// Return node to pool
			node.reset()
			d.nodePool.Put(node)

			d.size.Add(-1)
		}
	} else {
		// UNBOUNDED MODE: Just remove from map
		if _, exists := d.seen[id]; exists {
			delete(d.seen, id)
			d.size.Add(-1)
		}
	}
}

// evictLIFO removes the least recently added entry (tail of list) from the map.
// Must be called with d.mu.Lock() held.
func (d *inMemoryDeduper) evictLIFO() {
	if len(d.seen) == 0 || d.head == nil {
		return
	}

	// Find the second-to-last node
	var prev *node
	current := d.head

	// If there's only one node, remove it
	if current.next == nil {
		delete(d.seen, current.id)
		current.reset()
		d.nodePool.Put(current)
		d.head = nil
		d.size.Add(-1)
		return
	}

	// Find the second-to-last node
	for current.next != nil {
		prev = current
		current = current.next
	}

	// Remove the last node (tail)
	if prev != nil {
		prev.next = nil
		delete(d.seen, current.id)
		current.reset()
		d.nodePool.Put(current)
		d.size.Add(-1)
	}
}

// Size returns the current number of entries in the deduper.
func (d *inMemoryDeduper) Size() int64 {
	return d.size.Load()
}
