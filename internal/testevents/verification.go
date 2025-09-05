package testevents

import (
	"context"
	"fmt"
	"log"
	"sort"
)

// verifyResults verifies the consistency of rankings and leaderboard.
func verifyResults(ctx context.Context, config *Config, rankings, leaderboard []Entry, stats *Stats) error {
	log.Println("🔍 Verifying results...")

	if len(rankings) == 0 {
		return fmt.Errorf("no rankings to verify")
	}

	// Sort rankings by score (descending) to get top performers
	sortedRankings := make([]Entry, len(rankings))
	copy(sortedRankings, rankings)
	sort.Slice(sortedRankings, func(i, j int) bool {
		return sortedRankings[i].Score > sortedRankings[j].Score
	})

	// Verify leaderboard consistency if we have leaderboard data
	if len(leaderboard) > 0 {
		if err := verifyLeaderboardConsistency(sortedRankings, leaderboard); err != nil {
			log.Printf("⚠️  Leaderboard consistency warning: %v", err)
		} else {
			log.Println("✅ Leaderboard consistency verified")
		}
	}

	// Display top performers
	displayTopPerformers(sortedRankings, leaderboard, config.Verbose)

	log.Println("✅ Result verification completed")
	return nil
}

// verifyLeaderboardConsistency checks if leaderboard matches top rankings.
func verifyLeaderboardConsistency(sortedRankings, leaderboard []Entry) error {
	if len(leaderboard) == 0 {
		return fmt.Errorf("empty leaderboard")
	}

	// Check if top entry in leaderboard matches highest ranked talent
	topRanking := sortedRankings[0]
	topLeaderboard := leaderboard[0]

	if topRanking.TalentID != topLeaderboard.TalentID {
		return fmt.Errorf("top leaderboard entry (%s) does not match top ranked talent (%s)",
			topLeaderboard.TalentID, topRanking.TalentID)
	}

	if topRanking.Score != topLeaderboard.Score {
		return fmt.Errorf("top leaderboard score (%.3f) does not match top ranked score (%.3f)",
			topLeaderboard.Score, topRanking.Score)
	}

	// Check if leaderboard is properly sorted
	for i := 1; i < len(leaderboard); i++ {
		if leaderboard[i].Score > leaderboard[i-1].Score {
			return fmt.Errorf("leaderboard not properly sorted: entry %d has higher score than entry %d",
				i, i-1)
		}
	}

	return nil
}

// displayTopPerformers shows the top performers from rankings and leaderboard.
func displayTopPerformers(sortedRankings, leaderboard []Entry, verbose bool) {
	topN := 10
	if len(sortedRankings) < topN {
		topN = len(sortedRankings)
	}

	log.Printf("🏆 Top %d performers from rankings:", topN)
	for i := 0; i < topN; i++ {
		entry := sortedRankings[i]
		log.Printf("   %d. %s - Score: %.3f", i+1, entry.TalentID, entry.Score)
	}

	if len(leaderboard) > 0 {
		leaderboardTopN := topN
		if len(leaderboard) < leaderboardTopN {
			leaderboardTopN = len(leaderboard)
		}

		log.Printf("🥇 Top %d performers from leaderboard:", leaderboardTopN)
		for i := 0; i < leaderboardTopN; i++ {
			entry := leaderboard[i]
			log.Printf("   %d. %s - Score: %.3f", i+1, entry.TalentID, entry.Score)
		}
	}

	if verbose {
		// Show some statistics
		if len(sortedRankings) > 0 {
			avgScore := calculateAverageScore(sortedRankings)
			maxScore := sortedRankings[0].Score
			minScore := sortedRankings[len(sortedRankings)-1].Score

			log.Printf(`📊 Score statistics:
   Average: %.3f
   Maximum: %.3f
   Minimum: %.3f
`, avgScore, maxScore, minScore)
		}
	}
}

// calculateAverageScore calculates the average score from rankings.
func calculateAverageScore(rankings []Entry) float64 {
	if len(rankings) == 0 {
		return 0
	}

	sum := 0.0
	for _, entry := range rankings {
		sum += entry.Score
	}

	return sum / float64(len(rankings))
}
