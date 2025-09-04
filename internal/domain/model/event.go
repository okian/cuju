// Package model contains domain models passed between layers.
package model

import "time"

// Event represents a performance event submitted by clients.
// Fields mirror the OpenAPI schema for /events.
type Event struct {
	EventID   string    // unique id for idempotency
	TalentID  string    // subject/talent identifier
	RawMetric float64   // raw metric value (normalized to float64)
	Skill     string    // skill category, e.g., "dribble", "shot"
	TS        time.Time // event timestamp
}

// TalentScore captures a talent's best score used for ranking.
type TalentScore struct {
	TalentID string
	Score    float64
}
