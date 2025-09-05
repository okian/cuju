package queue

import "errors"

// Sentinel kinds for worker errors.
var (
	ErrStopped = errors.New("worker stopped")
)
