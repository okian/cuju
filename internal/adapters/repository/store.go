// Package repository defines the ranking store interface and errors.
package repository

import "context"

// Entry represents a leaderboard row.
type Entry struct {
	Rank      int
	TalentID  string
	Score     float64
	EventID   string
	Skill     string
	RawMetric float64
}

// Store provides read/write access to the ranking state.
type Store interface {
	// UpdateBest sets a new best score for talent if higher than the existing one.
	// Returns true if the store updated the score, false otherwise.
	UpdateBest(ctx context.Context, talentID string, score float64) (bool, error)
	// UpdateBestWithMeta sets a new best score and stores associated metadata when improved.
	UpdateBestWithMeta(ctx context.Context, talentID string, score float64, eventID string, skill string, rawMetric float64) (bool, error)

	// Rank returns the current rank and score for a talent.
	// Returns ErrNotFound if the talent is unknown.
	Rank(ctx context.Context, talentID string) (Entry, error)

	// TopN returns the top-N entries ordered by score desc.
	TopN(ctx context.Context, n int) ([]Entry, error)

	// Count returns the number of talents tracked in the leaderboard.
	Count(ctx context.Context) int
}
