package testevents

import "time"

// Config holds configuration for the event test
type Config struct {
	BaseURL    string        // Base URL of the service
	NumEvents  int           // Number of events to generate
	TopN       int           // Number of top entries to fetch
	Workers    int           // Number of concurrent workers
	Timeout    time.Duration // HTTP request timeout
	OutputFile string        // Output file for events
	LogFile    string        // Log file for test output
	Verbose    bool          // Enable verbose logging
}

// Event represents an event to be submitted
type Event struct {
	EventID   string  `json:"event_id"`
	TalentID  string  `json:"talent_id"`
	RawMetric float64 `json:"raw_metric"`
	Skill     string  `json:"skill"`
	TS        string  `json:"ts"`
}

// Entry represents a leaderboard entry
type Entry struct {
	Rank     int     `json:"rank"`
	TalentID string  `json:"talent_id"`
	Score    float64 `json:"score"`
}

// AckResponse represents the response from event submission
type AckResponse struct {
	Status    string `json:"status"`
	Duplicate bool   `json:"duplicate"`
}

// Stats holds test statistics
type Stats struct {
	EventsGenerated    int
	EventsSubmitted    int
	EventsSuccessful   int
	EventsDuplicate    int
	EventsFailed       int
	RankingsRetrieved  int
	LeaderboardEntries int
	StartTime          time.Time
	EndTime            time.Time
	Duration           time.Duration
}
