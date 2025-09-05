// Package api declares HTTP contracts and route registration helpers.
package api

import (
	"context"
	"net/http"
	"strings"
)

// RankDependencies defines the interface for rank operations.
type RankDependencies interface {
	Rank(ctx context.Context, talentID string) (Entry, error)
}

// RankHandler handles rank requests.
type RankHandler struct {
	deps RankDependencies
}

// NewRankHandler creates a new rank handler.
func NewRankHandler(deps RankDependencies) *RankHandler {
	return &RankHandler{deps: deps}
}

// HandleGetRank handles GET /rank/{talent_id} requests.
func (h *RankHandler) HandleGetRank(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	// Extract path parameter after /rank/
	path := strings.TrimPrefix(r.URL.Path, "/rank/")
	if path == "" || strings.Contains(path, "/") {
		writeError(w, http.StatusBadRequest, "bad_request", ErrBadRequest)
		return
	}
	entry, err := h.deps.Rank(r.Context(), path)
	if err != nil {
		// If upstream exposes not-found, translate; otherwise 500
		if isNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", err)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}
