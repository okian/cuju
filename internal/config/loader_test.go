package config_test

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/okian/cuju/internal/config"
	. "github.com/smartystreets/goconvey/convey"
)

func TestConfigLoader(t *testing.T) {
	Convey("Given a config loader", t, func() {
		ctx := context.Background()

		Convey("When loading config with defaults only", func() {
			// Clear any existing environment variables
			clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then it should load successfully with defaults", func() {
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(cfg.Addr, ShouldEqual, ":9080")
				So(cfg.EventQueueSize, ShouldEqual, 200_000)
				So(cfg.WorkerCount, ShouldEqual, runtime.NumCPU()*20) // runtime.NumCPU() * 20
				So(cfg.DedupeSize, ShouldEqual, 500_000)
				So(cfg.ScoringLatencyMinMS, ShouldEqual, 80)
				So(cfg.ScoringLatencyMaxMS, ShouldEqual, 150)
			})
		})

		Convey("When loading config with environment variables", func() {
			// Set environment variables
			os.Setenv("CUJU_ADDR", ":8080")
			os.Setenv("CUJU_QUEUE_SIZE", "100000")
			os.Setenv("CUJU_WORKER_COUNT", "16")
			os.Setenv("CUJU_DEDUPE_SIZE", "250000")
			os.Setenv("CUJU_SCORING_LATENCY_MIN_MS", "50")
			os.Setenv("CUJU_SCORING_LATENCY_MAX_MS", "100")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then it should override defaults with env vars", func() {
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(cfg.Addr, ShouldEqual, ":8080")
				So(cfg.EventQueueSize, ShouldEqual, 100000)
				So(cfg.WorkerCount, ShouldEqual, 16)
				So(cfg.DedupeSize, ShouldEqual, 250000)
				So(cfg.ScoringLatencyMinMS, ShouldEqual, 50)
				So(cfg.ScoringLatencyMaxMS, ShouldEqual, 100)
			})
		})

		Convey("When loading config with YAML file", func() {
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
			defer os.Remove(tmpFile)

			// Set the config file path
			os.Setenv("CUJU_CONFIG", tmpFile)
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then it should load from YAML file", func() {
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(cfg.Addr, ShouldEqual, ":9090")
				So(cfg.EventQueueSize, ShouldEqual, 300000)
				So(cfg.WorkerCount, ShouldEqual, 24)
				So(cfg.DedupeSize, ShouldEqual, 600000)
				So(cfg.ScoringLatencyMinMS, ShouldEqual, 60)
				So(cfg.ScoringLatencyMaxMS, ShouldEqual, 120)
			})
		})

		Convey("When loading config with both file and environment variables", func() {
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
			defer os.Remove(tmpFile)

			// Set both file and environment variables
			os.Setenv("CUJU_CONFIG", tmpFile)
			os.Setenv("CUJU_ADDR", ":8080")      // This should override the file
			os.Setenv("CUJU_WORKER_COUNT", "32") // This should override the file
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then environment variables should override file values", func() {
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(cfg.Addr, ShouldEqual, ":8080")            // Overridden by env
				So(cfg.EventQueueSize, ShouldEqual, 300000)   // From file
				So(cfg.WorkerCount, ShouldEqual, 32)          // Overridden by env
				So(cfg.DedupeSize, ShouldEqual, 600000)       // From file
				So(cfg.ScoringLatencyMinMS, ShouldEqual, 60)  // From file
				So(cfg.ScoringLatencyMaxMS, ShouldEqual, 120) // From file
			})
		})

		Convey("When loading config with invalid YAML file", func() {
			// Create an invalid YAML file
			invalidYaml := `invalid: yaml: content: [`
			tmpFile := createTempConfigFile(invalidYaml)
			defer os.Remove(tmpFile)

			os.Setenv("CUJU_CONFIG", tmpFile)
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(cfg, ShouldBeNil)
			})
		})

		Convey("When loading config with non-existent file", func() {
			os.Setenv("CUJU_CONFIG", "/non/existent/file.yaml")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(cfg, ShouldBeNil)
			})
		})

		Convey("When loading config with empty addr", func() {
			os.Setenv("CUJU_ADDR", "")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then it should return a validation error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "addr must not be empty")
				So(cfg, ShouldBeNil)
			})
		})

		Convey("When loading config with partial YAML file", func() {
			// Create a YAML file with only some fields
			yamlContent := `
addr: ":9090"
worker_count: 16
`
			tmpFile := createTempConfigFile(yamlContent)
			defer os.Remove(tmpFile)

			os.Setenv("CUJU_CONFIG", tmpFile)
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then it should merge with defaults for missing fields", func() {
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(cfg.Addr, ShouldEqual, ":9090")            // From file
				So(cfg.WorkerCount, ShouldEqual, 16)          // From file
				So(cfg.EventQueueSize, ShouldEqual, 200_000)  // From defaults
				So(cfg.DedupeSize, ShouldEqual, 500_000)      // From defaults
				So(cfg.ScoringLatencyMinMS, ShouldEqual, 80)  // From defaults
				So(cfg.ScoringLatencyMaxMS, ShouldEqual, 150) // From defaults
			})
		})

		Convey("When loading config with environment variables using different cases", func() {
			// Test case insensitivity
			os.Setenv("CUJU_ADDR", ":8080")
			os.Setenv("CUJU_QUEUE_SIZE", "100000") // uppercase prefix
			os.Setenv("CUJU_WORKER_COUNT", "16")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then it should handle case insensitive environment variables", func() {
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(cfg.Addr, ShouldEqual, ":8080")
				So(cfg.EventQueueSize, ShouldEqual, 100000)
				So(cfg.WorkerCount, ShouldEqual, 16)
			})
		})

		Convey("When loading config with numeric environment variables", func() {
			os.Setenv("CUJU_QUEUE_SIZE", "500000")
			os.Setenv("CUJU_WORKER_COUNT", "32")
			os.Setenv("CUJU_DEDUPE_SIZE", "750000")
			os.Setenv("CUJU_SCORING_LATENCY_MIN_MS", "40")
			os.Setenv("CUJU_SCORING_LATENCY_MAX_MS", "200")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then it should parse numeric values correctly", func() {
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(cfg.EventQueueSize, ShouldEqual, 500000)
				So(cfg.WorkerCount, ShouldEqual, 32)
				So(cfg.DedupeSize, ShouldEqual, 750000)
				So(cfg.ScoringLatencyMinMS, ShouldEqual, 40)
				So(cfg.ScoringLatencyMaxMS, ShouldEqual, 200)
			})
		})

		Convey("When loading config with invalid numeric environment variables", func() {
			os.Setenv("CUJU_QUEUE_SIZE", "invalid")
			os.Setenv("CUJU_WORKER_COUNT", "not_a_number")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(cfg, ShouldBeNil)
			})
		})
	})
}

func TestConfigLoaderEdgeCases(t *testing.T) {
	Convey("Given config loader edge cases", t, func() {
		ctx := context.Background()

		Convey("When loading config with very large values", func() {
			os.Setenv("CUJU_QUEUE_SIZE", "1000000")
			os.Setenv("CUJU_WORKER_COUNT", "1000")
			os.Setenv("CUJU_DEDUPE_SIZE", "2000000")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then it should handle large values", func() {
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(cfg.EventQueueSize, ShouldEqual, 1000000)
				So(cfg.WorkerCount, ShouldEqual, 1000)
				So(cfg.DedupeSize, ShouldEqual, 2000000)
			})
		})

		Convey("When loading config with zero values", func() {
			os.Setenv("CUJU_QUEUE_SIZE", "0")
			os.Setenv("CUJU_WORKER_COUNT", "0")
			os.Setenv("CUJU_DEDUPE_SIZE", "0")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then it should handle zero values", func() {
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(cfg.EventQueueSize, ShouldEqual, 0)
				So(cfg.WorkerCount, ShouldEqual, 0)
				So(cfg.DedupeSize, ShouldEqual, 0)
			})
		})

		Convey("When loading config with negative values", func() {
			os.Setenv("CUJU_QUEUE_SIZE", "-100")
			os.Setenv("CUJU_WORKER_COUNT", "-10")
			os.Setenv("CUJU_DEDUPE_SIZE", "-200")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then it should handle negative values", func() {
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(cfg.EventQueueSize, ShouldEqual, -100)
				So(cfg.WorkerCount, ShouldEqual, -10)
				So(cfg.DedupeSize, ShouldEqual, -200)
			})
		})

		Convey("When loading config with special characters in addr", func() {
			os.Setenv("CUJU_ADDR", "localhost:8080")
			os.Setenv("CUJU_ADDR", "0.0.0.0:9090")
			os.Setenv("CUJU_ADDR", "[::1]:8080")
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then it should handle various addr formats", func() {
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(cfg.Addr, ShouldEqual, "[::1]:8080") // Last one wins
			})
		})

		Convey("When loading config with YAML file containing comments", func() {
			yamlContent := `
# This is a comment
addr: ":9090"  # Inline comment
queue_size: 300000
worker_count: 24
# Another comment
dedupe_size: 600000
`
			tmpFile := createTempConfigFile(yamlContent)
			defer os.Remove(tmpFile)

			os.Setenv("CUJU_CONFIG", tmpFile)
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then it should parse YAML with comments", func() {
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(cfg.Addr, ShouldEqual, ":9090")
				So(cfg.EventQueueSize, ShouldEqual, 300000)
				So(cfg.WorkerCount, ShouldEqual, 24)
				So(cfg.DedupeSize, ShouldEqual, 600000)
			})
		})

		Convey("When loading config with YAML file containing empty values", func() {
			yamlContent := `
addr: ""
queue_size: 
worker_count: 24
dedupe_size: 600000
`
			tmpFile := createTempConfigFile(yamlContent)
			defer os.Remove(tmpFile)

			os.Setenv("CUJU_CONFIG", tmpFile)
			defer clearConfigEnvVars()

			cfg, err := config.Load(ctx)

			Convey("Then it should return validation error for empty addr", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "addr must not be empty")
				So(cfg, ShouldBeNil)
			})
		})
	})
}

// Helper functions

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
		os.Unsetenv(envVar)
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
