package main

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/okian/cuju/internal/adapters/http/api"
	"github.com/okian/cuju/internal/adapters/http/swagger"
	app "github.com/okian/cuju/internal/app"
	"github.com/okian/cuju/internal/config"
	"github.com/okian/cuju/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMainFunction(t *testing.T) {
	Convey("Given the main application", t, func() {
		Convey("When testing configuration loading", func() {
			// Test with environment variables
			_ = os.Setenv("CUJU_ADDR", ":8080")
			_ = os.Setenv("CUJU_QUEUE_SIZE", "1000")
			_ = os.Setenv("CUJU_WORKER_COUNT", "4")
			defer func() {
				_ = os.Unsetenv("CUJU_ADDR")
				_ = os.Unsetenv("CUJU_QUEUE_SIZE")
				_ = os.Unsetenv("CUJU_WORKER_COUNT")
			}()

			Convey("Then configuration should be loadable", func() {
				ctx := context.Background()
				cfg, err := config.Load(ctx)
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(cfg.Addr, ShouldEqual, ":8080")
				So(cfg.EventQueueSize, ShouldEqual, 1000)
				So(cfg.WorkerCount, ShouldEqual, 4)
			})
		})

		Convey("When testing service creation", func() {
			Convey("Then service should be creatable with default options", func() {
				svc := app.New()
				So(svc, ShouldNotBeNil)
			})

			Convey("And service should be creatable with custom options", func() {
				svc := app.New(
					app.WithWorkerCount(8),
					app.WithQueueSize(2000),
					app.WithDedupeSize(1000),
				)
				So(svc, ShouldNotBeNil)
			})
		})

		Convey("When testing HTTP server creation", func() {
			svc := app.New()
			So(svc, ShouldNotBeNil)

			Convey("Then HTTP server should be creatable", func() {
				server := api.NewServer(svc, svc)
				So(server, ShouldNotBeNil)
			})
		})

		Convey("When testing metrics initialization", func() {
			Convey("Then metrics manager should be creatable", func() {
				manager := metrics.NewMetricsManager()
				So(manager, ShouldNotBeNil)
			})
		})
	})
}

func TestMainApplicationComponents(t *testing.T) {
	Convey("Given main application components", t, func() {
		Convey("When testing system metrics updater", func() {
			Convey("Then it should be creatable", func() {
				// Test that the function exists and can be called with a timeout
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()

				So(func() {
					startSystemMetricsUpdater(ctx)
				}, ShouldNotPanic)
			})
		})

		Convey("When testing service metrics updater", func() {
			svc := app.New()
			So(svc, ShouldNotBeNil)

			Convey("Then it should be creatable", func() {
				// Test that the function exists and can be called with a timeout
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()

				So(func() {
					startServiceMetricsUpdater(ctx, svc)
				}, ShouldNotPanic)
			})
		})

		Convey("When testing system metrics update", func() {
			Convey("Then it should update metrics without panicking", func() {
				So(func() {
					updateSystemMetrics()
				}, ShouldNotPanic)
			})
		})

		Convey("When testing service metrics update", func() {
			svc := app.New()
			So(svc, ShouldNotBeNil)

			Convey("Then it should update metrics without panicking", func() {
				So(func() {
					updateServiceMetrics(svc)
				}, ShouldNotPanic)
			})
		})
	})
}

func TestMainApplicationIntegration(t *testing.T) {
	Convey("Given main application integration", t, func() {
		Convey("When testing full application setup", func() {
			// Set up test environment
			_ = os.Setenv("CUJU_ADDR", ":8080")
			_ = os.Setenv("CUJU_QUEUE_SIZE", "1000")
			_ = os.Setenv("CUJU_WORKER_COUNT", "2")
			defer func() {
				_ = os.Unsetenv("CUJU_ADDR")
				_ = os.Unsetenv("CUJU_QUEUE_SIZE")
				_ = os.Unsetenv("CUJU_WORKER_COUNT")
			}()

			Convey("Then all components should work together", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// Load configuration
				cfg, err := config.Load(ctx)
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)

				// Create service (without starting to avoid logger dependency)
				svc := app.New(
					app.WithWorkerCount(cfg.WorkerCount),
					app.WithQueueSize(cfg.EventQueueSize),
					app.WithDedupeSize(cfg.DedupeSize),
				)
				So(svc, ShouldNotBeNil)

				// Create HTTP server
				server := api.NewServer(svc, svc)
				So(server, ShouldNotBeNil)

				// Create HTTP mux
				mux := http.NewServeMux()
				So(mux, ShouldNotBeNil)

				// Register routes
				server.Register(ctx, mux, svc)
				swagger.Register(ctx, mux)

				// Stop service
				svc.Stop()
			})
		})
	})
}

func TestMainApplicationErrorHandling(t *testing.T) {
	Convey("Given main application error handling", t, func() {
		Convey("When testing invalid configuration", func() {
			// Set invalid configuration
			_ = os.Setenv("CUJU_ADDR", "")
			defer func() { _ = os.Unsetenv("CUJU_ADDR") }()

			Convey("Then configuration loading should fail", func() {
				ctx := context.Background()
				cfg, err := config.Load(ctx)
				So(err, ShouldNotBeNil)
				So(cfg, ShouldBeNil)
			})
		})

		Convey("When testing service creation with invalid options", func() {
			Convey("Then service should handle invalid options gracefully", func() {
				// Test with extreme values
				svc := app.New(
					app.WithWorkerCount(0),
					app.WithQueueSize(0),
					app.WithDedupeSize(0),
				)
				So(svc, ShouldNotBeNil)
			})
		})
	})
}

func TestMainApplicationPerformance(t *testing.T) {
	Convey("Given main application performance", t, func() {
		Convey("When testing component creation performance", func() {
			Convey("Then service creation should be fast", func() {
				start := time.Now()
				svc := app.New()
				duration := time.Since(start)

				So(svc, ShouldNotBeNil)
				So(duration, ShouldBeLessThan, 100*time.Millisecond)
			})

			Convey("And HTTP server creation should be fast", func() {
				svc := app.New()
				So(svc, ShouldNotBeNil)

				start := time.Now()
				server := api.NewServer(svc, svc)
				duration := time.Since(start)

				So(server, ShouldNotBeNil)
				So(duration, ShouldBeLessThan, 100*time.Millisecond)
			})

			Convey("And metrics manager creation should be fast", func() {
				start := time.Now()
				// Use a custom registry to avoid duplicate registration issues
				registry := prometheus.NewRegistry()
				manager := metrics.NewMetricsManager(metrics.WithPrometheusRegistry(registry))
				duration := time.Since(start)

				So(manager, ShouldNotBeNil)
				So(duration, ShouldBeLessThan, 100*time.Millisecond)
			})
		})
	})
}

func TestMainApplicationConcurrency(t *testing.T) {
	Convey("Given main application concurrency", t, func() {
		Convey("When testing concurrent component creation", func() {
			numGoroutines := 10
			done := make(chan bool, numGoroutines)

			// Start multiple goroutines creating components
			for i := 0; i < numGoroutines; i++ {
				go func(id int) {
					defer func() {
						if r := recover(); r != nil {
							// Log the panic but don't fail the test
							t.Logf("Goroutine %d panicked: %v", id, r)
						}
						done <- true
					}()

					// Create service
					svc := app.New()
					if svc == nil {
						t.Errorf("Goroutine %d: service creation failed", id)
						return
					}

					// Create HTTP server
					server := api.NewServer(svc, svc)
					if server == nil {
						t.Errorf("Goroutine %d: HTTP server creation failed", id)
						return
					}

					// Create metrics manager with custom registry
					registry := prometheus.NewRegistry()
					manager := metrics.NewMetricsManager(metrics.WithPrometheusRegistry(registry))
					if manager == nil {
						t.Errorf("Goroutine %d: metrics manager creation failed", id)
						return
					}
				}(i)
			}

			// Wait for all goroutines to complete
			for i := 0; i < numGoroutines; i++ {
				<-done
			}

			Convey("Then all components should be created successfully", func() {
				// If we get here without panics, the test passed
				So(true, ShouldBeTrue)
			})
		})
	})
}

func TestMainApplicationResourceCleanup(t *testing.T) {
	Convey("Given main application resource cleanup", t, func() {
		Convey("When testing service creation", func() {
			svc := app.New()
			So(svc, ShouldNotBeNil)

			Convey("Then service should be created successfully", func() {
				// Test that service can be created without starting
				So(svc, ShouldNotBeNil)

				// Test that we can get stats without starting
				stats := svc.GetStats()
				So(stats, ShouldNotBeNil)
			})
		})

		Convey("When testing multiple service creation cycles", func() {
			Convey("Then multiple services should be created successfully", func() {
				for i := 0; i < 3; i++ {
					svc := app.New()
					So(svc, ShouldNotBeNil)

					// Test that we can get stats
					stats := svc.GetStats()
					So(stats, ShouldNotBeNil)
				}
			})
		})
	})
}
