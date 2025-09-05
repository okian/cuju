package api

import "errors"

// Sentinel kinds for API errors.
var (
	ErrServe        = errors.New("swagger serve failed")
	ErrBadRequest   = errors.New("bad request")
	ErrBackpressure = errors.New("backpressure")
)
