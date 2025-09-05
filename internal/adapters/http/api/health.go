// Package api declares HTTP contracts and route registration helpers.
package api

import (
	"net/http"

	"github.com/okian/cuju/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HealthHandler handles health check requests.
type HealthHandler struct{}

// NewHealthHandler creates a new health handler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HandleHealth handles GET /healthz requests.
// If the Accept header contains "application/openmetrics-text" or "text/plain",.
// it returns Prometheus metrics. Otherwise, it returns JSON health status.
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	// Use our custom metrics registry to serve metrics
	promhttp.HandlerFor(metrics.GetRegistry(), promhttp.HandlerOpts{}).ServeHTTP(w, r)
}
