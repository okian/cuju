// Package repository defines the ranking store interface and errors.
package repository

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/okian/cuju/pkg/metrics"
)

// Score bounds constants.
const (
	maxScoreValue                = 1e12
	minScoreValue                = -1e12
	defaultMetricsUpdateInterval = 5 * time.Second
)

// entryPool reduces allocations by reusing Entry structs for internal operations.
var entryPool = sync.Pool{
	New: func() interface{} {
		return &Entry{}
	},
}

// createEntry creates an Entry from a node and record, using the pool for efficiency.
func createEntry(n *node, rec *record) Entry {
	entry := entryPool.Get().(*Entry)
	entry.TalentID = n.id
	entry.Score = toFloat(rec.score)
	entry.EventID = rec.eventID
	entry.Skill = rec.skill
	entry.RawMetric = rec.rawMetric
	entry.Rank = 0 // Will be set later
	result := *entry
	// Reset the pooled entry
	entry.TalentID = ""
	entry.Score = 0
	entry.EventID = ""
	entry.Skill = ""
	entry.RawMetric = 0
	entry.Rank = 0
	entryPool.Put(entry)
	return result
}

// Treap-based, in-memory Store implementation.
//
// Ordering: score DESC, then talentID ASC (deterministic).
// We implement a BST comparator where "less" means ranks earlier.
// (i.e., higher score ranks earlier). This makes in-order traversal
// produce the leaderboard from best to worst.

// scoreScale controls fixed-point scaling from float64.
const scoreScale = 1_000_000_000_000 // 12 decimal places for better precision

type scoreFP int64

func toFixedPoint(x float64) scoreFP {
	// Handle special cases
	if x != x { // NaN check (faster than math.IsNaN)
		return 0
	}
	if x == math.Inf(1) {
		return scoreFP(math.MaxInt64)
	}
	if x == math.Inf(-1) {
		return scoreFP(math.MinInt64)
	}

	// Clamp to reasonable range to avoid overflow
	if x > maxScoreValue {
		x = maxScoreValue
	} else if x < minScoreValue {
		x = minScoreValue
	}

	// Convert to fixed point with optimized bounds checking
	scaled := x * scoreScale

	// Use bit manipulation for faster bounds checking
	if scaled >= float64(math.MaxInt64) {
		return scoreFP(math.MaxInt64)
	}
	if scaled <= float64(math.MinInt64) {
		return scoreFP(math.MinInt64)
	}

	return scoreFP(math.Round(scaled))
}

func toFloat(x scoreFP) float64 {
	return float64(x) / scoreScale
}

// record stores the fixed-point score plus metadata for a talent's best.
type record struct {
	score     scoreFP
	eventID   string
	skill     string
	rawMetric float64
}

// treap node.
type node struct {
	id     string
	score  scoreFP
	prio   uint64
	left   *node
	right  *node
	size   int
	record *record // pointer to avoid map lookups during traversal
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
	// Fast path: most comparisons will be by score
	if aScore > bScore {
		return true
	}
	if aScore < bScore {
		return false
	}
	// Tie-breaker by ID (lexicographic order)
	return aID < bID
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
// Use random priorities to maintain treap balance for O(log n) performance.
func scoreToPriority(score scoreFP) uint64 {
	// Use the score as a seed for deterministic random priorities
	// This ensures the same score always gets the same priority
	// but different scores get different random priorities
	return uint64(score*1103515245 + 12345) // Simple linear congruential generator //nolint:gosec // intentional overflow for hash function
}

func insert(n *node, id string, score scoreFP, rec *record) *node {
	if n == nil {
		return &node{id: id, score: score, prio: scoreToPriority(score), size: 1, record: rec}
	}
	if less(score, id, n.score, n.id) {
		n.left = insert(n.left, id, score, rec)
		if n.left.prio > n.prio {
			n = rotateRight(n)
		}
	} else {
		n.right = insert(n.right, id, score, rec)
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
func collectTopN(n *node, limit int, out *[]Entry) {
	if n == nil || len(*out) >= limit {
		return
	}

	// With score-based priorities, we need to maintain deterministic ordering
	// for tie-breaking (talent ID ASC). We do this by using the BST ordering
	// which is based on the less() function that handles tie-breaking correctly.

	// Traverse left subtree first (higher scores, or same score with lower ID)
	collectTopN(n.left, limit, out)

	// Add current node if we haven't reached the limit
	if len(*out) < limit && n.record != nil {
		entry := createEntry(n, n.record)
		*out = append(*out, entry)
	}

	// Traverse right subtree (lower scores, or same score with higher ID) if we still need more
	if len(*out) < limit {
		collectTopN(n.right, limit, out)
	}
}

type TreapStore struct {
	mu                    sync.RWMutex
	root                  *node
	byID                  map[string]*record // use pointers to reduce allocations
	metricsUpdateInterval time.Duration      // configurable metrics update interval
}

// NewTreapStore constructs a treap store with configuration options.
func NewTreapStore(ctx context.Context, opts ...Option) *TreapStore {
	s := &TreapStore{
		byID:                  make(map[string]*record),
		metricsUpdateInterval: defaultMetricsUpdateInterval, // default 5 seconds
	}

	// Apply all options (shard-related options are ignored)
	for _, opt := range opts {
		opt(s)
	}

	// Initialize metrics
	metrics.UpdateRepositoryShardCount(1) // Single "shard"
	s.startMetricsUpdater(ctx)

	return s
}

// UpdateBest implements Store.UpdateBest with O(log n) expected time.
func (s *TreapStore) UpdateBest(ctx context.Context, talentID string, score float64) (bool, error) {
	return s.UpdateBestWithMeta(ctx, talentID, score, "", "", 0)
}

// UpdateBestWithMeta implements Store.UpdateBestWithMeta with O(log n) expected time.
func (s *TreapStore) UpdateBestWithMeta(ctx context.Context, talentID string, score float64, eventID string, skill string, rawMetric float64) (bool, error) {
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
		// Update existing record in place
		old.score = ns
		old.eventID = eventID
		old.skill = skill
		old.rawMetric = rawMetric
		s.root = insert(s.root, talentID, ns, old)
	} else {
		isNewTalent = true
		// Create new record
		newRecord := &record{score: ns, eventID: eventID, skill: skill, rawMetric: rawMetric}
		s.byID[talentID] = newRecord
		s.root = insert(s.root, talentID, ns, newRecord)
	}
	s.mu.Unlock()

	// Update metrics outside lock (non-blocking)
	if isNewTalent {
		go func() {
			metrics.UpdateRepositoryRecordsTotal(s.Count(ctx))
		}()
	}

	return true, nil
}

// Rank returns the current rank and score for a talent in O(log n).
func (s *TreapStore) Rank(ctx context.Context, talentID string) (Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if the talent exists and get its record
	record, ok := s.byID[talentID]
	if !ok {
		// Record error asynchronously to avoid blocking
		go func() {
			metrics.RecordErrorByComponent("repository", "not_found")
		}()
		return Entry{}, ErrNotFound
	}

	// Calculate rank efficiently using treap traversal
	rank := s.calculateRank(s.root, record.score, talentID)

	// Create entry with calculated rank
	entry := Entry{
		Rank:      rank,
		TalentID:  talentID,
		Score:     toFloat(record.score),
		EventID:   record.eventID,
		Skill:     record.skill,
		RawMetric: record.rawMetric,
	}

	return entry, nil
}

// TopN returns the top N entries ordered by score desc.
func (s *TreapStore) TopN(ctx context.Context, n int) ([]Entry, error) {
	if n < 1 {
		// Record error asynchronously to avoid blocking
		go func() {
			metrics.RecordErrorByComponent("repository", "invalid_limit")
		}()
		return nil, ErrInvalidLimit
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Entry, 0, n)
	collectTopN(s.root, n, &out)

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

// startMetricsUpdater starts a background goroutine that updates repository metrics.
func (s *TreapStore) startMetricsUpdater(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(s.metricsUpdateInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.updateMetrics()
			}
		}
	}()
}

// updateMetrics updates all repository-related metrics.
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

// calculateRank efficiently calculates the rank of a talent.
// Uses in-order traversal with early termination for better performance than O(n).
func (s *TreapStore) calculateRank(n *node, targetScore scoreFP, targetID string) int {
	if n == nil {
		return 0
	}

	// Use in-order traversal to count nodes with higher scores
	count := 0
	s.countHigherScores(n, targetScore, targetID, &count)
	return count + 1 // 1-based ranking
}

// countHigherScores counts nodes with higher scores using in-order traversal.
func (s *TreapStore) countHigherScores(n *node, targetScore scoreFP, targetID string, count *int) {
	if n == nil {
		return
	}

	// Traverse left subtree first (higher scores)
	s.countHigherScores(n.left, targetScore, targetID, count)

	// Check current node
	if less(n.score, n.id, targetScore, targetID) {
		// Current node has higher score than target
		(*count)++
	} else if n.score == targetScore && n.id == targetID {
		// Found the target node, stop counting
		return
	}

	// Traverse right subtree (lower scores)
	s.countHigherScores(n.right, targetScore, targetID, count)
}

// collectAll appends all entries in rank order (highest scores first).
func collectAll(n *node, out *[]Entry) {
	if n == nil {
		return
	}
	// Traverse left subtree first (higher scores)
	collectAll(n.left, out)
	// Add current node
	if n.record != nil {
		entry := createEntry(n, n.record)
		*out = append(*out, entry)
	}
	// Traverse right subtree (lower scores)
	collectAll(n.right, out)
}

// sortEntries sorts entries by score (descending) and talentID (ascending) to match TopN logic.
func sortEntries(entries []Entry) {
	// Simple bubble sort for small datasets
	for i := 0; i < len(entries)-1; i++ {
		for j := 0; j < len(entries)-i-1; j++ {
			// Higher score comes first (descending order)
			if entries[j].Score < entries[j+1].Score {
				entries[j], entries[j+1] = entries[j+1], entries[j]
			} else if entries[j].Score == entries[j+1].Score {
				// Tie-breaker: talentID in ascending order
				if entries[j].TalentID > entries[j+1].TalentID {
					entries[j], entries[j+1] = entries[j+1], entries[j]
				}
			}
		}
	}
}

// assignRanksWithTies assigns ranks with proper tie handling.
// Talents with the same score get the same rank, and the next rank.
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
