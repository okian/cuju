package swagger

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// Error represents a swagger-related error
type Error struct {
	Op   string
	Kind error
	Err  error
}

// Error implements the error interface
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.Op != "" && e.Kind != nil && e.Err != nil {
		return fmt.Sprintf("%s: %s: %s", e.Op, e.Kind.Error(), e.Err.Error())
	}
	if e.Op != "" && e.Kind != nil {
		return fmt.Sprintf("%s: %s", e.Op, e.Kind.Error())
	}
	if e.Op != "" && e.Err != nil {
		return fmt.Sprintf("%s: %s", e.Op, e.Err.Error())
	}
	if e.Kind != nil {
		return e.Kind.Error()
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "unknown error"
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Err
}

// Is checks if the error matches the target
func (e *Error) Is(target error) bool {
	if e == nil {
		return target == nil
	}
	if e.Kind == target {
		return true
	}
	if e.Err == target {
		return true
	}
	return errors.Is(e.Err, target)
}

// Error constants
var (
	ErrServe = errors.New("swagger serve failed")
)

// Wrap wraps an error with operation context
func Wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Op: op, Err: err}
}

// WrapKind wraps an error with operation and kind context
func WrapKind(op string, kind error, err error) error {
	return &Error{Op: op, Kind: kind, Err: err}
}

// NewKind creates a new error with operation and kind
func NewKind(op string, kind error) error {
	return &Error{Op: op, Kind: kind}
}

// Register attaches Swagger UI and the OpenAPI spec routes to mux.
// Routes:
//
//	GET /swagger                  -> ReDoc HTML
//	GET /swagger/openapi.yaml     -> Embedded OpenAPI spec
//	GET /swagger/redoc.standalone.js -> Embedded ReDoc JavaScript
func Register(_ context.Context, mux *http.ServeMux) {
	if mux == nil {
		panic("mux is nil")
	}

	// Serve ReDoc HTML at /api-docs
	mux.HandleFunc("/api-docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(indexHTML))
	})

	// Serve OpenAPI spec at /openapi.yaml
	mux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		_, _ = w.Write(OpenAPI)
	})

	// Serve embedded ReDoc JavaScript at /api-docs/redoc.standalone.js
	mux.HandleFunc("/api-docs/redoc.standalone.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		_, _ = w.Write(RedocJS)
	})
}

// Minimal HTML that uses embedded ReDoc and loads /swagger/openapi.yaml.
const indexHTML = `<!doctype html>
<html>
  <head>
    <meta charset="utf-8">
    <title>API Docs – ReDoc</title>
    <style>body{margin:0;padding:0}</style>
  </head>
  <body>
    <redoc id="redoc-container"></redoc>
    <script src="/api-docs/redoc.standalone.js"></script>
    <script>Redoc.init('/openapi.yaml', { suppressWarnings: true }, document.getElementById('redoc-container'));</script>
  </body>
</html>`
