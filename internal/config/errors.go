package config

import (
	"errors"
)

// Sentinel error kinds for this package. These allow errors.Is/As from callers.
var (
	ErrInvalidConfig = errors.New("invalid config")
	ErrLoadConfig    = errors.New("load config failed")
)
