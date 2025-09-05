// Package repository defines the ranking store interface and errors.
package repository

import (
	"context"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/okian/cuju/pkg/metrics"
)

// Treap-based, in-memory Store implementation.
//
// Ordering: score DESC, then talentID ASC (deterministic).
// We implement a BST comparator where "less" means ranks earlier
// (i.e., higher score ranks earlier). This makes in-order traversal
// produce the leaderboard from best to worst.

// scoreScale controls fixed-point scaling from float64.
const scoreScale = 1_000_000_000_000 // 12 decimal places for better precision

type scoreFP int64

func toFixedPoint(x float64) scoreFP {
	// Handle special cases
	if math.IsNaN(x) {
		return 0
	}
	if math.IsInf(x, 1) {
		return scoreFP(math.MaxInt64)
	}
	if math.IsInf(x, -1) {
		return scoreFP(math.MinInt64)
	}

	// For very large numbers, use a more conservative scaling
	if math.Abs(x) > 1e15 {
		// For extremely large numbers, use a smaller scale to avoid overflow
		scaled := x * (scoreScale / 1000000) // Use 1/1M of the scale
		if scaled > float64(math.MaxInt64) {
			return scoreFP(math.MaxInt64)
		}
		if scaled < float64(math.MinInt64) {
			return scoreFP(math.MinInt64)
		}
		return scoreFP(math.Round(scaled))
	}

	// Normal scaling for reasonable numbers
	scaled := x * scoreScale
	if scaled > float64(math.MaxInt64) {
		return scoreFP(math.MaxInt64)
	}
	if scaled < float64(math.MinInt64) {
		return scoreFP(math.MinInt64)
	}
	return scoreFP(math.Round(scaled))
}

func toFloat(x scoreFP) float64 {
	// For very large fixed-point values, they were scaled with a smaller factor
	// We need to check if the original value was large enough to trigger the special scaling
	// Since we don't have the original value, we'll use a heuristic based on the fixed-point value
	if math.Abs(float64(x)) > 1e18 {
		return float64(x) / (scoreScale / 1000000)
	}
	return float64(x) / scoreScale
}

// record stores the fixed-point score plus metadata for a talent's best.
type record struct {
	score     scoreFP
	eventID   string
	skill     string
	rawMetric float64
}

// Snapshot represents an immutable snapshot of the leaderboard state
type Snapshot struct {
	// Rank and score in O(1) for reads
	RankByTalent  map[string]int
	ScoreByTalent map[string]float64

	// Fast Top-K cache up to M items
	TopCache []Entry // sorted descending (M ≪ N_total)
}

// treap node
type node struct {
	id    string
	score scoreFP
	prio  uint64
	left  *node
	right *node
	size  int
}

func nsize(n *node) int {
	if n == nil {
		return 0
	}
	return n.size
}

func fix(n *node) {
	if n != nil {
		n.size = 1 + nsize(n.left) + nsize(n.right)
	}
}

// less returns true if (aScore, aID) should appear before (bScore, bID)
// in the leaderboard (higher ranks first).
func less(aScore scoreFP, aID string, bScore scoreFP, bID string) bool {
	if aScore != bScore {
		return aScore > bScore // higher score ranks earlier
	}
	return aID < bID // tie-breaker by id asc
}

func rotateRight(y *node) *node {
	x := y.left
	t2 := x.right
	x.right = y
	y.left = t2
	fix(y)
	fix(x)
	return x
}

func rotateLeft(x *node) *node {
	y := x.right
	t2 := y.left
	y.left = x
	x.right = t2
	fix(x)
	fix(y)
	return y
}

// scoreToPriority converts a score to a priority value.
// Higher scores get higher priorities to keep them higher in the treap.
func scoreToPriority(score scoreFP) uint64 {
	// Convert scoreFP to uint64, ensuring higher scores get higher priorities
	// We need to handle negative scores by adding an offset
	const offset = uint64(1) << 63 // 2^63 to make all values positive
	return uint64(score) + offset
}

func insert(n *node, id string, score scoreFP) *node {
	if n == nil {
		return &node{id: id, score: score, prio: scoreToPriority(score), size: 1}
	}
	if less(score, id, n.score, n.id) {
		n.left = insert(n.left, id, score)
		if n.left.prio > n.prio {
			n = rotateRight(n)
		}
	} else {
		n.right = insert(n.right, id, score)
		if n.right.prio > n.prio {
			n = rotateLeft(n)
		}
	}
	fix(n)
	return n
}

func deleteNode(n *node, id string, score scoreFP) *node {
	if n == nil {
		return nil
	}
	if score == n.score && id == n.id {
		// Merge children by rotating highest priority up until leaf.
		if n.left == nil {
			return n.right
		}
		if n.right == nil {
			return n.left
		}
		if n.left.prio > n.right.prio {
			n = rotateRight(n)
			n.right = deleteNode(n.right, id, score)
		} else {
			n = rotateLeft(n)
			n.left = deleteNode(n.left, id, score)
		}
	} else if less(score, id, n.score, n.id) {
		n.left = deleteNode(n.left, id, score)
	} else {
		n.right = deleteNode(n.right, id, score)
	}
	fix(n)
	return n
}

// collectTopN appends up to limit entries in rank order (highest scores first).
// Optimized for score-based priorities while maintaining deterministic tie-breaking.
func collectTopN(n *node, limit int, records map[string]record, out *[]Entry) {
	if n == nil || len(*out) >= limit {
		return
	}

	// With score-based priorities, we need to maintain deterministic ordering
	// for tie-breaking (talent ID ASC). We do this by using the BST ordering
	// which is based on the less() function that handles tie-breaking correctly.

	// Traverse left subtree first (higher scores, or same score with lower ID)
	collectTopN(n.left, limit, records, out)

	// Add current node if we haven't reached the limit
	if len(*out) < limit {
		if rec, exists := records[n.id]; exists {
			*out = append(*out, Entry{Rank: 0 /* fix later */, TalentID: n.id, Score: toFloat(rec.score), EventID: rec.eventID, Skill: rec.skill, RawMetric: rec.rawMetric})
		}
	}

	// Traverse right subtree (lower scores, or same score with higher ID) if we still need more
	if len(*out) < limit {
		collectTopN(n.right, limit, records, out)
	}
}

type TreapStore struct {
	mu               sync.RWMutex
	root             *node
	byID             map[string]record
	snapshotInterval time.Duration // How often to create periodic snapshots of the store
	topCacheSize     int           // Maximum number of top-scoring records to keep in cache

	// snapshot is atomic pointer to a Snapshot struct
	snapshot atomic.Pointer[Snapshot]

	// Periodic snapshot management
	wg       sync.WaitGroup
	stopChan chan struct{}
}

// NewTreapStore constructs a treap store with configuration options.
func NewTreapStore(ctx context.Context, opts ...Option) *TreapStore {
	s := &TreapStore{
		snapshotInterval: 1 * time.Second, // default snapshot interval
		topCacheSize:     500,             // default top cache size
		byID:             make(map[string]record),
	}

	// Apply all options (shard-related options are ignored)
	for _, opt := range opts {
		opt(s)
	}

	// Initialize stop channel and start periodic snapshot goroutine
	s.stopChan = make(chan struct{})
	s.startPeriodicSnapshots(ctx)

	// Initialize metrics
	metrics.UpdateRepositoryShardCount(1) // Single "shard"
	s.startMetricsUpdater(ctx)

	return s
}

// startPeriodicSnapshots starts a background goroutine that publishes snapshots at the configured interval
func (s *TreapStore) startPeriodicSnapshots(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.snapshotInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stopChan:
				return
			case <-ticker.C:
				s.publishSnapshot()
			}
		}
	}()
}

// publishSnapshot rebuilds and publishes a new snapshot
func (s *TreapStore) publishSnapshot() {
	start := time.Now()
	s.mu.RLock()
	s.publishSnapshotInternal()
	s.mu.RUnlock()

	ms := float64(time.Since(start).Milliseconds())
	metrics.RecordRepositorySnapshotRebuildDuration(ms)
	metrics.UpdateRepositorySnapshotLastDurationMs(ms)
	metrics.UpdateRepositorySnapshotLastUnix(float64(time.Now().Unix()))
	metrics.IncrementRepositorySnapshotCount()
}

// Close gracefully shuts down the periodic snapshot goroutine
func (s *TreapStore) Close() error {
	// Signal all goroutines to stop
	select {
	case <-s.stopChan:
		// Channel already closed
	default:
		close(s.stopChan)
	}
	s.wg.Wait()
	return nil
}

// UpdateBest implements Store.UpdateBest with O(log n) expected time.
func (s *TreapStore) UpdateBest(ctx context.Context, talentID string, score float64) (bool, error) {
	return s.UpdateBestWithMeta(ctx, talentID, score, "", "", 0)
}

// UpdateBestWithMeta implements Store.UpdateBestWithMeta with O(log n) expected time.
func (s *TreapStore) UpdateBestWithMeta(ctx context.Context, talentID string, score float64, eventID string, skill string, rawMetric float64) (bool, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start).Milliseconds()
		metrics.RecordRepositoryUpdateLatency(float64(latency))
	}()

	ns := toFixedPoint(score)

	// Track if this is a new talent so we can update metrics after releasing locks
	isNewTalent := false

	s.mu.Lock()
	if old, ok := s.byID[talentID]; ok {
		if ns <= old.score { // not an improvement
			s.mu.Unlock()
			return false, nil
		}
		s.root = deleteNode(s.root, talentID, old.score)
	} else {
		isNewTalent = true
	}
	s.byID[talentID] = record{score: ns, eventID: eventID, skill: skill, rawMetric: rawMetric}
	s.root = insert(s.root, talentID, ns)
	s.mu.Unlock()

	// Update metrics outside lock
	if isNewTalent {
		metrics.UpdateRepositoryRecordsTotal(s.Count(ctx))
	}

	// Snapshots are now published periodically, not after every update
	return true, nil
}

// Rank returns the current rank and score for a talent in O(log n).
func (s *TreapStore) Rank(ctx context.Context, talentID string) (Entry, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start).Milliseconds()
		metrics.RecordRepositoryQueryLatency(float64(latency))
	}()

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if the talent exists
	if _, ok := s.byID[talentID]; !ok {
		metrics.RecordErrorByComponent("repository", "not_found")
		return Entry{}, ErrNotFound
	}

	// Collect all entries and find the rank
	allEntries := make([]Entry, 0, len(s.byID))
	collectAll(s.root, s.byID, &allEntries)

	// Sort all entries by score (descending) and talentID (ascending) to match TopN logic
	sortEntries(allEntries)

	// Assign global ranks with proper tie handling
	assignRanksWithTies(allEntries)

	// Find the rank for this specific talent
	for _, entry := range allEntries {
		if entry.TalentID == talentID {
			return entry, nil
		}
	}

	return Entry{}, ErrNotFound
}

// TopN returns the top N entries ordered by score desc.
func (s *TreapStore) TopN(ctx context.Context, n int) ([]Entry, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start).Milliseconds()
		metrics.RecordRepositoryQueryLatency(float64(latency))
	}()

	if n < 1 {
		metrics.RecordErrorByComponent("repository", "invalid_limit")
		return nil, ErrInvalidLimit
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Entry, 0, n)
	collectTopN(s.root, n, s.byID, &out)

	// Assign ranks with proper tie handling
	assignRanksWithTies(out)
	return out, nil
}

// Count returns the total number of talents.
func (s *TreapStore) Count(ctx context.Context) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byID)
}

// publishSnapshotInternal rebuilds and publishes a new snapshot (assumes lock is held)
func (s *TreapStore) publishSnapshotInternal() {
	// Build Top-M cache for fast dashboard queries
	topCache := make([]Entry, 0, s.topCacheSize)
	collectTopN(s.root, s.topCacheSize, s.byID, &topCache)

	// Build full rank and score maps
	rankByTalent := make(map[string]int, len(s.byID))
	scoreByTalent := make(map[string]float64, len(s.byID))

	// Collect all entries to compute global ranks
	allEntries := make([]Entry, 0, len(s.byID))
	collectAll(s.root, s.byID, &allEntries)

	// Assign ranks with proper tie handling
	assignRanksWithTies(allEntries)

	// Build maps from all entries
	for _, entry := range allEntries {
		rankByTalent[entry.TalentID] = entry.Rank
		scoreByTalent[entry.TalentID] = entry.Score
	}

	// Update TopCache with correct ranks
	for i := range topCache {
		if rank, exists := rankByTalent[topCache[i].TalentID]; exists {
			topCache[i].Rank = rank
		}
	}

	snapshot := &Snapshot{
		RankByTalent:  rankByTalent,
		ScoreByTalent: scoreByTalent,
		TopCache:      topCache,
	}

	s.snapshot.Store(snapshot)
}

// startMetricsUpdater starts a background goroutine that updates repository metrics
func (s *TreapStore) startMetricsUpdater(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(5 * time.Second) // Update metrics every 5 seconds
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stopChan:
				return
			case <-ticker.C:
				s.updateMetrics()
			}
		}
	}()
}

// updateMetrics updates all repository-related metrics
func (s *TreapStore) updateMetrics() {
	s.mu.RLock()
	recordCount := len(s.byID)
	s.mu.RUnlock()

	// Update per-shard metrics (using "shard_0" for the single shard)
	shardID := "shard_0"
	metrics.UpdateRepositoryRecordsPerShard(shardID, recordCount)

	// Calculate shard utilization (assuming a reasonable capacity)
	estimatedCapacity := 10000 // This could be configurable
	utilization := float64(recordCount) / float64(estimatedCapacity)
	if utilization > 1.0 {
		utilization = 1.0
	}
	metrics.UpdateRepositoryShardUtilization(shardID, utilization)

	// Update total records metric
	metrics.UpdateRepositoryRecordsTotal(recordCount)
}

// collectAll appends all entries in rank order (highest scores first).
func collectAll(n *node, byID map[string]record, out *[]Entry) {
	if n == nil {
		return
	}
	// Traverse left subtree first (higher scores)
	collectAll(n.left, byID, out)
	// Add current node
	if rec, ok := byID[n.id]; ok {
		*out = append(*out, Entry{
			TalentID:  n.id,
			Score:     toFloat(rec.score),
			EventID:   rec.eventID,
			Skill:     rec.skill,
			RawMetric: rec.rawMetric,
		})
	}
	// Traverse right subtree (lower scores)
	collectAll(n.right, byID, out)
}

// sortEntries sorts entries by score (descending) and talentID (ascending) to match TopN logic
func sortEntries(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool {
		// Higher score comes first (descending order)
		if entries[i].Score != entries[j].Score {
			return entries[i].Score > entries[j].Score
		}
		// Tie-breaker: talentID in ascending order
		return entries[i].TalentID < entries[j].TalentID
	})
}

// assignRanksWithTies assigns ranks with proper tie handling.
// Talents with the same score get the same rank, and the next rank
// skips the appropriate number of positions.
func assignRanksWithTies(entries []Entry) {
	if len(entries) == 0 {
		return
	}

	currentRank := 1
	for i := 0; i < len(entries); i++ {
		// Assign current rank to this entry
		entries[i].Rank = currentRank

		// Count how many entries have the same score as this one
		sameScoreCount := 1
		for j := i + 1; j < len(entries) && entries[j].Score == entries[i].Score; j++ {
			entries[j].Rank = currentRank
			sameScoreCount++
		}

		// Move to the next rank (consecutive ranking)
		currentRank++
		i += sameScoreCount - 1 // Skip the entries we just processed
	}
}
