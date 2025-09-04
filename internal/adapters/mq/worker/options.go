// Package worker defines worker contracts for asynchronous scoring and updates.
package worker

import (
	"github.com/okian/cuju/pkg/logger"
)

// Option applies a configuration option to the InMemoryWorker.
type Option func(*InMemoryWorker)

// WithName sets the worker name for identification and logging.
func WithName(name string) Option {
	return func(w *InMemoryWorker) {
		if name != "" {
			w.name = name
		}
	}
}

// WithLogger sets a custom logger for the worker.
func WithLogger(logger logger.Logger) Option {
	return func(w *InMemoryWorker) {
		if logger != nil {
			w.logger = logger
		}
	}
}
