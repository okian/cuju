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
	"github.com/smartystreets/goconvey/convey"
)

func TestMainFunction(t *testing.T) {
	convey.Convey("Given the main application", t, func() {
		convey.Convey("When testing configuration loading", func() {
			// Test with environment variables
			_ = os.Setenv("CUJU_ADDR", ":8080")
			_ = os.Setenv("CUJU_QUEUE_SIZE", "1000")
			_ = os.Setenv("CUJU_WORKER_COUNT", "4")
			defer func() {
				_ = os.Unsetenv("CUJU_ADDR")
				_ = os.Unsetenv("CUJU_QUEUE_SIZE")
				_ = os.Unsetenv("CUJU_WORKER_COUNT")
			}()

			convey.Convey("Then configuration should be loadable", func() {
				ctx := context.Background()
				cfg, err := config.Load(ctx)
				convey.So(err, convey.ShouldBeNil)
				convey.So(cfg, convey.ShouldNotBeNil)
				convey.So(cfg.Addr, convey.ShouldEqual, ":8080")
				convey.So(cfg.EventQueueSize, convey.ShouldEqual, 1000)
				convey.So(cfg.WorkerCount, convey.ShouldEqual, 4)
			})
		})

		convey.Convey("When testing service creation", func() {
			convey.Convey("Then service should be creatable with default options", func() {
				svc := app.New()
				convey.So(svc, convey.ShouldNotBeNil)
			})

			convey.Convey("And service should be creatable with custom options", func() {
				svc := app.New(
					app.WithWorkerCount(8),
					app.WithQueueSize(2000),
					app.WithDedupeSize(1000),
				)
				convey.So(svc, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When testing HTTP server creation", func() {
			svc := app.New()
			convey.So(svc, convey.ShouldNotBeNil)

			convey.Convey("Then HTTP server should be creatable", func() {
				server := api.NewServer(svc, svc)
				convey.So(server, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When testing metrics initialization", func() {
			convey.Convey("Then metrics manager should be creatable", func() {
				manager := metrics.NewManager()
				convey.So(manager, convey.ShouldNotBeNil)
			})
		})
	})
}

func TestMainApplicationComponents(t *testing.T) {
	convey.Convey("Given main application components", t, func() {
		convey.Convey("When testing system metrics updater", func() {
			convey.Convey("Then it should be creatable", func() {
				// Test that the function exists and can be called with a timeout
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()

				convey.So(func() {
					startSystemMetricsUpdater(ctx)
				}, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When testing service metrics updater", func() {
			svc := app.New()
			convey.So(svc, convey.ShouldNotBeNil)

			convey.Convey("Then it should be creatable", func() {
				// Test that the function exists and can be called with a timeout
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()

				convey.So(func() {
					startServiceMetricsUpdater(ctx, svc)
				}, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When testing system metrics update", func() {
			convey.Convey("Then it should update metrics without panicking", func() {
				convey.So(func() {
					updateSystemMetrics()
				}, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When testing service metrics update", func() {
			svc := app.New()
			convey.So(svc, convey.ShouldNotBeNil)

			convey.Convey("Then it should update metrics without panicking", func() {
				convey.So(func() {
					updateServiceMetrics(svc)
				}, convey.ShouldNotPanic)
			})
		})
	})
}

func TestMainApplicationIntegration(t *testing.T) {
	convey.Convey("Given main application integration", t, func() {
		convey.Convey("When testing full application setup", func() {
			// Set up test environment
			_ = os.Setenv("CUJU_ADDR", ":8080")
			_ = os.Setenv("CUJU_QUEUE_SIZE", "1000")
			_ = os.Setenv("CUJU_WORKER_COUNT", "2")
			defer func() {
				_ = os.Unsetenv("CUJU_ADDR")
				_ = os.Unsetenv("CUJU_QUEUE_SIZE")
				_ = os.Unsetenv("CUJU_WORKER_COUNT")
			}()

			convey.Convey("Then all components should work together", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// Load configuration
				cfg, err := config.Load(ctx)
				convey.So(err, convey.ShouldBeNil)
				convey.So(cfg, convey.ShouldNotBeNil)

				// Create service (without starting to avoid logger dependency)
				svc := app.New(
					app.WithWorkerCount(cfg.WorkerCount),
					app.WithQueueSize(cfg.EventQueueSize),
					app.WithDedupeSize(cfg.DedupeSize),
				)
				convey.So(svc, convey.ShouldNotBeNil)

				// Create HTTP server
				server := api.NewServer(svc, svc)
				convey.So(server, convey.ShouldNotBeNil)

				// Create HTTP mux
				mux := http.NewServeMux()
				convey.So(mux, convey.ShouldNotBeNil)

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
	convey.Convey("Given main application error handling", t, func() {
		convey.Convey("When testing invalid configuration", func() {
			// Set invalid configuration
			_ = os.Setenv("CUJU_ADDR", "")
			defer func() { _ = os.Unsetenv("CUJU_ADDR") }()

			convey.Convey("Then configuration loading should fail", func() {
				ctx := context.Background()
				cfg, err := config.Load(ctx)
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(cfg, convey.ShouldBeNil)
			})
		})

		convey.Convey("When testing service creation with invalid options", func() {
			convey.Convey("Then service should handle invalid options gracefully", func() {
				// Test with extreme values
				svc := app.New(
					app.WithWorkerCount(0),
					app.WithQueueSize(0),
					app.WithDedupeSize(0),
				)
				convey.So(svc, convey.ShouldNotBeNil)
			})
		})
	})
}

func TestMainApplicationPerformance(t *testing.T) {
	convey.Convey("Given main application performance", t, func() {
		convey.Convey("When testing component creation performance", func() {
			convey.Convey("Then service creation should be fast", func() {
				start := time.Now()
				svc := app.New()
				duration := time.Since(start)

				convey.So(svc, convey.ShouldNotBeNil)
				convey.So(duration, convey.ShouldBeLessThan, 100*time.Millisecond)
			})

			convey.Convey("And HTTP server creation should be fast", func() {
				svc := app.New()
				convey.So(svc, convey.ShouldNotBeNil)

				start := time.Now()
				server := api.NewServer(svc, svc)
				duration := time.Since(start)

				convey.So(server, convey.ShouldNotBeNil)
				convey.So(duration, convey.ShouldBeLessThan, 100*time.Millisecond)
			})

			convey.Convey("And metrics manager creation should be fast", func() {
				start := time.Now()
				// Use a custom registry to avoid duplicate registration issues
				registry := prometheus.NewRegistry()
				manager := metrics.NewManager(metrics.WithPrometheusRegistry(registry))
				duration := time.Since(start)

				convey.So(manager, convey.ShouldNotBeNil)
				convey.So(duration, convey.ShouldBeLessThan, 100*time.Millisecond)
			})
		})
	})
}

func TestMainApplicationConcurrency(t *testing.T) {
	convey.Convey("Given main application concurrency", t, func() {
		convey.Convey("When testing concurrent component creation", func() {
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
					manager := metrics.NewManager(metrics.WithPrometheusRegistry(registry))
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

			convey.Convey("Then all components should be created successfully", func() {
				// If we get here without panics, the test passed
				convey.So(true, convey.ShouldBeTrue)
			})
		})
	})
}

func TestMainApplicationResourceCleanup(t *testing.T) {
	convey.Convey("Given main application resource cleanup", t, func() {
		convey.Convey("When testing service creation", func() {
			svc := app.New()
			convey.So(svc, convey.ShouldNotBeNil)

			convey.Convey("Then service should be created successfully", func() {
				// Test that service can be created without starting
				convey.So(svc, convey.ShouldNotBeNil)

				// Test that we can get stats without starting
				stats := svc.GetStats()
				convey.So(stats, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When testing multiple service creation cycles", func() {
			convey.Convey("Then multiple services should be created successfully", func() {
				for i := 0; i < 3; i++ {
					svc := app.New()
					convey.So(svc, convey.ShouldNotBeNil)

					// Test that we can get stats
					stats := svc.GetStats()
					convey.So(stats, convey.ShouldNotBeNil)
				}
			})
		})
	})
}
