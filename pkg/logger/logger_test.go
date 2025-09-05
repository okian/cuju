package logger

import (
	"context"
	"testing"
)

func TestLoggerInit(t *testing.T) {
	// Test development mode
	err := Init()
	if err != nil {
		t.Fatalf("failed to initialize development logger: %v", err)
	}
	defer func() {
		if err := Sync(); err != nil {
			t.Errorf("failed to sync logger: %v", err)
		}
	}()

	logger := Get()
	if logger == nil {
		t.Fatal("logger is nil after initialization")
	}

	// Test production mode
	err = Init()
	if err != nil {
		t.Fatalf("failed to initialize production logger: %v", err)
	}
	defer func() {
		if err := Sync(); err != nil {
			t.Errorf("failed to sync logger: %v", err)
		}
	}()

	logger = Get()
	if logger == nil {
		t.Fatal("logger is nil after initialization")
	}
}

// Basic logging test (slog-backed; no Sugar)
func TestLoggerBasic(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("failed to initialize logger: %v", err)
	}
	defer func() {
		if err := Sync(); err != nil {
			t.Errorf("failed to sync logger: %v", err)
		}
	}()

	logger := Get()
	if logger == nil {
		t.Fatal("logger is nil")
	}

	ctx := context.Background()
	logger.Info(ctx, "test message", String("k", "v"))
}

func TestLoggerNamed(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("failed to initialize logger: %v", err)
	}
	defer func() {
		if err := Sync(); err != nil {
			t.Errorf("failed to sync logger: %v", err)
		}
	}()

	namedLogger := Named("test")
	if namedLogger == nil {
		t.Fatal("named logger is nil")
	}

	ctx := context.Background()
	namedLogger.Info(ctx, "test message")
}
