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

// Default dedupe configuration constants.
const (
	defaultDedupeMaxSize = 50000
)

// node represents a single entry in the linked list.
type node struct {
	id   string
	next *node
}

// reset clears the node state for reuse.
func (n *node) reset() {
	n.id = ""
	n.next = nil
}

// inMemoryDeduper implements Deduper using an in-memory linked list with FIFO eviction.
// For bounded mode (maxSize > 0): uses linked list with FIFO eviction and sync.Pool for nodes
// For unbounded mode (maxSize <= 0): uses simple map (no eviction, no size limit).
type inMemoryDeduper struct {
	mu       sync.RWMutex
	seen     map[string]*node // id -> node pointer for bounded mode, nil for unbounded
	head     *node            // head of linked list (oldest entry)
	tail     *node            // tail of linked list (newest entry)
	maxSize  int              // maximum number of IDs to keep in memory (0 or negative = UNBOUNDED)
	size     atomic.Int64     // current number of entries (atomic)
	nodePool sync.Pool        // pool for reusing node objects
	// (removed unused evictionPolicy, ttl, cleanupInterval, metricsEnabled fields)
}

// NewInMemoryDeduper creates a new in-memory deduper with configuration options.
func NewInMemoryDeduper(opts ...Option) Deduper {
	d := &inMemoryDeduper{
		maxSize: defaultDedupeMaxSize, // default max size
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
		// BOUNDED MODE: Use linked list with FIFO eviction
		// Check if we need to evict before adding the new entry
		if len(d.seen) >= d.maxSize {
			d.evictFIFO()
		}

		// Create new node from pool
		n := d.nodePool.Get().(*node)
		n.id = id
		n.next = nil

		// Add to tail (FIFO: newest at tail)
		if d.tail == nil {
			// First node
			d.head = n
			d.tail = n
		} else {
			// Add to tail
			d.tail.next = n
			d.tail = n
		}
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
		d.unrecordBounded(id)
	} else {
		d.unrecordUnbounded(id)
	}
}

// unrecordBounded removes an ID from the bounded mode (linked list + map).
func (d *inMemoryDeduper) unrecordBounded(id string) {
	node, exists := d.seen[id]
	if !exists {
		return
	}

	// Remove from map
	delete(d.seen, id)

	// Remove from linked list
	d.removeFromLinkedList(node)

	// Return node to pool
	node.reset()
	d.nodePool.Put(node)
	d.size.Add(-1)
}

// unrecordUnbounded removes an ID from the unbounded mode (map only).
func (d *inMemoryDeduper) unrecordUnbounded(id string) {
	if _, exists := d.seen[id]; exists {
		delete(d.seen, id)
		d.size.Add(-1)
	}
}

// removeFromLinkedList removes a node from the linked list.
func (d *inMemoryDeduper) removeFromLinkedList(node *node) {
	if d.head == node {
		d.removeHead()
	} else if d.tail == node {
		d.removeTail()
	} else {
		d.removeMiddle(node)
	}
}

// removeHead removes the head node from the linked list.
func (d *inMemoryDeduper) removeHead() {
	d.head = d.head.next
	if d.head == nil {
		d.tail = nil
	}
}

// removeTail removes the tail node from the linked list.
func (d *inMemoryDeduper) removeTail() {
	current := d.head
	for current != nil && current.next != d.tail {
		current = current.next
	}
	if current != nil {
		current.next = nil
		d.tail = current
	}
}

// removeMiddle removes a middle node from the linked list.
func (d *inMemoryDeduper) removeMiddle(node *node) {
	current := d.head
	for current != nil && current.next != node {
		current = current.next
	}
	if current != nil {
		current.next = node.next
	}
}

// evictFIFO removes the oldest entry (head of list) from the map.
// Must be called with d.mu.Lock() held.
func (d *inMemoryDeduper) evictFIFO() {
	if len(d.seen) == 0 || d.head == nil {
		return
	}

	// Remove the head (oldest entry)
	oldHead := d.head
	delete(d.seen, oldHead.id)

	// Update head pointer
	d.head = oldHead.next

	// If this was the last node, also clear tail
	if d.head == nil {
		d.tail = nil
	}

	// Return node to pool
	oldHead.reset()
	d.nodePool.Put(oldHead)
	d.size.Add(-1)
}

// Size returns the current number of entries in the deduper.
func (d *inMemoryDeduper) Size() int64 {
	return d.size.Load()
}
