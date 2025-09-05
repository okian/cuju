// Package api declares HTTP contracts and route registration helpers.
package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/okian/cuju/pkg/metrics"
)

// HTTP status code constants.
const (
	statusBadRequest      = 400
	statusNotFound        = 404
	statusTooManyRequests = 429
	statusInternalError   = 500
)

// MetricsMiddleware wraps HTTP handlers to record Prometheus metrics.
func MetricsMiddleware(next http.HandlerFunc, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		// Record metrics
		durationMs := float64(time.Since(start).Milliseconds())
		statusCodeStr := strconv.Itoa(wrapped.statusCode)

		// Record basic HTTP metrics
		metrics.RecordHTTPRequest(endpoint, r.Method, statusCodeStr)
		metrics.RecordHTTPRequestDuration(endpoint, r.Method, statusCodeStr, durationMs)

		// Record error metrics if status indicates an error
		if wrapped.statusCode >= statusBadRequest {
			errorType := getErrorType(wrapped.statusCode)
			severity := getErrorSeverity(wrapped.statusCode)
			metrics.RecordErrorByEndpoint(endpoint, r.Method, errorType)
			metrics.RecordErrorByType(errorType, severity)
			metrics.RecordErrorLatency("http", errorType, durationMs)
		}
	}
}

// getErrorType returns a standardized error type based on HTTP status code.
func getErrorType(statusCode int) string {
	switch {
	case statusCode >= statusInternalError:
		return "server_error"
	case statusCode == statusTooManyRequests:
		return "rate_limit"
	case statusCode == statusNotFound:
		return "not_found"
	case statusCode >= statusBadRequest:
		return "client_error"
	default:
		return "unknown"
	}
}

// getErrorSeverity returns error severity based on HTTP status code.
func getErrorSeverity(statusCode int) string {
	switch {
	case statusCode >= statusInternalError:
		return "high"
	case statusCode >= statusBadRequest:
		return "medium"
	default:
		return "low"
	}
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	if err != nil {
		return n, fmt.Errorf("failed to write response: %w", err)
	}
	return n, nil
}
