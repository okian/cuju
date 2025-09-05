// Package api declares HTTP contracts and route registration helpers.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/okian/cuju/internal/domain/dedupe"
	"github.com/okian/cuju/internal/domain/types"
)

// Dependencies required by HTTP handlers. Using an interface bundle keeps
// the handler layer loosely coupled to implementations in other packages.
type Dependencies interface {
	dedupe.Deduper

	// Enqueue pushes an event for async processing. Returns false on backpressure.
	Enqueue(ctx context.Context, e any) bool

	// Read operations expose leaderboard data.
	TopN(ctx context.Context, n int) ([]Entry, error)
	Rank(ctx context.Context, talentID string) (Entry, error)
}

// Entry mirrors the read shape returned by leaderboard queries.
type Entry = types.Entry

// Server wires HTTP routes for the business API.
type Server struct {
	healthHandler      *HealthHandler
	statsHandler       *StatsHandler
	eventsHandler      *EventsHandler
	leaderboardHandler *LeaderboardHandler
	rankHandler        *RankHandler
	dashboardHandler   *dashboardHandler
}

// NewServer creates a new API server with all handlers.
func NewServer(deps Dependencies, statsProvider StatsProvider) *Server {
	return &Server{
		healthHandler:      NewHealthHandler(),
		statsHandler:       NewStatsHandler(statsProvider),
		eventsHandler:      NewEventsHandler(deps),
		leaderboardHandler: NewLeaderboardHandler(deps),
		rankHandler:        NewRankHandler(deps),
		dashboardHandler:   newdashboardHandler(),
	}
}

// Register attaches all HTTP routes to mux.
func (s *Server) Register(ctx context.Context, mux *http.ServeMux, deps Dependencies) {
	// Specific paths first (most specific to least specific)
	mux.HandleFunc("/healthz", MetricsMiddleware(s.healthHandler.HandleHealth, "healthz"))
	mux.HandleFunc("/dashboard", s.dashboardHandler.HandleDashboard)
	mux.HandleFunc("/stats", MetricsMiddleware(s.statsHandler.HandleStats, "stats"))
	mux.HandleFunc("/events", MetricsMiddleware(s.eventsHandler.HandlePostEvent, "events"))
	mux.HandleFunc("/leaderboard", MetricsMiddleware(s.leaderboardHandler.HandleGetLeaderboard, "leaderboard"))
	mux.HandleFunc("/rank/", MetricsMiddleware(s.rankHandler.HandleGetRank, "rank"))

}

// eventRequest mirrors the OpenAPI schema for POST /events.
type eventRequest struct {
	EventID   string  `json:"event_id"`
	TalentID  string  `json:"talent_id"`
	RawMetric float64 `json:"raw_metric"`
	Skill     string  `json:"skill"`
	TS        string  `json:"ts"`
}

func (e eventRequest) validate() error {
	switch {
	case strings.TrimSpace(e.EventID) == "":
		return errors.New("missing event_id")
	case strings.TrimSpace(e.TalentID) == "":
		return errors.New("missing talent_id")
	case strings.TrimSpace(e.Skill) == "":
		return errors.New("missing skill")
	case strings.TrimSpace(e.TS) == "":
		return errors.New("missing ts")
	}
	if _, err := time.Parse(time.RFC3339, e.TS); err != nil {
		return errors.New("invalid ts; must be RFC3339")
	}
	return nil
}

type ackResponse struct {
	Status    string `json:"status"`
	Duplicate bool   `json:"duplicate"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code string, err error) {
	msg := http.StatusText(status)
	if err != nil {
		msg = err.Error()
	}
	writeJSON(w, status, errorResponse{Code: code, Message: msg})
}

// isNotFound allows the API to translate upstream not-found errors to 404.
// This stays generic to avoid tight coupling with specific packages.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	// Best-effort string/unwrap check for a not-found condition.
	if errors.Is(err, errors.New("talent not found")) { // placeholder; real impl can specialize
		return true
	}
	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		return true
	}
	return false
}
