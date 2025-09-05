package config_test

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/okian/cuju/internal/config"
	"github.com/smartystreets/goconvey/convey"
)

func TestConfigLoader(t *testing.T) {
	convey.Convey("Given a config loader", t, func() {
		ctx := context.Background()

		convey.Convey("When loading config with defaults only", func() {
			// Clear any existing environment variables
			clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then it should load successfully with defaults", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(cfg, convey.ShouldNotBeNil)
				convey.So(cfg.Addr, convey.ShouldEqual, ":9080")
				convey.So(cfg.EventQueueSize, convey.ShouldEqual, 200_000)
				convey.So(cfg.WorkerCount, convey.ShouldEqual, runtime.NumCPU()*20) // runtime.NumCPU() * 20
				convey.So(cfg.DedupeSize, convey.ShouldEqual, 500_000)
				convey.So(cfg.ScoringLatencyMinMS, convey.ShouldEqual, 80)
				convey.So(cfg.ScoringLatencyMaxMS, convey.ShouldEqual, 150)
			})
		})

		convey.Convey("When loading config with environment variables", func() {
			// Set environment variables
			_ = os.Setenv("CUJU_ADDR", ":8080")
			_ = os.Setenv("CUJU_QUEUE_SIZE", "100000")
			_ = os.Setenv("CUJU_WORKER_COUNT", "16")
			_ = os.Setenv("CUJU_DEDUPE_SIZE", "250000")
			_ = os.Setenv("CUJU_SCORING_LATENCY_MIN_MS", "50")
			_ = os.Setenv("CUJU_SCORING_LATENCY_MAX_MS", "100")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then it should override defaults with env vars", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(cfg, convey.ShouldNotBeNil)
				convey.So(cfg.Addr, convey.ShouldEqual, ":8080")
				convey.So(cfg.EventQueueSize, convey.ShouldEqual, 100000)
				convey.So(cfg.WorkerCount, convey.ShouldEqual, 16)
				convey.So(cfg.DedupeSize, convey.ShouldEqual, 250000)
				convey.So(cfg.ScoringLatencyMinMS, convey.ShouldEqual, 50)
				convey.So(cfg.ScoringLatencyMaxMS, convey.ShouldEqual, 100)
			})
		})

		convey.Convey("When loading config with YAML file", func() {
			// Create a temporary YAML config file
			yamlContent := `
addr: ":9090"
queue_size: 300000
worker_count: 24
dedupe_size: 600000
scoring_latency_min_ms: 60
scoring_latency_max_ms: 120
`
			tmpFile := createTempConfigFile(yamlContent)
			defer func() { _ = os.Remove(tmpFile) }()

			// Set the config file path
			_ = os.Setenv("CUJU_CONFIG", tmpFile)
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then it should load from YAML file", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(cfg, convey.ShouldNotBeNil)
				convey.So(cfg.Addr, convey.ShouldEqual, ":9090")
				convey.So(cfg.EventQueueSize, convey.ShouldEqual, 300000)
				convey.So(cfg.WorkerCount, convey.ShouldEqual, 24)
				convey.So(cfg.DedupeSize, convey.ShouldEqual, 600000)
				convey.So(cfg.ScoringLatencyMinMS, convey.ShouldEqual, 60)
				convey.So(cfg.ScoringLatencyMaxMS, convey.ShouldEqual, 120)
			})
		})

		convey.Convey("When loading config with both file and environment variables", func() {
			// Create a YAML config file
			yamlContent := `
addr: ":9090"
queue_size: 300000
worker_count: 24
dedupe_size: 600000
scoring_latency_min_ms: 60
scoring_latency_max_ms: 120
`
			tmpFile := createTempConfigFile(yamlContent)
			defer func() { _ = os.Remove(tmpFile) }()

			// Set both file and environment variables
			_ = os.Setenv("CUJU_CONFIG", tmpFile)
			_ = os.Setenv("CUJU_ADDR", ":8080")      // This should override the file
			_ = os.Setenv("CUJU_WORKER_COUNT", "32") // This should override the file
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then environment variables should override file values", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(cfg, convey.ShouldNotBeNil)
				convey.So(cfg.Addr, convey.ShouldEqual, ":8080")            // Overridden by env
				convey.So(cfg.EventQueueSize, convey.ShouldEqual, 300000)   // From file
				convey.So(cfg.WorkerCount, convey.ShouldEqual, 32)          // Overridden by env
				convey.So(cfg.DedupeSize, convey.ShouldEqual, 600000)       // From file
				convey.So(cfg.ScoringLatencyMinMS, convey.ShouldEqual, 60)  // From file
				convey.So(cfg.ScoringLatencyMaxMS, convey.ShouldEqual, 120) // From file
			})
		})

		convey.Convey("When loading config with invalid YAML file", func() {
			// Create an invalid YAML file
			invalidYaml := `invalid: yaml: content: [`
			tmpFile := createTempConfigFile(invalidYaml)
			defer func() { _ = os.Remove(tmpFile) }()

			_ = os.Setenv("CUJU_CONFIG", tmpFile)
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then it should return an error", func() {
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(cfg, convey.ShouldBeNil)
			})
		})

		convey.Convey("When loading config with non-existent file", func() {
			_ = os.Setenv("CUJU_CONFIG", "/non/existent/file.yaml")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then it should return an error", func() {
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(cfg, convey.ShouldBeNil)
			})
		})

		convey.Convey("When loading config with empty addr", func() {
			_ = os.Setenv("CUJU_ADDR", "")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then it should return a validation error", func() {
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(err.Error(), convey.ShouldContainSubstring, "addr must not be empty")
				convey.So(cfg, convey.ShouldBeNil)
			})
		})

		convey.Convey("When loading config with partial YAML file", func() {
			// Create a YAML file with only some fields
			yamlContent := `
addr: ":9090"
worker_count: 16
`
			tmpFile := createTempConfigFile(yamlContent)
			defer func() { _ = os.Remove(tmpFile) }()

			_ = os.Setenv("CUJU_CONFIG", tmpFile)
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then it should merge with defaults for missing fields", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(cfg, convey.ShouldNotBeNil)
				convey.So(cfg.Addr, convey.ShouldEqual, ":9090")            // From file
				convey.So(cfg.WorkerCount, convey.ShouldEqual, 16)          // From file
				convey.So(cfg.EventQueueSize, convey.ShouldEqual, 200_000)  // From defaults
				convey.So(cfg.DedupeSize, convey.ShouldEqual, 500_000)      // From defaults
				convey.So(cfg.ScoringLatencyMinMS, convey.ShouldEqual, 80)  // From defaults
				convey.So(cfg.ScoringLatencyMaxMS, convey.ShouldEqual, 150) // From defaults
			})
		})

		convey.Convey("When loading config with environment variables using different cases", func() {
			// Test case insensitivity
			_ = os.Setenv("CUJU_ADDR", ":8080")
			_ = os.Setenv("CUJU_QUEUE_SIZE", "100000") // uppercase prefix
			_ = os.Setenv("CUJU_WORKER_COUNT", "16")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then it should handle case insensitive environment variables", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(cfg, convey.ShouldNotBeNil)
				convey.So(cfg.Addr, convey.ShouldEqual, ":8080")
				convey.So(cfg.EventQueueSize, convey.ShouldEqual, 100000)
				convey.So(cfg.WorkerCount, convey.ShouldEqual, 16)
			})
		})

		convey.Convey("When loading config with numeric environment variables", func() {
			_ = os.Setenv("CUJU_QUEUE_SIZE", "500000")
			_ = os.Setenv("CUJU_WORKER_COUNT", "32")
			_ = os.Setenv("CUJU_DEDUPE_SIZE", "750000")
			_ = os.Setenv("CUJU_SCORING_LATENCY_MIN_MS", "40")
			_ = os.Setenv("CUJU_SCORING_LATENCY_MAX_MS", "200")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then it should parse numeric values correctly", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(cfg, convey.ShouldNotBeNil)
				convey.So(cfg.EventQueueSize, convey.ShouldEqual, 500000)
				convey.So(cfg.WorkerCount, convey.ShouldEqual, 32)
				convey.So(cfg.DedupeSize, convey.ShouldEqual, 750000)
				convey.So(cfg.ScoringLatencyMinMS, convey.ShouldEqual, 40)
				convey.So(cfg.ScoringLatencyMaxMS, convey.ShouldEqual, 200)
			})
		})

		convey.Convey("When loading config with invalid numeric environment variables", func() {
			_ = os.Setenv("CUJU_QUEUE_SIZE", "invalid")
			_ = os.Setenv("CUJU_WORKER_COUNT", "not_a_number")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then it should return an error", func() {
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(cfg, convey.ShouldBeNil)
			})
		})
	})
}

func TestConfigLoaderEdgeCases(t *testing.T) {
	convey.Convey("Given config loader edge cases", t, func() {
		ctx := context.Background()

		convey.Convey("When loading config with very large values", func() {
			_ = os.Setenv("CUJU_QUEUE_SIZE", "1000000")
			_ = os.Setenv("CUJU_WORKER_COUNT", "1000")
			_ = os.Setenv("CUJU_DEDUPE_SIZE", "2000000")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then it should handle large values", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(cfg, convey.ShouldNotBeNil)
				convey.So(cfg.EventQueueSize, convey.ShouldEqual, 1000000)
				convey.So(cfg.WorkerCount, convey.ShouldEqual, 1000)
				convey.So(cfg.DedupeSize, convey.ShouldEqual, 2000000)
			})
		})

		convey.Convey("When loading config with zero values", func() {
			_ = os.Setenv("CUJU_QUEUE_SIZE", "0")
			_ = os.Setenv("CUJU_WORKER_COUNT", "0")
			_ = os.Setenv("CUJU_DEDUPE_SIZE", "0")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then it should handle zero values", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(cfg, convey.ShouldNotBeNil)
				convey.So(cfg.EventQueueSize, convey.ShouldEqual, 0)
				convey.So(cfg.WorkerCount, convey.ShouldEqual, 0)
				convey.So(cfg.DedupeSize, convey.ShouldEqual, 0)
			})
		})

		convey.Convey("When loading config with negative values", func() {
			_ = os.Setenv("CUJU_QUEUE_SIZE", "-100")
			_ = os.Setenv("CUJU_WORKER_COUNT", "-10")
			_ = os.Setenv("CUJU_DEDUPE_SIZE", "-200")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then it should handle negative values", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(cfg, convey.ShouldNotBeNil)
				convey.So(cfg.EventQueueSize, convey.ShouldEqual, -100)
				convey.So(cfg.WorkerCount, convey.ShouldEqual, -10)
				convey.So(cfg.DedupeSize, convey.ShouldEqual, -200)
			})
		})

		convey.Convey("When loading config with special characters in addr", func() {
			_ = os.Setenv("CUJU_ADDR", "localhost:8080")
			_ = os.Setenv("CUJU_ADDR", "0.0.0.0:9090")
			_ = os.Setenv("CUJU_ADDR", "[::1]:8080")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then it should handle various addr formats", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(cfg, convey.ShouldNotBeNil)
				convey.So(cfg.Addr, convey.ShouldEqual, "[::1]:8080") // Last one wins
			})
		})

		convey.Convey("When loading config with YAML file containing comments", func() {
			yamlContent := `
# This is a comment
addr: ":9090"  # Inline comment
queue_size: 300000
worker_count: 24
# Another comment
dedupe_size: 600000
`
			tmpFile := createTempConfigFile(yamlContent)
			defer func() { _ = os.Remove(tmpFile) }()

			_ = os.Setenv("CUJU_CONFIG", tmpFile)
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then it should parse YAML with comments", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(cfg, convey.ShouldNotBeNil)
				convey.So(cfg.Addr, convey.ShouldEqual, ":9090")
				convey.So(cfg.EventQueueSize, convey.ShouldEqual, 300000)
				convey.So(cfg.WorkerCount, convey.ShouldEqual, 24)
				convey.So(cfg.DedupeSize, convey.ShouldEqual, 600000)
			})
		})

		convey.Convey("When loading config with YAML file containing empty values", func() {
			yamlContent := `
addr: ""
queue_size: 
worker_count: 24
dedupe_size: 600000
`
			tmpFile := createTempConfigFile(yamlContent)
			defer func() { _ = os.Remove(tmpFile) }()

			_ = os.Setenv("CUJU_CONFIG", tmpFile)
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			convey.Convey("Then it should return validation error for empty addr", func() {
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(err.Error(), convey.ShouldContainSubstring, "addr must not be empty")
				convey.So(cfg, convey.ShouldBeNil)
			})
		})
	})
}

// Helper functions.

func clearConfigEnvVars() {
	envVars := []string{
		"CUJU_CONFIG",
		"CUJU_ADDR",
		"CUJU_QUEUE_SIZE",
		"CUJU_WORKER_COUNT",
		"CUJU_DEDUPE_SIZE",
		"CUJU_SCORING_LATENCY_MIN_MS",
		"CUJU_SCORING_LATENCY_MAX_MS",
	}
	for _, envVar := range envVars {
		_ = os.Unsetenv(envVar)
	}
}

func createTempConfigFile(content string) string {
	tmpFile, err := os.CreateTemp("", "cuju-config-*.yaml")
	if err != nil {
		panic(err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		panic(err)
	}

	if err := tmpFile.Close(); err != nil {
		panic(err)
	}

	return tmpFile.Name()
}
