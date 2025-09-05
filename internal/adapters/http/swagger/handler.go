package swagger

import (
	"context"
	"errors"
	"net/http"
)

// Error constants.
var (
	ErrServe = errors.New("swagger serve failed")
)

// Register attaches Swagger UI and the OpenAPI spec routes to mux.
// Routes:.
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
