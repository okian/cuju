// Package site handles the embedded documentation site.
package site

import (
	"context"
	"errors"
	"net/http"
)

// Error constants
var (
	ErrGenerate = errors.New("docs site generation failed")
	ErrServe    = errors.New("docs site serve failed")
)

// Register attaches the embedded documentation site routes to mux.
func Register(_ context.Context, mux *http.ServeMux) {
	if mux == nil {
		panic("mux is nil")
	}

	// Serve the embedded documentation site at root /
	files := http.FileServer(FS())
	mux.Handle("/", files)

}

// RootHandler handles root path requests
type RootHandler struct{}

// NewRootHandler creates a new root handler
func NewRootHandler() *RootHandler {
	return &RootHandler{}
}

// HandleRoot handles GET / requests and serves the embedded documentation site
func (h *RootHandler) HandleRoot(w http.ResponseWriter, r *http.Request) {
	// Serve the documentation index page
	files := http.FileServer(FS())
	files.ServeHTTP(w, r)
}
