package testevents

import "time"

// HTTP status code constants.
const (
	StatusOK       = 200
	StatusAccepted = 202
)

// Worker configuration constants.
const (
	WorkerChannelMultiplier = 2
)

// Runner configuration constants.
const (
	HealthCheckDelay     = 2 * time.Minute
	PercentageMultiplier = 100
)
