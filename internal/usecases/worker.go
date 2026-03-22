package usecases

import (
	"context"
	"encoding/json"
	"time"

	"github.com/stratogroup/neuroforge-sdx/internal/ports"
)

type HeartbeatWorker struct {
	publisher ports.EventPublisher
	interval  time.Duration
	now       func() time.Time
}

func NewHeartbeatWorker(publisher ports.EventPublisher, interval time.Duration) *HeartbeatWorker {
	return &HeartbeatWorker{
		publisher: publisher,
		interval:  interval,
		now:       time.Now,
	}
}

func (w *HeartbeatWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	if err := w.emit(ctx, "worker.started"); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.emit(ctx, "worker.heartbeat"); err != nil {
				return err
			}
		}
	}
}

func (w *HeartbeatWorker) emit(ctx context.Context, subject string) error {
	if w.publisher == nil {
		return nil
	}

	body, err := json.Marshal(map[string]any{
		"subject":   subject,
		"timestamp": w.now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}

	return w.publisher.Publish(ctx, subject, body)
}
