package logger

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

// contextKey is a custom type for context keys to avoid using basic string type.
type contextKey string

func TestLoggerInit(t *testing.T) {
	convey.Convey("Given logger initialization", t, func() {
		convey.Convey("When initializing the logger", func() {
			err := Init()
			defer func() {
				if err := Sync(); err != nil {
					t.Errorf("failed to sync logger: %v", err)
				}
			}()

			convey.Convey("Then it should initialize successfully", func() {
				convey.So(err, convey.ShouldBeNil)
			})

			convey.Convey("And the global logger should be available", func() {
				logger := Get()
				convey.So(logger, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When initializing multiple times", func() {
			err1 := Init()
			err2 := Init()
			defer func() {
				if err := Sync(); err != nil {
					t.Errorf("failed to sync logger: %v", err)
				}
			}()

			convey.Convey("Then both initializations should succeed", func() {
				convey.So(err1, convey.ShouldBeNil)
				convey.So(err2, convey.ShouldBeNil)
			})
		})
	})
}

func TestLoggerBasic(t *testing.T) {
	convey.Convey("Given an initialized logger", t, func() {
		err := Init()
		convey.So(err, convey.ShouldBeNil)
		defer func() {
			if err := Sync(); err != nil {
				t.Errorf("failed to sync logger: %v", err)
			}
		}()

		logger := Get()
		convey.So(logger, convey.ShouldNotBeNil)

		ctx := context.Background()

		convey.Convey("When logging info messages", func() {
			convey.Convey("Then it should log without errors", func() {
				convey.So(func() {
					logger.Info(ctx, "test message", String("key", "value"))
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should handle multiple fields", func() {
				convey.So(func() {
					logger.Info(ctx, "multi-field message",
						String("string_field", "value"),
						Int("int_field", 42),
						Float64("float_field", 3.14),
						Any("any_field", map[string]string{"nested": "value"}),
					)
				}, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When logging error messages", func() {
			convey.Convey("Then it should log without errors", func() {
				convey.So(func() {
					logger.Error(ctx, "error message", Error(errors.New("test error")))
				}, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When logging debug messages", func() {
			convey.Convey("Then it should log without errors", func() {
				convey.So(func() {
					logger.Debug(ctx, "debug message", String("debug_key", "debug_value"))
				}, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When logging warn messages", func() {
			convey.Convey("Then it should log without errors", func() {
				convey.So(func() {
					logger.Warn(ctx, "warning message", String("warn_key", "warn_value"))
				}, convey.ShouldNotPanic)
			})
		})
	})
}

func TestLoggerNamed(t *testing.T) {
	convey.Convey("Given an initialized logger", t, func() {
		err := Init()
		convey.So(err, convey.ShouldBeNil)
		defer func() {
			if err := Sync(); err != nil {
				t.Errorf("failed to sync logger: %v", err)
			}
		}()

		convey.Convey("When creating a named logger", func() {
			namedLogger := Named("test")
			convey.So(namedLogger, convey.ShouldNotBeNil)

			ctx := context.Background()

			convey.Convey("Then it should log without errors", func() {
				convey.So(func() {
					namedLogger.Info(ctx, "test message")
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And it should handle nested naming", func() {
				nestedLogger := namedLogger.Named("nested")
				convey.So(nestedLogger, convey.ShouldNotBeNil)

				convey.So(func() {
					nestedLogger.Info(ctx, "nested message")
				}, convey.ShouldNotPanic)
			})
		})
	})
}

func TestLoggerFieldConstructors(t *testing.T) {
	convey.Convey("Given field constructors", t, func() {
		convey.Convey("When creating String field", func() {
			field := String("key", "value")

			convey.Convey("Then it should have correct key and value", func() {
				convey.So(field.Key, convey.ShouldEqual, "key")
				convey.So(field.Value, convey.ShouldEqual, "value")
			})
		})

		convey.Convey("When creating Int field", func() {
			field := Int("count", 42)

			convey.Convey("Then it should have correct key and value", func() {
				convey.So(field.Key, convey.ShouldEqual, "count")
				convey.So(field.Value, convey.ShouldEqual, 42)
			})
		})

		convey.Convey("When creating Float64 field", func() {
			field := Float64("ratio", 3.14)

			convey.Convey("Then it should have correct key and value", func() {
				convey.So(field.Key, convey.ShouldEqual, "ratio")
				convey.So(field.Value, convey.ShouldEqual, 3.14)
			})
		})

		convey.Convey("When creating Any field", func() {
			value := map[string]interface{}{"nested": "value"}
			field := Any("data", value)

			convey.Convey("Then it should have correct key and value", func() {
				convey.So(field.Key, convey.ShouldEqual, "data")
				convey.So(field.Value, convey.ShouldEqual, value)
			})
		})

		convey.Convey("When creating Error field", func() {
			err := errors.New("test error")
			field := Error(err)

			convey.Convey("Then it should have correct key and value", func() {
				convey.So(field.Key, convey.ShouldEqual, "error")
				convey.So(field.Value, convey.ShouldEqual, err)
			})
		})
	})
}

func TestLoggerLevels(t *testing.T) {
	convey.Convey("Given logger level management", t, func() {
		err := Init()
		convey.So(err, convey.ShouldBeNil)
		defer func() {
			if err := Sync(); err != nil {
				t.Errorf("failed to sync logger: %v", err)
			}
		}()

		convey.Convey("When setting log level to debug", func() {
			SetLevel(slog.LevelDebug)

			convey.Convey("Then debug messages should be logged", func() {
				logger := Get()
				ctx := context.Background()

				convey.So(func() {
					logger.Debug(ctx, "debug message")
				}, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When setting log level to info", func() {
			SetLevel(slog.LevelInfo)

			convey.Convey("Then info messages should be logged", func() {
				logger := Get()
				ctx := context.Background()

				convey.So(func() {
					logger.Info(ctx, "info message")
				}, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When setting log level to warn", func() {
			SetLevel(slog.LevelWarn)

			convey.Convey("Then warn messages should be logged", func() {
				logger := Get()
				ctx := context.Background()

				convey.So(func() {
					logger.Warn(ctx, "warn message")
				}, convey.ShouldNotPanic)
			})
		})

		convey.Convey("When setting log level to error", func() {
			SetLevel(slog.LevelError)

			convey.Convey("Then error messages should be logged", func() {
				logger := Get()
				ctx := context.Background()

				convey.So(func() {
					logger.Error(ctx, "error message")
				}, convey.ShouldNotPanic)
			})
		})
	})
}

func TestLoggerLevelString(t *testing.T) {
	convey.Convey("Given logger level string parsing", t, func() {
		err := Init()
		convey.So(err, convey.ShouldBeNil)
		defer func() {
			if err := Sync(); err != nil {
				t.Errorf("failed to sync logger: %v", err)
			}
		}()

		convey.Convey("When setting level to 'debug'", func() {
			err := SetLevelString("debug")

			convey.Convey("Then it should succeed", func() {
				convey.So(err, convey.ShouldBeNil)
			})
		})

		convey.Convey("When setting level to 'info'", func() {
			err := SetLevelString("info")

			convey.Convey("Then it should succeed", func() {
				convey.So(err, convey.ShouldBeNil)
			})
		})

		convey.Convey("When setting level to 'warn'", func() {
			err := SetLevelString("warn")

			convey.Convey("Then it should succeed", func() {
				convey.So(err, convey.ShouldBeNil)
			})
		})

		convey.Convey("When setting level to 'warning'", func() {
			err := SetLevelString("warning")

			convey.Convey("Then it should succeed", func() {
				convey.So(err, convey.ShouldBeNil)
			})
		})

		convey.Convey("When setting level to 'error'", func() {
			err := SetLevelString("error")

			convey.Convey("Then it should succeed", func() {
				convey.So(err, convey.ShouldBeNil)
			})
		})

		convey.Convey("When setting level to empty string", func() {
			err := SetLevelString("")

			convey.Convey("Then it should default to info", func() {
				convey.So(err, convey.ShouldBeNil)
			})
		})

		convey.Convey("When setting level to invalid string", func() {
			err := SetLevelString("invalid")

			convey.Convey("Then it should return an error", func() {
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(err.Error(), convey.ShouldContainSubstring, "unknown log level")
			})
		})

		convey.Convey("When setting level with case variations", func() {
			testCases := []string{"DEBUG", "Info", "WARN", "Error", "WARNING"}

			for _, level := range testCases {
				convey.Convey("And level is '"+level+"'", func() {
					err := SetLevelString(level)

					convey.Convey("Then it should succeed", func() {
						convey.So(err, convey.ShouldBeNil)
					})
				})
			}
		})

		convey.Convey("When setting level with whitespace", func() {
			err := SetLevelString("  debug  ")

			convey.Convey("Then it should trim and succeed", func() {
				convey.So(err, convey.ShouldBeNil)
			})
		})
	})
}

func TestLoggerSync(t *testing.T) {
	convey.Convey("Given logger sync functionality", t, func() {
		convey.Convey("When calling Sync", func() {
			err := Sync()

			convey.Convey("Then it should succeed", func() {
				convey.So(err, convey.ShouldBeNil)
			})
		})
	})
}

func TestLoggerGetWithoutInit(t *testing.T) {
	convey.Convey("Given an uninitialized logger", t, func() {
		// Reset global logger to nil
		global = nil

		convey.Convey("When calling Get", func() {
			convey.Convey("Then it should panic", func() {
				convey.So(func() {
					Get()
				}, convey.ShouldPanic)
			})
		})

		convey.Convey("When calling Named", func() {
			convey.Convey("Then it should panic", func() {
				convey.So(func() {
					Named("test")
				}, convey.ShouldPanic)
			})
		})
	})
}

func TestLoggerContextHandling(t *testing.T) {
	convey.Convey("Given an initialized logger", t, func() {
		err := Init()
		convey.So(err, convey.ShouldBeNil)
		defer func() {
			if err := Sync(); err != nil {
				t.Errorf("failed to sync logger: %v", err)
			}
		}()

		logger := Get()
		convey.So(logger, convey.ShouldNotBeNil)

		convey.Convey("When logging with different contexts", func() {
			convey.Convey("And using background context", func() {
				ctx := context.Background()

				convey.Convey("Then it should log without errors", func() {
					convey.So(func() {
						logger.Info(ctx, "background context message")
					}, convey.ShouldNotPanic)
				})
			})

			convey.Convey("And using TODO context", func() {
				ctx := context.TODO()

				convey.Convey("Then it should log without errors", func() {
					convey.So(func() {
						logger.Info(ctx, "TODO context message")
					}, convey.ShouldNotPanic)
				})
			})

			convey.Convey("And using context with values", func() {
				ctx := context.WithValue(context.Background(), contextKey("request_id"), "12345")

				convey.Convey("Then it should log without errors", func() {
					convey.So(func() {
						logger.Info(ctx, "context with values message")
					}, convey.ShouldNotPanic)
				})
			})

			convey.Convey("And using cancelled context", func() {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				convey.Convey("Then it should log without errors", func() {
					convey.So(func() {
						logger.Info(ctx, "cancelled context message")
					}, convey.ShouldNotPanic)
				})
			})
		})
	})
}

func TestLoggerFieldTypes(t *testing.T) {
	convey.Convey("Given an initialized logger", t, func() {
		err := Init()
		convey.So(err, convey.ShouldBeNil)
		defer func() {
			if err := Sync(); err != nil {
				t.Errorf("failed to sync logger: %v", err)
			}
		}()

		logger := Get()
		convey.So(logger, convey.ShouldNotBeNil)

		ctx := context.Background()

		convey.Convey("When logging with various field types", func() {
			convey.Convey("And using string fields", func() {
				convey.So(func() {
					logger.Info(ctx, "string fields", String("text", "hello world"))
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And using int fields", func() {
				convey.So(func() {
					logger.Info(ctx, "int fields", Int("count", 42), Int("negative", -10))
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And using float64 fields", func() {
				convey.So(func() {
					logger.Info(ctx, "float fields", Float64("pi", 3.14159), Float64("negative", -2.5))
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And using any fields", func() {
				convey.So(func() {
					logger.Info(ctx, "any fields",
						Any("slice", []string{"a", "b", "c"}),
						Any("map", map[string]int{"x": 1, "y": 2}),
						Any("struct", struct{ Name string }{"test"}),
					)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And using error fields", func() {
				convey.So(func() {
					logger.Error(ctx, "error fields", Error(errors.New("test error")))
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And using mixed field types", func() {
				convey.So(func() {
					logger.Info(ctx, "mixed fields",
						String("name", "test"),
						Int("age", 25),
						Float64("score", 95.5),
						Any("metadata", map[string]interface{}{"active": true}),
						Error(nil),
					)
				}, convey.ShouldNotPanic)
			})
		})
	})
}

func TestLoggerEdgeCases(t *testing.T) {
	convey.Convey("Given an initialized logger", t, func() {
		err := Init()
		convey.So(err, convey.ShouldBeNil)
		defer func() {
			if err := Sync(); err != nil {
				t.Errorf("failed to sync logger: %v", err)
			}
		}()

		logger := Get()
		convey.So(logger, convey.ShouldNotBeNil)

		ctx := context.Background()

		convey.Convey("When logging with edge cases", func() {
			convey.Convey("And using empty message", func() {
				convey.So(func() {
					logger.Info(ctx, "")
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And using very long message", func() {
				longMessage := strings.Repeat("a", 10000)
				convey.So(func() {
					logger.Info(ctx, longMessage)
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And using empty field key", func() {
				convey.So(func() {
					logger.Info(ctx, "empty key", String("", "value"))
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And using nil field value", func() {
				convey.So(func() {
					logger.Info(ctx, "nil value", Any("nil_field", nil))
				}, convey.ShouldNotPanic)
			})

			convey.Convey("And using many fields", func() {
				fields := make([]Field, 100)
				for i := 0; i < 100; i++ {
					fields[i] = String("key"+string(rune(i)), "value"+string(rune(i)))
				}

				convey.So(func() {
					logger.Info(ctx, "many fields", fields...)
				}, convey.ShouldNotPanic)
			})
		})
	})
}
