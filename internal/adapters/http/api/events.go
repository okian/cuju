// Package api declares HTTP contracts and route registration helpers.
package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/okian/cuju/internal/domain/dedupe"
)

// EventDependencies defines the interface for event processing dependencies
type EventDependencies interface {
	dedupe.Deduper
	Enqueue(ctx context.Context, e any) bool
}

// EventsHandler handles event requests
type EventsHandler struct {
	deps EventDependencies
}

// NewEventsHandler creates a new events handler
func NewEventsHandler(deps EventDependencies) *EventsHandler {
	return &EventsHandler{deps: deps}
}

// HandlePostEvent handles POST /events requests
func (h *EventsHandler) HandlePostEvent(w http.ResponseWriter, r *http.Request) {
	const op = "api.post_event"
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var req eventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", WrapKind(op, ErrBadRequest, err))
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", WrapKind(op, ErrBadRequest, err))
		return
	}

	// Idempotency check - mark as seen first
	if h.deps.SeenAndRecord(r.Context(), req.EventID) {
		writeJSON(w, http.StatusOK, ackResponse{Status: "duplicate", Duplicate: true})
		return
	}

	// Try to enqueue for async processing
	if ok := h.deps.Enqueue(r.Context(), req); !ok {
		// Rollback the "seen" status since enqueue failed
		h.deps.Unrecord(r.Context(), req.EventID)
		writeError(w, http.StatusTooManyRequests, "backpressure", NewKind(op, ErrBackpressure))
		return
	}
	writeJSON(w, http.StatusAccepted, ackResponse{Status: "accepted", Duplicate: false})
}
