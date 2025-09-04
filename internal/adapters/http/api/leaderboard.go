// Package api declares HTTP contracts and route registration helpers.
package api

import (
	"context"
	"net/http"
	"strconv"
)

// LeaderboardDependencies defines the interface for leaderboard operations
type LeaderboardDependencies interface {
	TopN(ctx context.Context, n int) ([]Entry, error)
}

// LeaderboardHandler handles leaderboard requests
type LeaderboardHandler struct {
	deps     LeaderboardDependencies
	maxLimit int
}

// NewLeaderboardHandler creates a new leaderboard handler
func NewLeaderboardHandler(deps LeaderboardDependencies, maxLimit int) *LeaderboardHandler {
	return &LeaderboardHandler{
		deps:     deps,
		maxLimit: maxLimit,
	}
}

// HandleGetLeaderboard handles GET /leaderboard?limit=N requests
func (h *LeaderboardHandler) HandleGetLeaderboard(w http.ResponseWriter, r *http.Request) {
	const op = "api.get_leaderboard"
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	limitStr := r.URL.Query().Get("limit")
	n, err := strconv.Atoi(limitStr)
	if err != nil || n < 1 {
		writeError(w, http.StatusBadRequest, "bad_request", NewKind(op, ErrBadRequest))
		return
	}
	if n > h.maxLimit {
		writeError(w, http.StatusBadRequest, "limit_exceeded", NewKind(op, ErrBadRequest))
		return
	}
	entries, err := h.deps.TopN(r.Context(), n)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", Wrap(op, err))
		return
	}
	writeJSON(w, http.StatusOK, entries)
}
