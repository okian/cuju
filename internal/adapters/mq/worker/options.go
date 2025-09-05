// Package worker defines worker contracts for asynchronous scoring and updates.
package worker

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
