# CUJU Data Structures & Algorithms

## Overview

This document provides a detailed analysis of the data structures and algorithms used in the CUJU leaderboard system, including time and space complexities, implementation details, and design rationale.

## 1. Treap-Based Leaderboard Store

### Data Structure: Treap (Tree + Heap)

A **Treap** is a binary search tree where each node has both a key (for BST ordering) and a priority (for heap ordering). This combination provides the benefits of both structures:

- **BST Property**: Maintains sorted order for efficient range queries
- **Heap Property**: Self-balancing through random priorities

### Implementation Details

```go
type node struct {
    id    string    // talent_id (BST key)
    score scoreFP   // score (BST ordering)
    prio  uint64    // random priority (heap ordering)
    left  *node     // left child
    right *node     // right child
    size  int       // subtree size for rank queries
}
```

### Key Operations

#### 1. Insert Operation
```go
func insert(n *node, id string, score scoreFP) *node
```

**Time Complexity**: **O(log n)** average case
**Space Complexity**: **O(1)** additional space

**Algorithm**:
1. Insert as in BST based on score (descending order)
2. If heap property violated, rotate to fix
3. Update subtree sizes

**Rotations**:
- **Right Rotation**: When left child has higher priority
- **Left Rotation**: When right child has higher priority

#### 2. Delete Operation
```go
func deleteNode(n *node, id string, score scoreFP) *node
```

**Time Complexity**: **O(log n)** average case
**Space Complexity**: **O(1)** additional space

**Algorithm**:
1. Find node to delete
2. Rotate down until it becomes a leaf
3. Remove the leaf node
4. Update subtree sizes

#### 3. Rank Query
```go
func rank(n *node, id string, score scoreFP) int
```

**Time Complexity**: **O(log n)** average case
**Space Complexity**: **O(1)** additional space

**Algorithm**:
1. Traverse tree to find target node
2. Count nodes with higher scores (left subtree + ancestors)
3. Return 1-based rank

#### 4. Top-N Query
```go
func collectTopN(n *node, limit int, records map[string]record, out *[]Entry)
```

**Time Complexity**: **O(log n + N)** where N is the limit
**Space Complexity**: **O(N)** for output

**Algorithm**:
1. In-order traversal (left-first for descending order)
2. Collect up to `limit` entries
3. Stop early when limit reached

### Fixed-Point Arithmetic

**Problem**: Floating-point precision issues in score comparisons
**Solution**: Convert scores to fixed-point integers

```go
const scoreScale = 1_000_000_000_000 // 12 decimal places

type scoreFP int64

func toFixedPoint(x float64) scoreFP {
    // Handle special cases (NaN, Inf)
    // Scale by scoreScale
    // Clamp to int64 range
    return scoreFP(math.Round(x * scoreScale))
}
```

**Benefits**:
- Exact comparisons without floating-point errors
- Deterministic ordering
- Consistent tie-breaking

### Sharding Strategy

**Hash Function**: FNV-1a (64-bit)
```go
func (s *TreapStore) shardFor(id string) *shard {
    hasher := fnv.New64a()
    _, _ = hasher.Write([]byte(id))
    idx := int(hasher.Sum64() % uint64(len(s.shards)))
    return &s.shards[idx]
}
```

**Benefits**:
- Even distribution across shards
- O(1) shard selection
- Good load balancing

**Trade-offs**:
- Global operations require cross-shard coordination
- More complex rank calculations

## 2. In-Memory Event Queue

### Data Structure: Buffered Channel

```go
type InMemoryQueue struct {
    events     chan Event  // buffered channel
    capacity   int         // maximum capacity
    bufferSize int         // channel buffer size
    mu         sync.RWMutex
    closed     bool
}
```

### Operations

#### 1. Enqueue
```go
func (q *InMemoryQueue) Enqueue(ctx context.Context, e Event) bool
```

**Time Complexity**: **O(1)**
**Space Complexity**: **O(1)**

**Algorithm**:
1. Check if queue is closed
2. Check capacity limit
3. Non-blocking channel send with select
4. Update metrics

#### 2. Dequeue
```go
func (q *InMemoryQueue) Dequeue(ctx context.Context) <-chan Event
```

**Time Complexity**: **O(1)** per event
**Space Complexity**: **O(1)**

**Algorithm**:
1. Return wrapped channel
2. Forward events from internal channel
3. Handle context cancellation
4. Update metrics

### Backpressure Handling

**Strategy**: Non-blocking enqueue with capacity checks
**Benefits**: Prevents memory exhaustion
**Trade-off**: Events may be dropped under high load

## 3. Deduplication Cache

### Data Structure: Hash Map + Linked List (LIFO)

```go
type inMemoryDeduper struct {
    mu       sync.RWMutex
    seen     map[string]*node  // id -> node pointer
    head     *node             // head of linked list
    maxSize  int               // maximum cache size
    size     atomic.Int64      // current size
    nodePool sync.Pool         // node reuse pool
}
```

### Operations

#### 1. SeenAndRecord
```go
func (d *inMemoryDeduper) SeenAndRecord(ctx context.Context, id string) bool
```

**Time Complexity**: **O(1)**
**Space Complexity**: **O(1)**

**Algorithm**:
1. Check if ID exists in hash map
2. If exists, return true (duplicate)
3. If not exists and cache full, evict LIFO
4. Add new node to head of list
5. Update hash map and size counter

#### 2. Eviction (LIFO)
```go
func (d *inMemoryDeduper) evictLIFO()
```

**Time Complexity**: **O(1)** average case
**Space Complexity**: **O(1)**

**Algorithm**:
1. Find tail of linked list
2. Remove from hash map
3. Update list pointers
4. Return node to pool

### Memory Management

**Node Pool**: Reuse nodes to reduce GC pressure
```go
nodePool: sync.Pool{
    New: func() interface{} {
        return &node{}
    },
}
```

**Benefits**:
- Reduced memory allocations
- Lower GC pressure
- Better performance

## 4. Worker Pool

### Data Structure: Goroutine Pool

```go
type WorkerPool struct {
    workers []*InMemoryWorker
    queue   Queue
    scorer  Scorer
    updater Updater
    // ... other fields
}
```

### Operations

#### 1. Event Processing
```go
func (w *InMemoryWorker) processEvent(ctx context.Context, event queue.Event) error
```

**Time Complexity**: **O(1)** + scoring latency
**Space Complexity**: **O(1)**

**Algorithm**:
1. Score event (with simulated latency)
2. Update leaderboard if score improved
3. Record metrics
4. Handle errors

#### 2. Pool Management
```go
func (p *WorkerPool) Start(ctx context.Context)
```

**Time Complexity**: **O(W)** where W is worker count
**Space Complexity**: **O(W)**

**Algorithm**:
1. Start W goroutines
2. Each worker processes events from shared queue
3. Handle graceful shutdown

## 5. Global Operations

### Cross-Shard K-Way Merge

**Problem**: Top-N queries need global ordering across shards
**Solution**: K-way merge of shard results

```go
func (s *TreapStore) TopN(ctx context.Context, n int) ([]Entry, error) {
    // 1. Collect top-N from each shard
    per := make([]shardSlice, len(s.shards))
    for i := range s.shards {
        sh := &s.shards[i]
        sh.mu.RLock()
        out := make([]Entry, 0, n)
        collectTopN(sh.root, n, sh.byID, &out)
        sh.mu.RUnlock()
        per[i] = shardSlice{entries: out, idx: 0}
    }
    
    // 2. K-way merge
    out := make([]Entry, 0, n)
    for len(out) < n {
        bestIdx := -1
        var best Entry
        for i := range per {
            // Find best candidate across all shards
        }
        if bestIdx == -1 {
            break // all exhausted
        }
        out = append(out, best)
        per[bestIdx].idx++
    }
    
    return out, nil
}
```

**Time Complexity**: **O(n_shard × log n_shard + N)**
- Collect from shards: O(n_shard × log n_shard)
- K-way merge: O(N)

**Space Complexity**: **O(n_shard × N + N)**

### Global Rank Calculation

**Problem**: Rank queries need global position across all shards
**Solution**: Collect all entries and calculate global rank

```go
func (s *TreapStore) Rank(ctx context.Context, talentID string) (Entry, error) {
    // 1. Collect all entries from all shards
    allEntries := make([]Entry, 0)
    for i := range s.shards {
        sh := &s.shards[i]
        sh.mu.RLock()
        shardEntries := make([]Entry, 0, len(sh.byID))
        collectAll(sh.root, sh.byID, &shardEntries)
        sh.mu.RUnlock()
        allEntries = append(allEntries, shardEntries...)
    }
    
    // 2. Sort and assign global ranks
    sortEntries(allEntries)
    assignRanksWithTies(allEntries)
    
    // 3. Find target talent
    for _, entry := range allEntries {
        if entry.TalentID == talentID {
            return entry, nil
        }
    }
    
    return Entry{}, ErrNotFound
}
```

**Time Complexity**: **O(n_shard × log n_shard)**
**Space Complexity**: **O(N)** where N is total talents

## 6. Performance Optimizations

### 1. Periodic Snapshots

**Problem**: Frequent global operations are expensive
**Solution**: Periodic snapshot generation

```go
func (s *TreapStore) startPeriodicSnapshots(ctx context.Context) {
    ticker := time.NewTicker(s.snapshotInterval)
    for {
        select {
        case <-ticker.C:
            s.publishAllSnapshots()
        }
    }
}
```

**Benefits**:
- Fast read access to top entries
- Reduced lock contention
- Better cache locality

**Trade-offs**:
- Stale data during snapshot intervals
- Additional memory usage
- Background CPU overhead

### 2. Memory Pool Usage

**Node Pool**: Reuse treap nodes and deduplication nodes
**Benefits**: Reduced GC pressure, better performance

### 3. Lock Granularity

**Shard-Level Locks**: Each shard has its own RWMutex
**Benefits**: Better concurrency, reduced lock contention

### 4. Atomic Operations

**Size Counters**: Use atomic operations for thread-safe counters
**Benefits**: No locking required for simple increments

## 7. Complexity Analysis Summary

| Operation | Data Structure | Time Complexity | Space Complexity | Notes |
|-----------|---------------|-----------------|------------------|-------|
| Insert | Treap | O(log n) | O(1) | Average case |
| Delete | Treap | O(log n) | O(1) | Average case |
| Rank | Treap | O(log n) | O(1) | Per shard |
| Top-N | Treap | O(log n + N) | O(N) | Per shard |
| Global Top-N | Sharded Treap | O(n_shard × log n_shard + N) | O(n_shard × N) | Cross-shard merge |
| Global Rank | Sharded Treap | O(n_shard × log n_shard) | O(N) | Global calculation |
| Enqueue | Channel | O(1) | O(1) | Non-blocking |
| Dequeue | Channel | O(1) | O(1) | Per event |
| Dedupe Check | Hash Map | O(1) | O(1) | Average case |
| Eviction | Linked List | O(1) | O(1) | LIFO |

Where:
- n = number of talents per shard
- n_shard = number of shards
- N = total number of talents
- M = deduplication cache size

## 8. Design Rationale

### Why Treap Over Simple Map?

**The Problem with Hash Maps for Leaderboards**:
While hash maps provide O(1) insert, update, and read operations, they have a critical limitation for leaderboard systems:

```go
// Simple map approach - what we avoided
type SimpleLeaderboard struct {
    talents map[string]float64  // talent_id -> score
}

func (s *SimpleLeaderboard) TopN(n int) []Entry {
    // PROBLEM: This requires O(n) time complexity
    // 1. Collect all entries: O(n)
    // 2. Sort by score: O(n log n)  
    // 3. Take top N: O(n)
    // Total: O(n log n) - becomes expensive for large datasets
}
```

**Why This Matters for Large Datasets**:
- **30M Talents**: O(n log n) = O(30M × log(30M)) ≈ O(30M × 25) ≈ 750M operations
- **Performance Impact**: Each leaderboard query would take seconds, not milliseconds
- **User Experience**: Unacceptable latency for real-time leaderboards

**Alternative Solutions We Considered**:

1. **Cache in Front of Repository**:
   ```go
   type CachedLeaderboard struct {
       talents map[string]float64
       topCache []Entry  // Keep top 1000 cached
   }
   ```
   - **Problem**: Leaderboard boundaries are not well-defined
   - **Issue**: What if user wants top 2000? Cache miss = full scan
   - **Complexity**: Cache invalidation becomes unreliable
   - **Result**: Not feasible for dynamic, unbounded leaderboards

   **Detailed Cache Boundary Problems**:
   
   **Scenario 1 - Variable Query Sizes**:
   - Cache top 1000 entries
   - User requests top 500 → Cache hit ✅
   - User requests top 1500 → Cache miss ❌ (fallback to O(n log n) scan)
   - User requests top 10,000 → Cache miss ❌ (fallback to O(n log n) scan)
   
   **Scenario 2 - Dynamic Score Changes**:
   - Cache contains top 1000 with scores 900-1000
   - New talent gets score 950 → Cache invalidation needed
   - Which entries to evict? How to maintain cache consistency?
   - Partial cache updates become complex and error-prone
   
   **Scenario 3 - Memory vs Performance Trade-off**:
   - Cache top 10,000 → High memory usage, but covers more queries
   - Cache top 100 → Low memory, but frequent cache misses
   - No optimal cache size exists for all possible query patterns
   
   **Scenario 4 - Real-world Usage Patterns**:
   - Dashboard shows top 10 (cache hit)
   - User scrolls to see top 100 (cache hit)
   - User wants to see their rank among top 1000 (cache miss)
   - Admin wants to export top 10,000 (cache miss)
   - Each cache miss triggers expensive O(n log n) operation

2. **Sorted Array**:
   - **Insert**: O(n) - too slow for frequent updates
   - **Lookup**: O(log n) - acceptable
   - **Top-N**: O(n) - still problematic

3. **Skip List**:
   - **All Operations**: O(log n) - good performance
   - **Memory Overhead**: Higher than Treap
   - **Complexity**: More complex implementation

**Why Treap is the Optimal Choice**:

1. **Self-Balancing**: Random priorities ensure O(log n) height
2. **Efficient Operations**: All operations are O(log n)
3. **Deterministic Ordering**: Consistent tie-breaking
4. **Memory Efficient**: No extra balancing information needed
5. **Optimal Top-N**: O(n_shard × log n_shard + N) where N is the limit, not total size

**Performance Comparison**:

| Operation | Hash Map | Sorted Array | Skip List | **Treap** |
|-----------|----------|--------------|-----------|-----------|
| Insert | O(1) | O(n) | O(log n) | **O(log n)** |
| Update | O(1) | O(n) | O(log n) | **O(log n)** |
| Read | O(1) | O(log n) | O(log n) | **O(log n)** |
| Top-N | **O(n log n)** | O(n) | O(log n + N) | **O(n_shard × log n_shard + N)** |
| Rank | O(n) | O(log n) | O(log n) | **O(log n)** |

**The Key Insight**: 
We compromised on individual read/update performance (O(1) → O(log n)) to achieve optimal Top-N performance (O(n log n) → O(n_shard × log n_shard + N)). This trade-off is justified because:

- **Top-N queries are the most frequent and critical operations**
- **O(log n) is still very fast** (log(30M) ≈ 25 operations)
- **The system is read-heavy** - leaderboard queries happen more often than updates

**Note on Sharding Complexity**: The actual Top-N complexity is O(n_shard × log n_shard + N) due to cross-shard k-way merging. With 8 shards and 30M talents, this becomes O(8 × log(3.75M) + N) ≈ O(8 × 22 + N) ≈ O(176 + N), which is still much better than O(n log n) = O(745M) for a simple map approach.

### Why Sharding?

1. **Concurrency**: Multiple shards can be updated simultaneously
2. **Scalability**: Linear scaling with number of shards
3. **Load Distribution**: Even distribution of data and operations

### Why Fixed-Point Arithmetic?

1. **Precision**: Eliminates floating-point comparison issues
2. **Determinism**: Consistent ordering across runs
3. **Performance**: Integer comparisons are faster

### Why LIFO Eviction?

1. **Simplicity**: Easy to implement and understand
2. **Performance**: O(1) eviction from tail
3. **Memory Efficiency**: Reuses nodes from pool

This comprehensive analysis shows that the CUJU system uses well-chosen data structures and algorithms that provide excellent performance characteristics while maintaining simplicity and reliability.
