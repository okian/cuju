package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/okian/cuju/internal/adapters/http/api"
	"github.com/okian/cuju/internal/adapters/http/swagger"
	app "github.com/okian/cuju/internal/app"
	"github.com/okian/cuju/internal/config"
	"github.com/okian/cuju/pkg/logger"
	"github.com/okian/cuju/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// HTTP server timeout constants.
const (
	readTimeout               = 10 * time.Second
	writeTimeout              = 10 * time.Second
	idleTimeout               = 60 * time.Second
	readHeaderTimeout         = 5 * time.Second
	shutdownTimeout           = 30 * time.Second
	systemMetricsInterval     = 10 * time.Second
	serviceMetricsInterval    = 5 * time.Second
	nanosecondsPerMillisecond = 1e6
)

func main() {
	// Disable default Go metrics collection to avoid duplicate metrics
	// We collect our own custom system metrics instead
	prometheus.Unregister(collectors.NewGoCollector())
	prometheus.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// Initialize logging
	if err := logger.Init(); err != nil {
		// Use fmt for initialization errors since logger isn't available yet
		os.Stderr.WriteString("failed to initialize logging: " + err.Error() + "\n")
		return
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			logger.Error(err)
		}
	}()

	loggerInstance := logger.Get()

	// Root context with cancel on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Load configuration (defaults -> optional file -> env)
	cfg, err := config.Load(ctx)
	if err != nil {
		// Use fmt for initialization errors since logger isn't available yet
		os.Stderr.WriteString("failed to load config: " + err.Error() + "\n")
		return
	}

	// Apply configured log level (fallback to info on invalid input)
	if err := logger.SetLevelString(cfg.LogLevel); err != nil {
		loggerInstance.Warn(ctx, "invalid log_level; falling back to info", logger.String("log_level", cfg.LogLevel), logger.Error(err))
		_ = logger.SetLevelString("info")
	}

	// Create and start the service with configuration options
	svc := app.New(
		app.WithLogger(loggerInstance),
		app.WithWorkerCount(cfg.WorkerCount),
		app.WithQueueSize(cfg.EventQueueSize),
		app.WithDedupeSize(cfg.DedupeSize),
		app.WithSkillWeights(cfg.SkillWeights),
		app.WithDefaultSkillWeight(cfg.DefaultSkillWeight),
		app.WithScoringLatencyRange(time.Duration(cfg.ScoringLatencyMinMS)*time.Millisecond, time.Duration(cfg.ScoringLatencyMaxMS)*time.Millisecond),
	)
	if err := svc.Start(ctx); err != nil {
		// Use fmt for initialization errors since logger isn't available yet
		os.Stderr.WriteString("failed to start service: " + err.Error() + "\n")
		return
	}
	defer svc.Stop()

	// Start system metrics updater
	go startSystemMetricsUpdater(ctx)

	// Start service metrics updater
	go startServiceMetricsUpdater(ctx, svc)

	// HTTP mux and routes.
	mux := http.NewServeMux()

	// Register Swagger UI under /swagger
	swagger.Register(ctx, mux)

	// Register business API routes with the service dependency.
	apiServer := api.NewServer(svc, svc)
	apiServer.Register(ctx, mux, svc)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	// Start the HTTP server
	go func() {
		loggerInstance.Info(ctx, "starting HTTP server", logger.String("addr", cfg.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Use fmt for server errors
			os.Stderr.WriteString("HTTP server failed: " + err.Error() + "\n")
			return
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	loggerInstance.Info(ctx, "shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		loggerInstance.Error(ctx, "server shutdown failed", logger.Error(err))
	}

	loggerInstance.Info(ctx, "server stopped")
}

// startSystemMetricsUpdater starts a background goroutine that updates system metrics.
func startSystemMetricsUpdater(ctx context.Context) {
	ticker := time.NewTicker(systemMetricsInterval) // Update every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			updateSystemMetrics()
		}
	}
}

// startServiceMetricsUpdater starts a background goroutine that updates service metrics.
func startServiceMetricsUpdater(ctx context.Context, svc *app.Service) {
	ticker := time.NewTicker(serviceMetricsInterval) // Update every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			updateServiceMetrics(svc)
		}
	}
}

// updateSystemMetrics updates system-level metrics.
func updateSystemMetrics() {
	// Update memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	metrics.UpdateSystemMemoryUsage(m.Alloc)

	// Update goroutine count
	metrics.UpdateSystemGoroutineCount(runtime.NumGoroutine())

	// Update GC pause time
	if m.NumGC > 0 {
		// Calculate average GC pause time
		avgPauseMs := float64(m.PauseTotalNs) / float64(m.NumGC) / nanosecondsPerMillisecond
		metrics.RecordSystemGCPauseTime(avgPauseMs)
	}
}

// updateServiceMetrics updates service-level metrics.
func updateServiceMetrics(svc *app.Service) {
	// Get current stats from the service
	stats := svc.GetStats()

	// The GetStats method already updates the metrics, but we can also
	// update additional metrics here if needed
	if queueLen, ok := stats["queueLength"].(int); ok {
		metrics.UpdateQueueSize(queueLen)
	}

	if totalTalents, ok := stats["totalTalents"].(int); ok {
		metrics.UpdateTotalTalents(totalTalents)
	}

	if workerCount, ok := stats["workerCount"].(int); ok {
		metrics.UpdateWorkerCount(workerCount)
	}
}
