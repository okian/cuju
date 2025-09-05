package logger

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLoggerInit(t *testing.T) {
	Convey("Given logger initialization", t, func() {
		Convey("When initializing the logger", func() {
			err := Init()
			defer func() {
				if err := Sync(); err != nil {
					t.Errorf("failed to sync logger: %v", err)
				}
			}()

			Convey("Then it should initialize successfully", func() {
				So(err, ShouldBeNil)
			})

			Convey("And the global logger should be available", func() {
				logger := Get()
				So(logger, ShouldNotBeNil)
			})
		})

		Convey("When initializing multiple times", func() {
			err1 := Init()
			err2 := Init()
			defer func() {
				if err := Sync(); err != nil {
					t.Errorf("failed to sync logger: %v", err)
				}
			}()

			Convey("Then both initializations should succeed", func() {
				So(err1, ShouldBeNil)
				So(err2, ShouldBeNil)
			})
		})
	})
}

func TestLoggerBasic(t *testing.T) {
	Convey("Given an initialized logger", t, func() {
		err := Init()
		So(err, ShouldBeNil)
		defer func() {
			if err := Sync(); err != nil {
				t.Errorf("failed to sync logger: %v", err)
			}
		}()

		logger := Get()
		So(logger, ShouldNotBeNil)

		ctx := context.Background()

		Convey("When logging info messages", func() {
			Convey("Then it should log without errors", func() {
				So(func() {
					logger.Info(ctx, "test message", String("key", "value"))
				}, ShouldNotPanic)
			})

			Convey("And it should handle multiple fields", func() {
				So(func() {
					logger.Info(ctx, "multi-field message",
						String("string_field", "value"),
						Int("int_field", 42),
						Float64("float_field", 3.14),
						Any("any_field", map[string]string{"nested": "value"}),
					)
				}, ShouldNotPanic)
			})
		})

		Convey("When logging error messages", func() {
			Convey("Then it should log without errors", func() {
				So(func() {
					logger.Error(ctx, "error message", Error(errors.New("test error")))
				}, ShouldNotPanic)
			})
		})

		Convey("When logging debug messages", func() {
			Convey("Then it should log without errors", func() {
				So(func() {
					logger.Debug(ctx, "debug message", String("debug_key", "debug_value"))
				}, ShouldNotPanic)
			})
		})

		Convey("When logging warn messages", func() {
			Convey("Then it should log without errors", func() {
				So(func() {
					logger.Warn(ctx, "warning message", String("warn_key", "warn_value"))
				}, ShouldNotPanic)
			})
		})
	})
}

func TestLoggerNamed(t *testing.T) {
	Convey("Given an initialized logger", t, func() {
		err := Init()
		So(err, ShouldBeNil)
		defer func() {
			if err := Sync(); err != nil {
				t.Errorf("failed to sync logger: %v", err)
			}
		}()

		Convey("When creating a named logger", func() {
			namedLogger := Named("test")
			So(namedLogger, ShouldNotBeNil)

			ctx := context.Background()

			Convey("Then it should log without errors", func() {
				So(func() {
					namedLogger.Info(ctx, "test message")
				}, ShouldNotPanic)
			})

			Convey("And it should handle nested naming", func() {
				nestedLogger := namedLogger.Named("nested")
				So(nestedLogger, ShouldNotBeNil)

				So(func() {
					nestedLogger.Info(ctx, "nested message")
				}, ShouldNotPanic)
			})
		})
	})
}

func TestLoggerFieldConstructors(t *testing.T) {
	Convey("Given field constructors", t, func() {
		Convey("When creating String field", func() {
			field := String("key", "value")

			Convey("Then it should have correct key and value", func() {
				So(field.Key, ShouldEqual, "key")
				So(field.Value, ShouldEqual, "value")
			})
		})

		Convey("When creating Int field", func() {
			field := Int("count", 42)

			Convey("Then it should have correct key and value", func() {
				So(field.Key, ShouldEqual, "count")
				So(field.Value, ShouldEqual, 42)
			})
		})

		Convey("When creating Float64 field", func() {
			field := Float64("ratio", 3.14)

			Convey("Then it should have correct key and value", func() {
				So(field.Key, ShouldEqual, "ratio")
				So(field.Value, ShouldEqual, 3.14)
			})
		})

		Convey("When creating Any field", func() {
			value := map[string]interface{}{"nested": "value"}
			field := Any("data", value)

			Convey("Then it should have correct key and value", func() {
				So(field.Key, ShouldEqual, "data")
				So(field.Value, ShouldEqual, value)
			})
		})

		Convey("When creating Error field", func() {
			err := errors.New("test error")
			field := Error(err)

			Convey("Then it should have correct key and value", func() {
				So(field.Key, ShouldEqual, "error")
				So(field.Value, ShouldEqual, err)
			})
		})
	})
}

func TestLoggerLevels(t *testing.T) {
	Convey("Given logger level management", t, func() {
		err := Init()
		So(err, ShouldBeNil)
		defer func() {
			if err := Sync(); err != nil {
				t.Errorf("failed to sync logger: %v", err)
			}
		}()

		Convey("When setting log level to debug", func() {
			SetLevel(slog.LevelDebug)

			Convey("Then debug messages should be logged", func() {
				logger := Get()
				ctx := context.Background()

				So(func() {
					logger.Debug(ctx, "debug message")
				}, ShouldNotPanic)
			})
		})

		Convey("When setting log level to info", func() {
			SetLevel(slog.LevelInfo)

			Convey("Then info messages should be logged", func() {
				logger := Get()
				ctx := context.Background()

				So(func() {
					logger.Info(ctx, "info message")
				}, ShouldNotPanic)
			})
		})

		Convey("When setting log level to warn", func() {
			SetLevel(slog.LevelWarn)

			Convey("Then warn messages should be logged", func() {
				logger := Get()
				ctx := context.Background()

				So(func() {
					logger.Warn(ctx, "warn message")
				}, ShouldNotPanic)
			})
		})

		Convey("When setting log level to error", func() {
			SetLevel(slog.LevelError)

			Convey("Then error messages should be logged", func() {
				logger := Get()
				ctx := context.Background()

				So(func() {
					logger.Error(ctx, "error message")
				}, ShouldNotPanic)
			})
		})
	})
}

func TestLoggerLevelString(t *testing.T) {
	Convey("Given logger level string parsing", t, func() {
		err := Init()
		So(err, ShouldBeNil)
		defer func() {
			if err := Sync(); err != nil {
				t.Errorf("failed to sync logger: %v", err)
			}
		}()

		Convey("When setting level to 'debug'", func() {
			err := SetLevelString("debug")

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When setting level to 'info'", func() {
			err := SetLevelString("info")

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When setting level to 'warn'", func() {
			err := SetLevelString("warn")

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When setting level to 'warning'", func() {
			err := SetLevelString("warning")

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When setting level to 'error'", func() {
			err := SetLevelString("error")

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When setting level to empty string", func() {
			err := SetLevelString("")

			Convey("Then it should default to info", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When setting level to invalid string", func() {
			err := SetLevelString("invalid")

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "unknown log level")
			})
		})

		Convey("When setting level with case variations", func() {
			testCases := []string{"DEBUG", "Info", "WARN", "Error", "WARNING"}

			for _, level := range testCases {
				Convey("And level is '"+level+"'", func() {
					err := SetLevelString(level)

					Convey("Then it should succeed", func() {
						So(err, ShouldBeNil)
					})
				})
			}
		})

		Convey("When setting level with whitespace", func() {
			err := SetLevelString("  debug  ")

			Convey("Then it should trim and succeed", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestLoggerSync(t *testing.T) {
	Convey("Given logger sync functionality", t, func() {
		Convey("When calling Sync", func() {
			err := Sync()

			Convey("Then it should succeed", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestLoggerGetWithoutInit(t *testing.T) {
	Convey("Given an uninitialized logger", t, func() {
		// Reset global logger to nil
		global = nil

		Convey("When calling Get", func() {
			Convey("Then it should panic", func() {
				So(func() {
					Get()
				}, ShouldPanic)
			})
		})

		Convey("When calling Named", func() {
			Convey("Then it should panic", func() {
				So(func() {
					Named("test")
				}, ShouldPanic)
			})
		})
	})
}

func TestLoggerContextHandling(t *testing.T) {
	Convey("Given an initialized logger", t, func() {
		err := Init()
		So(err, ShouldBeNil)
		defer func() {
			if err := Sync(); err != nil {
				t.Errorf("failed to sync logger: %v", err)
			}
		}()

		logger := Get()
		So(logger, ShouldNotBeNil)

		Convey("When logging with different contexts", func() {
			Convey("And using background context", func() {
				ctx := context.Background()

				Convey("Then it should log without errors", func() {
					So(func() {
						logger.Info(ctx, "background context message")
					}, ShouldNotPanic)
				})
			})

			Convey("And using TODO context", func() {
				ctx := context.TODO()

				Convey("Then it should log without errors", func() {
					So(func() {
						logger.Info(ctx, "TODO context message")
					}, ShouldNotPanic)
				})
			})

			Convey("And using context with values", func() {
				ctx := context.WithValue(context.Background(), "request_id", "12345")

				Convey("Then it should log without errors", func() {
					So(func() {
						logger.Info(ctx, "context with values message")
					}, ShouldNotPanic)
				})
			})

			Convey("And using cancelled context", func() {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				Convey("Then it should log without errors", func() {
					So(func() {
						logger.Info(ctx, "cancelled context message")
					}, ShouldNotPanic)
				})
			})
		})
	})
}

func TestLoggerFieldTypes(t *testing.T) {
	Convey("Given an initialized logger", t, func() {
		err := Init()
		So(err, ShouldBeNil)
		defer func() {
			if err := Sync(); err != nil {
				t.Errorf("failed to sync logger: %v", err)
			}
		}()

		logger := Get()
		So(logger, ShouldNotBeNil)

		ctx := context.Background()

		Convey("When logging with various field types", func() {
			Convey("And using string fields", func() {
				So(func() {
					logger.Info(ctx, "string fields", String("text", "hello world"))
				}, ShouldNotPanic)
			})

			Convey("And using int fields", func() {
				So(func() {
					logger.Info(ctx, "int fields", Int("count", 42), Int("negative", -10))
				}, ShouldNotPanic)
			})

			Convey("And using float64 fields", func() {
				So(func() {
					logger.Info(ctx, "float fields", Float64("pi", 3.14159), Float64("negative", -2.5))
				}, ShouldNotPanic)
			})

			Convey("And using any fields", func() {
				So(func() {
					logger.Info(ctx, "any fields",
						Any("slice", []string{"a", "b", "c"}),
						Any("map", map[string]int{"x": 1, "y": 2}),
						Any("struct", struct{ Name string }{"test"}),
					)
				}, ShouldNotPanic)
			})

			Convey("And using error fields", func() {
				So(func() {
					logger.Error(ctx, "error fields", Error(errors.New("test error")))
				}, ShouldNotPanic)
			})

			Convey("And using mixed field types", func() {
				So(func() {
					logger.Info(ctx, "mixed fields",
						String("name", "test"),
						Int("age", 25),
						Float64("score", 95.5),
						Any("metadata", map[string]interface{}{"active": true}),
						Error(nil),
					)
				}, ShouldNotPanic)
			})
		})
	})
}

func TestLoggerEdgeCases(t *testing.T) {
	Convey("Given an initialized logger", t, func() {
		err := Init()
		So(err, ShouldBeNil)
		defer func() {
			if err := Sync(); err != nil {
				t.Errorf("failed to sync logger: %v", err)
			}
		}()

		logger := Get()
		So(logger, ShouldNotBeNil)

		ctx := context.Background()

		Convey("When logging with edge cases", func() {
			Convey("And using empty message", func() {
				So(func() {
					logger.Info(ctx, "")
				}, ShouldNotPanic)
			})

			Convey("And using very long message", func() {
				longMessage := strings.Repeat("a", 10000)
				So(func() {
					logger.Info(ctx, longMessage)
				}, ShouldNotPanic)
			})

			Convey("And using empty field key", func() {
				So(func() {
					logger.Info(ctx, "empty key", String("", "value"))
				}, ShouldNotPanic)
			})

			Convey("And using nil field value", func() {
				So(func() {
					logger.Info(ctx, "nil value", Any("nil_field", nil))
				}, ShouldNotPanic)
			})

			Convey("And using many fields", func() {
				fields := make([]Field, 100)
				for i := 0; i < 100; i++ {
					fields[i] = String("key"+string(rune(i)), "value"+string(rune(i)))
				}

				So(func() {
					logger.Info(ctx, "many fields", fields...)
				}, ShouldNotPanic)
			})
		})
	})
}
