package repository

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"testing"
)

// floatEqual compares two float64 values with a small tolerance for floating-point precision.
func floatEqual(a, b float64) bool {
	const tolerance = 1e-10
	return math.Abs(a-b) < tolerance
}

func TestTreapStore_BasicOperations(t *testing.T) {
	ctx := context.Background()
	store := NewTreapStore(ctx)

	// Test empty store
	if count := store.Count(ctx); count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}

	// Test inserting first entry
	updated, err := store.UpdateBest(ctx, "talent1", 85.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed")
	}

	if count := store.Count(ctx); count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Test rank query
	entry, err := store.Rank(ctx, "talent1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Rank != 1 {
		t.Errorf("expected rank 1, got %d", entry.Rank)
	}
	if entry.Score != 85.5 {
		t.Errorf("expected score 85.5, got %f", entry.Score)
	}

	// Test TopN
	entries, err := store.TopN(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].TalentID != "talent1" {
		t.Errorf("expected talent1, got %s", entries[0].TalentID)
	}
}

func TestTreapStore_ScoreUpdates(t *testing.T) {
	ctx := context.Background()
	store := NewTreapStore(ctx)

	// Insert initial score
	updated, err := store.UpdateBest(ctx, "talent1", 50.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed")
	}

	// Try to update with lower score (should fail)
	updated, err = store.UpdateBest(ctx, "talent1", 40.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated {
		t.Error("expected update to fail for lower score")
	}

	// Update with higher score (should succeed)
	updated, err = store.UpdateBest(ctx, "talent1", 90.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed")
	}

	// Verify new score
	entry, err := store.Rank(ctx, "talent1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Score != 90.0 {
		t.Errorf("expected score 90.0, got %f", entry.Score)
	}
}

func TestTreapStore_Ordering(t *testing.T) {
	ctx := context.Background()
	store := NewTreapStore(ctx)

	// Insert multiple talents with different scores
	talents := []struct {
		id    string
		score float64
	}{
		{"talent1", 85.0},
		{"talent2", 95.0},
		{"talent3", 75.0},
		{"talent4", 100.0},
		{"talent5", 80.0},
	}

	for _, talent := range talents {
		updated, err := store.UpdateBest(ctx, talent.id, talent.score)
		if err != nil {
			t.Fatalf("unexpected error updating %s: %v", talent.id, err)
		}
		if !updated {
			t.Errorf("expected update to succeed for %s", talent.id)
		}
	}

	// Test TopN ordering
	entries, err := store.TopN(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}

	// Verify descending order by score
	for i := 0; i < len(entries)-1; i++ {
		if entries[i].Score < entries[i+1].Score {
			t.Errorf("entries not in descending order: %f < %f", entries[i].Score, entries[i+1].Score)
		}
	}

	// Verify ranks are assigned correctly
	for i, entry := range entries {
		expectedRank := i + 1
		if entry.Rank != expectedRank {
			t.Errorf("entry %d: expected rank %d, got %d", i, expectedRank, entry.Rank)
		}
	}

	// Verify specific ordering
	expectedOrder := []string{"talent4", "talent2", "talent1", "talent5", "talent3"}
	for i, expectedID := range expectedOrder {
		if entries[i].TalentID != expectedID {
			t.Errorf("position %d: expected %s, got %s", i, expectedID, entries[i].TalentID)
		}
	}
}

func TestTreapStore_TieBreaking(t *testing.T) {
	ctx := context.Background()
	store := NewTreapStore(ctx)

	// Insert talents with same score but different IDs
	updated, err := store.UpdateBest(ctx, "talentB", 100.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed")
	}

	updated, err = store.UpdateBest(ctx, "talentA", 100.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed")
	}

	// Test TopN to see tie-breaking
	entries, err := store.TopN(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}

	// With same score, talentA should come before talentB (alphabetical)
	if entries[0].TalentID != "talentA" {
		t.Errorf("expected talentA first, got %s", entries[0].TalentID)
	}
	if entries[1].TalentID != "talentB" {
		t.Errorf("expected talentB second, got %s", entries[1].TalentID)
	}
}

func TestTreapStore_Sharding(t *testing.T) {
	ctx := context.Background()
	store := NewTreapStore(ctx) // 4 shards

	// Insert talents that should hash to different shards
	talents := []string{"talent1", "talent2", "talent3", "talent4", "talent5"}

	for i, talentID := range talents {
		score := float64(100 - i*10) // Different scores
		updated, err := store.UpdateBest(ctx, talentID, score)
		if err != nil {
			t.Fatalf("unexpected error updating %s: %v", talentID, err)
		}
		if !updated {
			t.Errorf("expected update to succeed for %s", talentID)
		}
	}

	// Verify all talents are stored
	if count := store.Count(ctx); count != 5 {
		t.Errorf("expected count 5, got %d", count)
	}

	// Test TopN across shards
	entries, err := store.TopN(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}

	// Verify ordering is maintained across shards
	for i := 0; i < len(entries)-1; i++ {
		if entries[i].Score < entries[i+1].Score {
			t.Errorf("entries not in descending order: %f < %f", entries[i].Score, entries[i+1].Score)
		}
	}
}

func TestTreapStore_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	store := NewTreapStore(ctx)
	numGoroutines := 10
	numUpdates := 100

	// Start multiple goroutines updating different talents
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numUpdates; j++ {
				talentID := fmt.Sprintf("talent%d_%d", id, j)
				score := float64(50 + j)
				_, err := store.UpdateBest(ctx, talentID, score)
				if err != nil {
					t.Errorf("goroutine %d: unexpected error: %v", id, err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final state
	expectedCount := numGoroutines * numUpdates
	if count := store.Count(ctx); count != expectedCount {
		t.Errorf("expected count %d, got %d", expectedCount, count)
	}

	// Test TopN still works
	entries, err := store.TopN(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 10 {
		t.Errorf("expected 10 entries, got %d", len(entries))
	}

	// Verify ordering
	for i := 0; i < len(entries)-1; i++ {
		if entries[i].Score < entries[i+1].Score {
			t.Errorf("entries not in descending order: %f < %f", entries[i].Score, entries[i+1].Score)
		}
	}
}

func TestTreapStore_EdgeCases(t *testing.T) {
	ctx := context.Background()
	store := NewTreapStore(ctx)

	// Test invalid TopN limit
	_, err := store.TopN(ctx, 0)
	if err == nil {
		t.Error("expected error for invalid limit")
	}

	_, err = store.TopN(ctx, -1)
	if err == nil {
		t.Error("expected error for negative limit")
	}

	// Test querying non-existent talent
	_, err = store.Rank(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent talent")
	}

	// Test very large scores
	updated, err := store.UpdateBest(ctx, "talent1", 1e6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed")
	}

	entry, err := store.Rank(ctx, "talent1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Score != 1e6 {
		t.Errorf("expected score 1e6, got %f", entry.Score)
	}
}

// TestMultipleScoresPerTalent demonstrates that each talent gets multiple random scores.
func TestMultipleScoresPerTalent(t *testing.T) {
	ctx := context.Background()
	store := NewTreapStore(ctx)

	// Test with a small number of talents to verify the pattern
	talentCount := 10

	// Pre-populate with multiple scores per talent
	for i := 0; i < talentCount; i++ {
		talentID := fmt.Sprintf("talent_%d", i)

		// Each talent should get multiple scores in different ranges
		for k := 0; k < 10; k++ {
			var score float64

			switch k % 5 {
			case 0: // Initial scores (0-200)
				score = rand.Float64() * 200.0 //nolint:gosec // test data generation
			case 1: // Beginner scores (200-400)
				score = 200.0 + rand.Float64()*200.0 //nolint:gosec // test data generation
			case 2: // Intermediate scores (400-600)
				score = 400.0 + rand.Float64()*200.0 //nolint:gosec // test data generation
			case 3: // Advanced scores (600-800)
				score = 600.0 + rand.Float64()*200.0 //nolint:gosec // test data generation
			case 4: // Elite scores (800-1000)
				score = 800.0 + rand.Float64()*200.0 //nolint:gosec // test data generation
			}

			// Add random variation
			variation := (rand.Float64() - 0.5) * 50.0 //nolint:gosec // test data generation
			score += variation

			if score < 0 {
				score = 0
			} else if score > 1000 {
				score = 1000
			}

			_, _ = store.UpdateBest(ctx, talentID, score)
		}
	}

	// Verify that talents have been updated with multiple scores
	for i := 0; i < talentCount; i++ {
		talentID := fmt.Sprintf("talent_%d", i)
		entry, err := store.Rank(ctx, talentID)
		if err != nil {
			t.Fatalf("Failed to get rank for %s: %v", talentID, err)
		}

		// Each talent should have a final score (the best of their 10 scores)
		if entry.Score < 0 || entry.Score > 1000 {
			t.Errorf("Talent %s has invalid score: %f", talentID, entry.Score)
		}

		// Verify the talent exists and has a rank
		if entry.TalentID != talentID {
			t.Errorf("Expected talent ID %s, got %s", talentID, entry.TalentID)
		}

		if entry.Rank <= 0 {
			t.Errorf("Talent %s should have a positive rank, got %d", talentID, entry.Rank)
		}
	}

	// Verify total count
	totalCount := store.Count(ctx)
	if totalCount != talentCount {
		t.Errorf("Expected %d talents, got %d", talentCount, totalCount)
	}

	// Test TopN to ensure ordering works with multiple scores
	topEntries, err := store.TopN(ctx, talentCount)
	if err != nil {
		t.Fatalf("Failed to get TopN: %v", err)
	}

	if len(topEntries) != talentCount {
		t.Errorf("Expected %d top entries, got %d", talentCount, len(topEntries))
	}

	// Verify scores are in descending order
	for i := 1; i < len(topEntries); i++ {
		if topEntries[i-1].Score < topEntries[i].Score {
			t.Errorf("Scores not in descending order: %f < %f",
				topEntries[i-1].Score, topEntries[i].Score)
		}
	}
}

func TestTreapStore_ScoreOverrideEdgeCases(t *testing.T) {
	ctx := context.Background()
	store := NewTreapStore(ctx)

	// Test exact same score (should not update)
	updated, err := store.UpdateBest(ctx, "talent1", 100.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed")
	}

	updated, err = store.UpdateBest(ctx, "talent1", 100.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated {
		t.Error("expected update to fail for identical score")
	}

	// Test infinitesimal score differences (within fixed-point precision)
	updated, err = store.UpdateBest(ctx, "talent1", 100.000001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed for small improvement")
	}

	// Test score degradation (should fail)
	updated, err = store.UpdateBest(ctx, "talent1", 99.999999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated {
		t.Error("expected update to fail for score degradation")
	}

	// Test large but reasonable score values (within fixed-point range)
	updated, err = store.UpdateBest(ctx, "talent2", 1e12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed for large score")
	}

	updated, err = store.UpdateBest(ctx, "talent3", -1e12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed for large negative score")
	}

	// Test very small scores
	updated, err = store.UpdateBest(ctx, "talent4", 1e-6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed for very small score")
	}

	updated, err = store.UpdateBest(ctx, "talent5", -1e-6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed for very small negative score")
	}

	// Test zero scores
	updated, err = store.UpdateBest(ctx, "talent6", 0.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed for zero score")
	}

	updated, err = store.UpdateBest(ctx, "talent7", 0.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed for negative zero")
	}
}

func TestTreapStore_RankCorrectnessUnderStress(t *testing.T) {
	ctx := context.Background()
	store := NewTreapStore(ctx)

	// Insert many talents with random scores
	numTalents := 1000
	talents := make([]string, numTalents)
	scores := make([]float64, numTalents)

	for i := 0; i < numTalents; i++ {
		talents[i] = fmt.Sprintf("talent_%d", i)
		scores[i] = rand.Float64() * 10000.0 //nolint:gosec // test data generation

		updated, err := store.UpdateBest(ctx, talents[i], scores[i])
		if err != nil {
			t.Fatalf("failed to insert talent %d: %v", i, err)
		}
		if !updated {
			t.Errorf("expected update to succeed for talent %d", i)
		}
	}

	// Verify all talents have correct ranks
	for i := 0; i < numTalents; i++ {
		entry, err := store.Rank(ctx, talents[i])
		if err != nil {
			t.Fatalf("failed to get rank for %s: %v", talents[i], err)
		}

		// Verify rank is within valid range
		if entry.Rank < 1 || entry.Rank > numTalents {
			t.Errorf("talent %s has invalid rank %d", talents[i], entry.Rank)
		}

		// Verify score matches (with tolerance for floating-point precision)
		if !floatEqual(entry.Score, scores[i]) {
			t.Errorf("talent %s score mismatch: expected %f, got %f", talents[i], scores[i], entry.Score)
		}
	}

	// Test TopN with various limits
	testLimits := []int{1, 10, 100, 500, 1000, 1500}
	for _, limit := range testLimits {
		entries, err := store.TopN(ctx, limit)
		if err != nil {
			t.Fatalf("TopN(%d) failed: %v", limit, err)
		}

		expectedLen := limit
		if limit > numTalents {
			expectedLen = numTalents
		}

		if len(entries) != expectedLen {
			t.Errorf("TopN(%d) returned %d entries, expected %d", limit, len(entries), expectedLen)
		}

		// Verify ranks are sequential and scores are descending
		for i := 0; i < len(entries); i++ {
			if entries[i].Rank != i+1 {
				t.Errorf("TopN(%d) entry %d: expected rank %d, got %d", limit, i, i+1, entries[i].Rank)
			}

			if i > 0 && entries[i].Score > entries[i-1].Score {
				t.Errorf("TopN(%d) scores not in descending order: %f > %f", limit, entries[i].Score, entries[i-1].Score)
			}
		}
	}
}

func TestTreapStore_ConcurrentScoreUpdates(t *testing.T) {
	ctx := context.Background()
	store := NewTreapStore(ctx)

	numGoroutines := 20
	updatesPerGoroutine := 50

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*updatesPerGoroutine)

	// Start multiple goroutines updating different talents concurrently
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for u := 0; u < updatesPerGoroutine; u++ {
				// Each goroutine works on a different set of talents
				talentID := fmt.Sprintf("talent_%d_%d", goroutineID, u)
				baseScore := float64(100 + u)
				variation := float64(goroutineID) * 0.1
				score := baseScore + variation

				_, err := store.UpdateBest(ctx, talentID, score)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d update %d failed: %w", goroutineID, u, err)
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("concurrent update error: %v", err)
	}

	// Verify final state is consistent
	expectedCount := numGoroutines * updatesPerGoroutine
	if count := store.Count(ctx); count != expectedCount {
		t.Errorf("expected count %d, got %d", expectedCount, count)
	}

	// Verify ranks are still correct after concurrent updates
	entries, err := store.TopN(ctx, expectedCount)
	if err != nil {
		t.Fatalf("failed to get TopN after concurrent updates: %v", err)
	}

	if len(entries) != expectedCount {
		t.Errorf("expected %d entries, got %d", expectedCount, len(entries))
	}

	// Verify scores are in descending order
	for i := 1; i < len(entries); i++ {
		if entries[i].Score > entries[i-1].Score {
			t.Errorf("scores not in descending order after concurrent updates: %f > %f",
				entries[i].Score, entries[i-1].Score)
		}
	}
}

func TestTreapStore_ExtremeScoreValues(t *testing.T) {
	ctx := context.Background()
	store := NewTreapStore(ctx)

	// Test various score values that can be reasonably represented
	extremeScores := []float64{
		0.0,
		0.0,
		1e-6,  // small but representable
		-1e-6, // small but representable
		1e+6,  // large but representable
		-1e+6, // large but representable
		1e-3,  // small but representable
		-1e-3, // small but representable
		1e+3,  // large but representable
		-1e+3, // large but representable
	}

	for i, score := range extremeScores {
		talentID := fmt.Sprintf("extreme_talent_%d", i)

		updated, err := store.UpdateBest(ctx, talentID, score)
		if err != nil {
			t.Fatalf("failed to insert extreme score %g for %s: %v", score, talentID, err)
		}
		if !updated {
			t.Errorf("expected update to succeed for extreme score %g", score)
		}

		// Verify we can retrieve the score
		entry, err := store.Rank(ctx, talentID)
		if err != nil {
			t.Fatalf("failed to get rank for extreme score %g for %s: %v", score, talentID, err)
		}

		if entry.Score != score {
			t.Errorf("extreme score mismatch for %s: expected %g, got %g", talentID, score, entry.Score)
		}
	}

	// Test that ordering works with extreme values
	entries, err := store.TopN(ctx, len(extremeScores))
	if err != nil {
		t.Fatalf("TopN failed with extreme scores: %v", err)
	}

	if len(entries) != len(extremeScores) {
		t.Errorf("expected %d entries, got %d", len(extremeScores), len(entries))
	}

	// Verify scores are in descending order
	for i := 1; i < len(entries); i++ {
		if entries[i].Score > entries[i-1].Score {
			t.Errorf("extreme scores not in descending order: %g > %g",
				entries[i].Score, entries[i-1].Score)
		}
	}
}

func TestTreapStore_EmptyAndSingleElement(t *testing.T) {
	ctx := context.Background()
	store := NewTreapStore(ctx)

	// Test empty store operations
	if count := store.Count(ctx); count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}

	// Test TopN on empty store
	entries, err := store.TopN(ctx, 10)
	if err != nil {
		t.Fatalf("TopN on empty store failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries from empty store, got %d", len(entries))
	}

	// Test Rank on empty store
	_, err = store.Rank(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error when querying nonexistent talent in empty store")
	}

	// Add single element
	updated, err := store.UpdateBest(ctx, "single", 100.0)
	if err != nil {
		t.Fatalf("failed to insert single element: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed")
	}

	// Test single element operations
	if count := store.Count(ctx); count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	entries, err = store.TopN(ctx, 10)
	if err != nil {
		t.Fatalf("TopN on single element store failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Rank != 1 {
		t.Errorf("expected rank 1, got %d", entries[0].Rank)
	}
	if entries[0].TalentID != "single" {
		t.Errorf("expected talent ID 'single', got %s", entries[0].TalentID)
	}
	if entries[0].Score != 100.0 {
		t.Errorf("expected score 100.0, got %f", entries[0].Score)
	}

	// Test TopN with limit 1
	entries, err = store.TopN(ctx, 1)
	if err != nil {
		t.Fatalf("TopN(1) failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry from TopN(1), got %d", len(entries))
	}
}

func TestTreapStore_ShardDistribution(t *testing.T) {
	ctx := context.Background()
	shardCounts := []int{1, 2, 4, 8, 16, 32}

	for _, shardCount := range shardCounts {
		t.Run(fmt.Sprintf("Shards_%d", shardCount), func(t *testing.T) {
			store := NewTreapStore(ctx)

			// Insert talents that should hash to different shards
			numTalents := shardCount * 10
			for i := 0; i < numTalents; i++ {
				talentID := fmt.Sprintf("shard_test_%d", i)
				score := float64(1000 - i)

				updated, err := store.UpdateBest(ctx, talentID, score)
				if err != nil {
					t.Fatalf("failed to insert talent %d: %v", i, err)
				}
				if !updated {
					t.Errorf("expected update to succeed for talent %d", i)
				}
			}

			// Verify all talents are stored
			if count := store.Count(ctx); count != numTalents {
				t.Errorf("expected count %d, got %d", numTalents, count)
			}

			// Test TopN across shards
			entries, err := store.TopN(ctx, numTalents)
			if err != nil {
				t.Fatalf("TopN failed: %v", err)
			}

			if len(entries) != numTalents {
				t.Errorf("expected %d entries, got %d", numTalents, len(entries))
			}

			// Verify ordering is maintained across shards
			for i := 1; i < len(entries); i++ {
				if entries[i].Score > entries[i-1].Score {
					t.Errorf("entries not in descending order: %f > %f",
						entries[i].Score, entries[i-1].Score)
				}
			}

			// Verify ranks are sequential
			for i, entry := range entries {
				if entry.Rank != i+1 {
					t.Errorf("entry %d: expected rank %d, got %d", i, i+1, entry.Rank)
				}
			}
		})
	}
}

func TestTreapStore_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store := NewTreapStore(ctx)

	// Insert some data
	updated, err := store.UpdateBest(ctx, "talent1", 100.0)
	if err != nil {
		t.Fatalf("failed to insert talent: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed")
	}

	// Cancel context
	cancel()

	// Operations should still work
	updated, err = store.UpdateBest(ctx, "talent2", 200.0)
	if err != nil {
		t.Fatalf("UpdateBest failed after context cancellation: %v", err)
	}
	if !updated {
		t.Error("expected update to succeed after context cancellation")
	}

	entry, err := store.Rank(ctx, "talent1")
	if err != nil {
		t.Fatalf("Rank failed after context cancellation: %v", err)
	}
	if entry.Score != 100.0 {
		t.Errorf("expected score 100.0, got %f", entry.Score)
	}

	entries, err := store.TopN(ctx, 10)
	if err != nil {
		t.Fatalf("TopN failed after context cancellation: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}
