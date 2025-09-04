// Package api declares HTTP contracts and route registration helpers.
package api

import (
	"net/http"
)

// dashboardHandler handles dashboard requests
type dashboardHandler struct{}

// newdashboardHandler creates a new dashboard handler
func newdashboardHandler() *dashboardHandler {
	return &dashboardHandler{}
}

// HandleDashboard handles GET /dashboard requests
// Returns an HTML page with JavaScript to scrape and visualize metrics from /healthz
func (h *dashboardHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	// Serve embedded dashboard page
	http.ServeFileFS(w, r, dashboardFS, "dashboard.html")
}
