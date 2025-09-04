// Package types contains common types used across the application
package types

// Entry represents a leaderboard entry
type Entry struct {
	Rank     int     `json:"rank"`
	TalentID string  `json:"talent_id"`
	Score    float64 `json:"score"`
}
