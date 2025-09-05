// Package api declares HTTP contracts and route registration helpers.
package api

import (
	"encoding/json"
	"net/http"
)

// StatsProvider defines the interface for getting service statistics.
type StatsProvider interface {
	GetStats() map[string]interface{}
}

// StatsHandler handles stats requests.
type StatsHandler struct {
	statsProvider StatsProvider
}

// NewStatsHandler creates a new stats handler.
func NewStatsHandler(statsProvider StatsProvider) *StatsHandler {
	return &StatsHandler{statsProvider: statsProvider}
}

// HandleStats handles GET /stats requests.
func (h *StatsHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	stats := h.statsProvider.GetStats()
	_ = json.NewEncoder(w).Encode(stats)
}
