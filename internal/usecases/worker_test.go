package usecases

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type workerEvent struct {
	subject string
	body    []byte
}

type workerPublisher struct {
	mu     sync.Mutex
	events []workerEvent
	ch     chan workerEvent
}

func (p *workerPublisher) Publish(_ context.Context, subject string, body []byte) error {
	event := workerEvent{subject: subject, body: append([]byte(nil), body...)}
	p.mu.Lock()
	p.events = append(p.events, event)
	p.mu.Unlock()
	if p.ch != nil {
		p.ch <- event
	}
	return nil
}

func (p *workerPublisher) Close() error {
	return nil
}

func TestHeartbeatWorkerEmitPublishesStructuredPayload(t *testing.T) {
	publisher := &workerPublisher{}
	worker := NewHeartbeatWorker(publisher, time.Second)
	worker.now = func() time.Time { return time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC) }

	require.NoError(t, worker.emit(context.Background(), "worker.started"))
	require.Len(t, publisher.events, 1)
	require.Equal(t, "worker.started", publisher.events[0].subject)

	payload := map[string]any{}
	require.NoError(t, json.Unmarshal(publisher.events[0].body, &payload))
	require.Equal(t, "worker.started", payload["subject"])
	require.Equal(t, "2026-03-21T10:00:00Z", payload["timestamp"])
}

func TestHeartbeatWorkerRunStopsAfterCancellation(t *testing.T) {
	publisher := &workerPublisher{ch: make(chan workerEvent, 4)}
	worker := NewHeartbeatWorker(publisher, 5*time.Millisecond)
	worker.now = func() time.Time { return time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC) }

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- worker.Run(ctx)
	}()

	first := <-publisher.ch
	second := <-publisher.ch
	cancel()

	require.NoError(t, <-done)
	require.Equal(t, "worker.started", first.subject)
	require.Equal(t, "worker.heartbeat", second.subject)
}

func TestHeartbeatWorkerEmitWithoutPublisherIsNoop(t *testing.T) {
	worker := NewHeartbeatWorker(nil, time.Second)
	require.NoError(t, worker.emit(context.Background(), "worker.started"))
}
