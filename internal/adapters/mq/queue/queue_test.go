package queue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/okian/cuju/internal/domain/model"
)

func TestInMemoryQueue_BasicOperations(t *testing.T) {
	q := NewInMemoryQueue(WithCapacity(2))
	ctx := context.Background()

	// Test empty queue
	if l := q.Len(ctx); l != 0 {
		t.Errorf("expected length 0, got %d", l)
	}

	// Test enqueue
	event1 := model.Event{EventID: "event1", TalentID: "talent1", RawMetric: 100.0, Skill: "test"}
	if !q.Enqueue(ctx, event1) {
		t.Error("expected enqueue to succeed")
	}

	if l := q.Len(ctx); l != 1 {
		t.Errorf("expected length 1, got %d", l)
	}

	// Test dequeue
	eventChan := q.Dequeue(ctx)
	event := <-eventChan
	if event.EventID != "event1" {
		t.Errorf("expected event1, got %v", event.EventID)
	}

	if l := q.Len(ctx); l != 0 {
		t.Errorf("expected length 0, got %d", l)
	}
}

func TestInMemoryQueue_Capacity(t *testing.T) {
	q := NewInMemoryQueue(WithCapacity(2))
	ctx := context.Background()

	// Fill the queue
	event1 := model.Event{EventID: "event1", TalentID: "talent1", RawMetric: 100.0, Skill: "test"}
	event2 := model.Event{EventID: "event2", TalentID: "talent2", RawMetric: 200.0, Skill: "test"}
	event3 := model.Event{EventID: "event3", TalentID: "talent3", RawMetric: 300.0, Skill: "test"}

	if !q.Enqueue(ctx, event1) {
		t.Error("expected enqueue to succeed")
	}
	if !q.Enqueue(ctx, event2) {
		t.Error("expected enqueue to succeed")
	}

	// Try to enqueue when full
	if q.Enqueue(ctx, event3) {
		t.Error("expected enqueue to fail when full")
	}

	if l := q.Len(ctx); l != 2 {
		t.Errorf("expected length 2, got %d", l)
	}
}

func TestInMemoryQueue_ConcurrentAccess(t *testing.T) {
	q := NewInMemoryQueue(WithCapacity(100))
	ctx := context.Background()
	numGoroutines := 10
	numEvents := 100

	// Start producer goroutines
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numEvents; j++ {
				event := model.Event{
					EventID:   fmt.Sprintf("event%d_%d", id, j),
					TalentID:  fmt.Sprintf("talent%d", id),
					RawMetric: float64(j),
					Skill:     "test",
				}
				for !q.Enqueue(ctx, event) {
					time.Sleep(time.Millisecond)
				}
			}
			done <- true
		}(i)
	}

	// Start consumer goroutines
	consumed := make(chan string, numGoroutines*numEvents)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			eventChan := q.Dequeue(ctx)
			for event := range eventChan {
				consumed <- event.EventID
			}
		}()
	}

	// Wait for producers to finish
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Wait a bit for consumers to process
	time.Sleep(100 * time.Millisecond)

	// Check final queue length
	if l := q.Len(ctx); l != 0 {
		t.Errorf("expected final length 0, got %d", l)
	}
}

func TestInMemoryQueue_GracefulShutdown(t *testing.T) {
	q := NewInMemoryQueue(WithCapacity(10))
	ctx := context.Background()

	// Enqueue some events
	event1 := model.Event{EventID: "event1", TalentID: "talent1", RawMetric: 100.0, Skill: "test"}
	event2 := model.Event{EventID: "event2", TalentID: "talent2", RawMetric: 200.0, Skill: "test"}

	if !q.Enqueue(ctx, event1) {
		t.Error("expected enqueue to succeed")
	}
	if !q.Enqueue(ctx, event2) {
		t.Error("expected enqueue to succeed")
	}

	// Check initial state
	if q.IsClosed() {
		t.Error("expected queue to be open initially")
	}

	// Close the queue
	if err := q.Close(); err != nil {
		t.Errorf("expected close to succeed, got error: %v", err)
	}

	// Check closed state
	if !q.IsClosed() {
		t.Error("expected queue to be closed after Close()")
	}

	// Try to enqueue after closing (should fail)
	if q.Enqueue(ctx, event1) {
		t.Error("expected enqueue to fail after closing")
	}

	// Dequeue channel should be closed
	eventChan := q.Dequeue(ctx)

	// Wait for channel to be closed
	timeout := time.After(100 * time.Millisecond)
	for {
		select {
		case _, ok := <-eventChan:
			if !ok {
				// Channel is closed, which is expected
				goto channelClosed
			}
		case <-timeout:
			t.Error("expected dequeue channel to be closed within timeout")
			return
		}
	}
channelClosed:

	// Close again should not error
	if err := q.Close(); err != nil {
		t.Errorf("expected second close to succeed, got error: %v", err)
	}
}
