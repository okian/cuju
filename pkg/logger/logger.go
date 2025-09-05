// Package logger provides a simple, clean logging interface.
package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Constants for logging operations.
const (
	callerSkipFrames = 2 // Skip frames: getCaller -> logging method -> actual caller
)

// Logger defines the logging interface.
type Logger interface {
	// Context-aware variants
	Info(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)
	Debug(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Fatal(ctx context.Context, msg string, fields ...Field)

	Named(name string) Logger
}

// Field represents a key-value pair for structured logging.
type Field struct {
	Key   string
	Value interface{}
}

// Field constructors.
func String(key, val string) Field          { return Field{Key: key, Value: val} }
func Int(key string, val int) Field         { return Field{Key: key, Value: val} }
func Float64(key string, val float64) Field { return Field{Key: key, Value: val} }
func Any(key string, val interface{}) Field { return Field{Key: key, Value: val} }
func Error(err error) Field                 { return Field{Key: "error", Value: err} }

// slogLogger implements Logger using slog.
type slogLogger struct {
	Logger *slog.Logger
}

func (l *slogLogger) Named(name string) Logger {
	return &slogLogger{Logger: l.Logger.WithGroup(name)}
}

func (l *slogLogger) Info(ctx context.Context, msg string, fields ...Field) {
	caller := getCaller()
	fields = append(fields, String("source", caller))
	l.Logger.LogAttrs(ctx, slog.LevelInfo, msg, convertFields(fields)...)
}

func (l *slogLogger) Error(ctx context.Context, msg string, fields ...Field) {
	caller := getCaller()
	fields = append(fields, String("source", caller))
	l.Logger.LogAttrs(ctx, slog.LevelError, msg, convertFields(fields)...)
}

func (l *slogLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	caller := getCaller()
	fields = append(fields, String("source", caller))
	l.Logger.LogAttrs(ctx, slog.LevelDebug, msg, convertFields(fields)...)
}

func (l *slogLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	caller := getCaller()
	fields = append(fields, String("source", caller))
	l.Logger.LogAttrs(ctx, slog.LevelWarn, msg, convertFields(fields)...)
}

func (l *slogLogger) Fatal(ctx context.Context, msg string, fields ...Field) {
	caller := getCaller()
	fields = append(fields, String("source", caller))
	l.Logger.LogAttrs(ctx, slog.LevelError, msg, convertFields(fields)...)
	os.Exit(1)
}

// convertFields converts our Field type to slog.Attr.
func convertFields(fields []Field) []slog.Attr {
	attrs := make([]slog.Attr, len(fields))
	for i, f := range fields {
		attrs[i] = slog.Any(f.Key, f.Value)
	}
	return attrs
}

var global Logger
var levelVar slog.LevelVar

// Init initializes the global logger.
func Init() error {
	// Default to info; can be changed with SetLevel*/SetLevelString.
	levelVar.Set(slog.LevelInfo)
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: &levelVar, AddSource: false})
	logger := slog.New(h)
	global = &slogLogger{Logger: logger}
	return nil
}

// getCaller returns the caller location in format relative/path/file.go:line (IDE-friendly).
func getCaller() string {
	// Skip 2 frames: getCaller -> logging method -> actual caller
	_, file, line, ok := runtime.Caller(callerSkipFrames)
	if !ok {
		return "unknown:0"
	}

	// Get current working directory to make path relative
	cwd, err := os.Getwd()
	if err != nil {
		// Fallback to just filename if we can't get working directory
		fileName := filepath.Base(file)
		return fmt.Sprintf("%s:%d", fileName, line)
	}

	// Make the file path relative to the working directory
	relPath, err := filepath.Rel(cwd, file)
	if err != nil {
		// Fallback to just filename if relative path fails
		fileName := filepath.Base(file)
		return fmt.Sprintf("%s:%d", fileName, line)
	}

	return fmt.Sprintf("%s:%d", relPath, line)
}

// Get returns the global logger.
func Get() Logger {
	if global == nil {
		// Don't auto-initialize with production settings
		// The logger should be explicitly initialized by the application
		panic("logger not initialized. Call logging.init() first")
	}
	return global
}

// Named creates a named logger.
func Named(name string) Logger {
	return Get().Named(name)
}

// Sync flushes buffered log entries.
func Sync() error {
	// slog does not buffer; nothing to flush
	return nil
}

// SetLevel updates the current logging level for the global logger handler.
func SetLevel(level slog.Level) { levelVar.Set(level) }

// SetLevelString parses and sets the logging level.
// Accepts: debug, info, warn/warning, error (case-insensitive).
func SetLevelString(level string) error {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		SetLevel(slog.LevelDebug)
	case "", "info":
		SetLevel(slog.LevelInfo)
	case "warn", "warning":
		SetLevel(slog.LevelWarn)
	case "error":
		SetLevel(slog.LevelError)
	default:
		return fmt.Errorf("unknown log level: %s", level)
	}
	return nil
}
