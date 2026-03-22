package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	cfgadapter "github.com/stratogroup/neuroforge-sdx/internal/adapters/config"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/messaging"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/observability"
	"github.com/stratogroup/neuroforge-sdx/internal/usecases"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := cfgadapter.Load(cfgadapter.ModeWorker)
	if err := cfg.Validate(); err != nil {
		slog.Error("configuration validation failed", "error", err)
		os.Exit(1)
	}
	logger := observability.NewLogger(cfg.ServiceName+"-worker", cfg.Environment, cfg.LogLevel)

	shutdownTelemetry, err := observability.InitTelemetry(ctx, cfg.ServiceName+"-worker", cfg.Environment)
	if err != nil {
		slog.Error("telemetry initialization failed", "error", err)
		os.Exit(1)
	}
	defer shutdownTelemetry(context.Background()) //nolint:errcheck

	publisher, err := messaging.New(cfg.NATSURL, cfg.NATSSubject, messaging.WithJetStreamStream(cfg.NATSStream))
	if err != nil {
		logger.Warn("falling back to noop event publisher", "error", err)
		publisher = messaging.NoopPublisher{}
	}
	defer publisher.Close() //nolint:errcheck

	worker := usecases.NewHeartbeatWorker(publisher, 30*time.Second)

	logger.Info("starting worker", "addr", cfg.ListenAddress())
	if err := worker.Run(ctx); err != nil {
		slog.Error("worker stopped with error", "error", err)
		os.Exit(1)
	}

	logger.Info("worker stopped")
}
