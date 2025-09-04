// Package repository defines the ranking store interface and errors.
package repository

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/okian/cuju/pkg/metrics"
)

// Treap-based, sharded in-memory Store implementation.
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

// rng returns a random 64-bit priority.
func rng() uint64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		return binary.LittleEndian.Uint64(b[:])
	}
	// Fallback (should not happen in practice).
	h := fnv.New64()
	_, _ = h.Write([]byte("fallback"))
	return h.Sum64()
}

func insert(n *node, id string, score scoreFP) *node {
	if n == nil {
		return &node{id: id, score: score, prio: rng(), size: 1}
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

// rank returns 1-based rank of (id, score).
// If not found, returns 0.
func rank(n *node, id string, score scoreFP) int {
	if n == nil {
		return 0
	}
	if score == n.score && id == n.id {
		return nsize(n.left) + 1
	}
	if less(score, id, n.score, n.id) {
		return rank(n.left, id, score)
	}
	// It is after current node: skip left subtree + current + recurse right
	r := rank(n.right, id, score)
	if r == 0 {
		return 0
	}
	return nsize(n.left) + 1 + r
}

// countGreater returns the number of nodes with score greater than or equal to x.
func countGreater(n *node, x scoreFP) int {
	if n == nil {
		return 0
	}
	if less(x, "", n.score, n.id) {
		return 1 + nsize(n.left) + countGreater(n.right, x)
	}
	return countGreater(n.left, x)
}

// collectTopN appends up to limit entries in rank order (highest scores first).
// For a treap ordered by score (descending), we traverse left-first to get highest scores.
func collectTopN(n *node, limit int, records map[string]record, out *[]Entry) {
	if n == nil || len(*out) >= limit {
		return
	}

	// Traverse left subtree first (higher scores)
	collectTopN(n.left, limit, records, out)

	// Add current node if we haven't reached the limit
	if len(*out) < limit {
		if rec, exists := records[n.id]; exists {
			*out = append(*out, Entry{Rank: 0 /* fix later */, TalentID: n.id, Score: toFloat(rec.score), EventID: rec.eventID, Skill: rec.skill, RawMetric: rec.rawMetric})
		}
	}

	// Traverse right subtree (lower scores) if we still need more
	if len(*out) < limit {
		collectTopN(n.right, limit, records, out)
	}
}

// shard holds an independent treap and id->score map protected by RWMutex.
type shard struct {
	mu   sync.RWMutex
	root *node
	byID map[string]record
	// snapshot is atomic pointer to a Snapshot struct
	snapshot atomic.Pointer[Snapshot]
}

type TreapStore struct {
	shards           []shard       // Array of shards for concurrent access, each containing a treap and id->score map
	shardCount       int           // Number of shards to distribute data across for better concurrency
	snapshotInterval time.Duration // How often to create periodic snapshots of the store
	topCacheSize     int           // Maximum number of top-scoring records to keep in cache
	// (removed unused scorePrecision and compactionThreshold fields)

	// Periodic snapshot management
	wg       sync.WaitGroup
	stopChan chan struct{}
}

// NewTreapStore constructs a sharded treap store with configuration options.
func NewTreapStore(ctx context.Context, opts ...Option) *TreapStore {
	s := &TreapStore{
		shardCount:       16,              // default shard count
		snapshotInterval: 2 * time.Second, // default snapshot interval
		topCacheSize:     200,             // default top cache size
	}

	// Apply all options
	for _, opt := range opts {
		opt(s)
	}

	// Initialize shards
	s.shards = make([]shard, s.shardCount)
	for i := range s.shards {
		s.shards[i].byID = make(map[string]record)
	}

	// Initialize stop channel and start periodic snapshot goroutine
	s.stopChan = make(chan struct{})
	s.startPeriodicSnapshots(ctx)

	// Initialize metrics
	metrics.UpdateRepositoryShardCount(s.shardCount)
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
				s.publishAllSnapshots()
			}
		}
	}()
}

// publishAllSnapshots publishes snapshots for all shards
func (s *TreapStore) publishAllSnapshots() {
	start := time.Now()
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.RLock()
		sh.publishSnapshot(s.topCacheSize)
		sh.mu.RUnlock()
	}
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

func (s *TreapStore) shardFor(id string) *shard {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(id))
	idx := int(hasher.Sum64() % uint64(len(s.shards)))
	return &s.shards[idx]
}

// UpdateBest implements Store.UpdateBest with O(log n_shard) expected time.
func (s *TreapStore) UpdateBest(ctx context.Context, talentID string, score float64) (bool, error) {
	return s.UpdateBestWithMeta(ctx, talentID, score, "", "", 0)
}

// UpdateBestWithMeta implements Store.UpdateBestWithMeta with O(log n_shard) expected time.
func (s *TreapStore) UpdateBestWithMeta(ctx context.Context, talentID string, score float64, eventID string, skill string, rawMetric float64) (bool, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start).Milliseconds()
		metrics.RecordRepositoryUpdateLatency(float64(latency))
	}()

	sh := s.shardFor(talentID)
	ns := toFixedPoint(score)

	// Track if this is a new talent so we can update metrics after releasing locks
	isNewTalent := false

	sh.mu.Lock()
	if old, ok := sh.byID[talentID]; ok {
		if ns <= old.score { // not an improvement
			sh.mu.Unlock()
			return false, nil
		}
		sh.root = deleteNode(sh.root, talentID, old.score)
	} else {
		isNewTalent = true
	}
	sh.byID[talentID] = record{score: ns, eventID: eventID, skill: skill, rawMetric: rawMetric}
	sh.root = insert(sh.root, talentID, ns)
	sh.mu.Unlock()

	// Update metrics outside shard lock (Count acquires read locks on shards)
	if isNewTalent {
		metrics.UpdateRepositoryRecordsTotal(s.Count(ctx))
	}

	// Snapshots are now published periodically, not after every update
	return true, nil
}

// Rank returns the current rank and score for a talent in O(log n_shard).
func (s *TreapStore) Rank(ctx context.Context, talentID string) (Entry, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start).Milliseconds()
		metrics.RecordRepositoryQueryLatency(float64(latency))
	}()

	// First, check if the talent exists in any shard
	var found bool
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.RLock()
		if _, ok := sh.byID[talentID]; ok {
			found = true
		}
		sh.mu.RUnlock()
		if found {
			break
		}
	}

	if !found {
		metrics.RecordErrorByComponent("repository", "not_found")
		return Entry{}, NewKind("leaderboard.treap.rank", ErrNotFound)
	}

	// Collect all entries from all shards and merge them
	allEntries := make([]Entry, 0)
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.RLock()
		shardEntries := make([]Entry, 0, len(sh.byID))
		collectAll(sh.root, sh.byID, &shardEntries)
		sh.mu.RUnlock()
		allEntries = append(allEntries, shardEntries...)
	}

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

	return Entry{}, NewKind("leaderboard.treap.rank", ErrNotFound)
}

// TopN returns the global top N by merging shard prefixes.
// Strategy: collect top-N from each shard (bounded by N), then k-way merge.
func (s *TreapStore) TopN(ctx context.Context, n int) ([]Entry, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start).Milliseconds()
		metrics.RecordRepositoryQueryLatency(float64(latency))
	}()

	if n < 1 {
		metrics.RecordErrorByComponent("repository", "invalid_limit")
		return nil, NewKind("leaderboard.treap.topn", ErrInvalidLimit)
	}
	// Collect per-shard prefixes.
	type shardSlice struct {
		entries []Entry
		idx     int
	}
	per := make([]shardSlice, len(s.shards))
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.RLock()
		out := make([]Entry, 0, n)
		collectTopN(sh.root, n, sh.byID, &out)
		sh.mu.RUnlock()
		per[i] = shardSlice{entries: out, idx: 0}
	}

	// k-way merge using a small heap of size S (shards).
	// We implement a simple linear select since shard count is small; if it grows,
	// replace with a binary heap for O(log S) per selection.
	out := make([]Entry, 0, n)
	for len(out) < n {
		bestIdx := -1
		var best Entry
		for i := range per {
			sl := &per[i]
			if sl.idx >= len(sl.entries) {
				continue
			}
			cand := sl.entries[sl.idx]
			if bestIdx == -1 || less(scoreFP(toFixedPoint(cand.Score)), cand.TalentID, scoreFP(toFixedPoint(best.Score)), best.TalentID) {
				bestIdx = i
				best = cand
			}
		}
		if bestIdx == -1 {
			break
		} // all exhausted
		out = append(out, best)
		per[bestIdx].idx++
	}
	// Assign ranks with proper tie handling
	assignRanksWithTies(out)
	return out, nil
}

// Count returns the total number of talents across shards.
func (s *TreapStore) Count(ctx context.Context) int {
	total := 0
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.RLock()
		total += len(sh.byID)
		sh.mu.RUnlock()
	}
	return total
}

// publishSnapshot rebuilds and publishes a new snapshot for the shard
func (sh *shard) publishSnapshot(topCacheSize int) {
	// Build Top-M cache for fast dashboard queries
	topCache := make([]Entry, 0, topCacheSize)
	collectTopN(sh.root, topCacheSize, sh.byID, &topCache)

	// Build full rank and score maps
	rankByTalent := make(map[string]int, len(sh.byID))
	scoreByTalent := make(map[string]float64, len(sh.byID))

	// Collect all entries to compute global ranks
	allEntries := make([]Entry, 0, len(sh.byID))
	collectAll(sh.root, sh.byID, &allEntries)

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

	sh.snapshot.Store(snapshot)
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
	totalRecords := 0
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.RLock()
		recordCount := len(sh.byID)
		sh.mu.RUnlock()

		totalRecords += recordCount
		shardID := fmt.Sprintf("shard_%d", i)

		// Update per-shard metrics
		metrics.UpdateRepositoryRecordsPerShard(shardID, recordCount)

		// Calculate shard utilization (assuming a reasonable capacity per shard)
		// This is a simplified calculation - in practice, you might want to track actual capacity
		estimatedCapacity := 10000 // This could be configurable
		utilization := float64(recordCount) / float64(estimatedCapacity)
		if utilization > 1.0 {
			utilization = 1.0
		}
		metrics.UpdateRepositoryShardUtilization(shardID, utilization)
	}

	// Update total records metric
	metrics.UpdateRepositoryRecordsTotal(totalRecords)
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
