package metrics

import (
	"errors"
)

// Sentinel kinds for metrics errors.
var (
	ErrObserveFailed = errors.New("metrics observe failed")
)
