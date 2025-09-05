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
	defaultSnapshotInterval      = 1 * time.Second // refresh rank snapshot every second
)

// entryPool reduces allocations by reusing Entry structs for internal operations.
var entryPool = sync.Pool{ //nolint:gochecknoglobals // intentional global for performance optimization
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

// scoreToPriority converts a score and talent ID to a priority value.
// Use deterministic priorities to maintain treap balance for O(log n) performance
// while ensuring consistent ordering for tie-breaking.
func scoreToPriority(score scoreFP, talentID string) uint64 {
	// Use both score and talent ID to create deterministic priorities
	// This ensures consistent ordering even for users with the same score
	const (
		multiplier = scoreFP(1103515245) // LCG multiplier constant
		addend     = scoreFP(12345)      // LCG addend constant
	)

	// Create a hash from the talent ID
	const hashMultiplier = 31 // Standard hash multiplier
	idHash := scoreFP(0)
	for _, c := range talentID {
		idHash = idHash*hashMultiplier + scoreFP(c)
	}

	result := score*multiplier + addend + idHash

	// Handle potential overflow by using modulo to keep within uint64 range
	if result < 0 {
		result = -result
	}
	return uint64(result) //nolint:gosec // intentional conversion for hash function
}

func insert(n *node, id string, score scoreFP, rec *record) *node {
	if n == nil {
		return &node{id: id, score: score, prio: scoreToPriority(score, id), size: 1, record: rec}
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
	if score == n.score && id == n.id { //nolint:nestif // complex nested logic required for treap deletion
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

	// Snapshot-based ranking for fast and consistent rank queries
	rankSnapshot     map[string]int // talentID -> rank
	snapshotMutex    sync.RWMutex   // protects rankSnapshot
	lastSnapshotTime time.Time
	snapshotInterval time.Duration // how often to refresh the snapshot
}

// collectAllEntries appends all entries in rank order (highest scores first).
// This is used by the snapshot updater to ensure consistent ranking.
func (s *TreapStore) collectAllEntries(n *node, out *[]Entry) {
	if n == nil {
		return
	}

	// Traverse left subtree first (higher scores, or same score with lower ID)
	s.collectAllEntries(n.left, out)

	// Add current node
	if n.record != nil {
		entry := createEntry(n, n.record)
		*out = append(*out, entry)
	}

	// Traverse right subtree (lower scores, or same score with higher ID)
	s.collectAllEntries(n.right, out)
}

// calculateRank efficiently calculates the rank of a talent using treap traversal.
// Returns the 1-based rank (1 = highest score).
// This is O(log n) average case, O(n) worst case for degenerate trees.
func (s *TreapStore) calculateRank(n *node, talentID string, score float64) int {
	if n == nil {
		return 0
	}

	// Convert score to fixed point for comparison
	scoreFP := toFixedPoint(score)

	// Count how many nodes rank higher (have better scores or same score with lower ID)
	rank := 1 // Start at rank 1

	// Traverse the treap to count nodes that rank higher
	s.countHigherRanked(n, talentID, scoreFP, &rank)

	return rank
}

// countHigherRanked recursively counts nodes that rank higher than the target.
// Uses the same ordering logic as the treap: score DESC, then talentID ASC.
func (s *TreapStore) countHigherRanked(n *node, targetID string, targetScore scoreFP, rank *int) {
	if n == nil {
		return
	}

	// Compare current node with target using the same logic as the treap
	if less(n.score, n.id, targetScore, targetID) {
		// Current node ranks higher than target
		(*rank)++
		// Continue searching in both subtrees
		s.countHigherRanked(n.left, targetID, targetScore, rank)
		s.countHigherRanked(n.right, targetID, targetScore, rank)
	} else if less(targetScore, targetID, n.score, n.id) {
		// Target ranks higher than current node
		// Only search left subtree (higher scores)
		s.countHigherRanked(n.left, targetID, targetScore, rank)
	} else {
		// Same score and same ID - this is the target node itself, don't count it
		if n.id == targetID {
			// This is the target node, don't count it
			return
		}
		// Same score, different ID - need to check ID ordering
		if n.id < targetID {
			// Current node has same score but lower ID (ranks higher)
			(*rank)++
		}
		// Continue searching in both subtrees for same score
		s.countHigherRanked(n.left, targetID, targetScore, rank)
		s.countHigherRanked(n.right, targetID, targetScore, rank)
	}
}

// NewTreapStore constructs a treap store with configuration options.
func NewTreapStore(ctx context.Context, opts ...Option) *TreapStore {
	s := &TreapStore{
		byID:                  make(map[string]*record),
		metricsUpdateInterval: defaultMetricsUpdateInterval, // default 5 seconds
		rankSnapshot:          make(map[string]int),
		snapshotInterval:      defaultSnapshotInterval, // default 1 second
	}

	// Apply all options (shard-related options are ignored)
	for _, opt := range opts {
		opt(s)
	}

	// Initialize metrics
	metrics.UpdateRepositoryShardCount(1) // Single "shard"
	s.startMetricsUpdater(ctx)
	s.startSnapshotUpdater(ctx)

	return s
}

// UpdateBest implements Store.UpdateBest with O(log n) expected time.
func (s *TreapStore) UpdateBest(ctx context.Context, talentID string, score float64) (bool, error) {
	return s.UpdateBestWithMeta(ctx, talentID, score, "", "", 0)
}

// UpdateBestWithMeta implements Store.UpdateBestWithMeta with O(log n) expected time.
func (s *TreapStore) UpdateBestWithMeta(ctx context.Context, talentID string, score float64, eventID, skill string, rawMetric float64) (bool, error) {
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

// Rank returns the current rank and score for a talent.
// Uses snapshot-based ranking for O(1) performance and consistency.
func (s *TreapStore) Rank(ctx context.Context, talentID string) (Entry, error) {
	// First check if talent exists and get its record
	s.mu.RLock()
	record, ok := s.byID[talentID]
	s.mu.RUnlock()

	if !ok {
		// Record error asynchronously to avoid blocking
		go func() {
			metrics.RecordErrorByComponent("repository", "not_found")
		}()
		return Entry{}, ErrNotFound
	}

	// Get rank from snapshot (O(1) lookup)
	s.snapshotMutex.RLock()
	rank, exists := s.rankSnapshot[talentID]
	s.snapshotMutex.RUnlock()

	// If not in snapshot, fall back to calculation (shouldn't happen in normal operation)
	if !exists {
		s.mu.RLock()
		rank = s.calculateRank(s.root, talentID, toFloat(record.score))
		s.mu.RUnlock()
	}

	// Create entry with rank from snapshot
	return Entry{
		Rank:      rank,
		TalentID:  talentID,
		Score:     toFloat(record.score),
		EventID:   record.eventID,
		Skill:     record.skill,
		RawMetric: record.rawMetric,
	}, nil
}

// TopN returns the top N entries ordered by score desc.
// Uses snapshot-based ranking for consistency with Rank API.
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

	// Get top N entries without ranks
	out := make([]Entry, 0, n)
	collectTopN(s.root, n, &out)

	// Assign ranks from snapshot for consistency with Rank API
	s.snapshotMutex.RLock()
	for i := range out {
		if rank, exists := s.rankSnapshot[out[i].TalentID]; exists {
			out[i].Rank = rank
		} else {
			// Fallback: assign ranks with tie handling (shouldn't happen in normal operation)
			assignRanksWithTies(out)
			break
		}
	}
	s.snapshotMutex.RUnlock()

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

// startSnapshotUpdater starts a background goroutine that updates the rank snapshot.
func (s *TreapStore) startSnapshotUpdater(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(s.snapshotInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.updateRankSnapshot()
			}
		}
	}()
}

// updateRankSnapshot creates a fresh snapshot of all ranks.
// This runs in the background and provides consistent, fast rank lookups.
func (s *TreapStore) updateRankSnapshot() {
	s.mu.RLock()
	// Collect all entries in rank order
	allEntries := make([]Entry, 0, len(s.byID))
	s.collectAllEntries(s.root, &allEntries)
	s.mu.RUnlock()

	// Assign ranks with proper tie handling
	assignRanksWithTies(allEntries)

	// Create new snapshot
	newSnapshot := make(map[string]int, len(allEntries))
	for _, entry := range allEntries {
		newSnapshot[entry.TalentID] = entry.Rank
	}

	// Update snapshot atomically
	s.snapshotMutex.Lock()
	s.rankSnapshot = newSnapshot
	s.lastSnapshotTime = time.Now()
	s.snapshotMutex.Unlock()
}

// assignRanksWithTies assigns ranks with proper tie handling.
// Talents with the same score get the same rank, and the next rank
// skips the appropriate number of positions.
// This function handles cases where entries with the same score may not be consecutive.
func assignRanksWithTies(entries []Entry) {
	if len(entries) == 0 {
		return
	}

	// Group entries by score to handle non-consecutive entries with same score
	scoreGroups := make(map[float64][]int)
	for i, entry := range entries {
		scoreGroups[entry.Score] = append(scoreGroups[entry.Score], i)
	}

	// Sort scores in descending order
	scores := make([]float64, 0, len(scoreGroups))
	for score := range scoreGroups {
		scores = append(scores, score)
	}

	// Simple sort in descending order
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[i] < scores[j] {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Assign ranks
	currentRank := 1
	for _, score := range scores {
		indices := scoreGroups[score]
		// All entries with this score get the same rank
		for _, idx := range indices {
			entries[idx].Rank = currentRank
		}
		// Move to next rank, skipping the number of tied positions
		currentRank += len(indices)
	}
}
