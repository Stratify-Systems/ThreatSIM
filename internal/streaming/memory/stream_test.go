package memory

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
)

func TestStream_PublishAndSubscribe(t *testing.T) {
	stream := NewStream()

	var count int32

	// Subscribe in background (Subscribe blocks until context cancel)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go stream.Subscribe(ctx, "test-topic", func(ctx context.Context, event core.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	// Small delay to let Subscribe register the handler
	time.Sleep(10 * time.Millisecond)

	// Publish events
	for i := 0; i < 5; i++ {
		err := stream.Publish(context.Background(), "test-topic", core.Event{
			ID:        "evt-" + string(rune('A'+i)),
			Type:      "test_event",
			SourceIP:  "10.0.0.1",
			Timestamp: time.Now(),
		})
		if err != nil {
			t.Fatalf("Publish failed: %v", err)
		}
	}

	total := atomic.LoadInt32(&count)
	if total != 5 {
		t.Errorf("expected 5 events received, got %d", total)
	}
}

func TestStream_PublishToUnsubscribedTopic(t *testing.T) {
	stream := NewStream()

	// Publish to a topic with no subscribers — should not error
	err := stream.Publish(context.Background(), "empty-topic", core.Event{
		ID:   "evt-1",
		Type: "test",
	})
	if err != nil {
		t.Fatalf("expected no error publishing to empty topic, got: %v", err)
	}
}

func TestStream_MultipleSubscribers(t *testing.T) {
	stream := NewStream()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var countA, countB int32

	go stream.Subscribe(ctx, "multi", func(ctx context.Context, event core.Event) error {
		atomic.AddInt32(&countA, 1)
		return nil
	})

	// Small delay between subscribes to ensure both register
	time.Sleep(10 * time.Millisecond)

	go stream.Subscribe(ctx, "multi", func(ctx context.Context, event core.Event) error {
		atomic.AddInt32(&countB, 1)
		return nil
	})

	time.Sleep(10 * time.Millisecond)

	stream.Publish(context.Background(), "multi", core.Event{ID: "1", Type: "test"})
	stream.Publish(context.Background(), "multi", core.Event{ID: "2", Type: "test"})

	if atomic.LoadInt32(&countA) != 2 {
		t.Errorf("subscriber A expected 2 events, got %d", atomic.LoadInt32(&countA))
	}
	if atomic.LoadInt32(&countB) != 2 {
		t.Errorf("subscriber B expected 2 events, got %d", atomic.LoadInt32(&countB))
	}
}

func TestStream_CloseStopsPublish(t *testing.T) {
	stream := NewStream()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var received int32
	go stream.Subscribe(ctx, "close-test", func(ctx context.Context, event core.Event) error {
		atomic.AddInt32(&received, 1)
		return nil
	})

	time.Sleep(10 * time.Millisecond)

	stream.Publish(context.Background(), "close-test", core.Event{ID: "1", Type: "test"})
	if atomic.LoadInt32(&received) != 1 {
		t.Fatalf("expected 1 event before close, got %d", atomic.LoadInt32(&received))
	}

	stream.Close()

	stream.Publish(context.Background(), "close-test", core.Event{ID: "2", Type: "test"})
	if atomic.LoadInt32(&received) != 1 {
		t.Errorf("expected no events after close, got %d total", atomic.LoadInt32(&received))
	}
}

func TestStream_SubscribeBlocksUntilCancel(t *testing.T) {
	stream := NewStream()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		err := stream.Subscribe(ctx, "blocking", func(ctx context.Context, event core.Event) error {
			return nil
		})
		done <- err
	}()

	// Give Subscribe a moment to block
	time.Sleep(50 * time.Millisecond)

	select {
	case <-done:
		t.Fatal("Subscribe should be blocking, but it returned")
	default:
		// Good — it's blocking
	}

	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Subscribe did not return after cancel")
	}
}

func TestStream_TopicIsolation(t *testing.T) {
	stream := NewStream()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	topicACalls := 0
	topicBCalls := 0

	go stream.Subscribe(ctx, "topic-a", func(ctx context.Context, event core.Event) error {
		mu.Lock()
		topicACalls++
		mu.Unlock()
		return nil
	})

	go stream.Subscribe(ctx, "topic-b", func(ctx context.Context, event core.Event) error {
		mu.Lock()
		topicBCalls++
		mu.Unlock()
		return nil
	})

	time.Sleep(10 * time.Millisecond)

	stream.Publish(context.Background(), "topic-a", core.Event{ID: "1", Type: "test"})
	stream.Publish(context.Background(), "topic-a", core.Event{ID: "2", Type: "test"})
	stream.Publish(context.Background(), "topic-b", core.Event{ID: "3", Type: "test"})

	mu.Lock()
	defer mu.Unlock()
	if topicACalls != 2 {
		t.Errorf("topic-a expected 2 events, got %d", topicACalls)
	}
	if topicBCalls != 1 {
		t.Errorf("topic-b expected 1 event, got %d", topicBCalls)
	}
}
