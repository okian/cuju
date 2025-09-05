package repository

import "errors"

// Sentinel kinds for leaderboard errors.
var (
	ErrNotFound     = errors.New("talent not found")
	ErrInvalidLimit = errors.New("invalid leaderboard limit")
)
