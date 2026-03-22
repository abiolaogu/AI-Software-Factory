package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stratogroup/neuroforge-sdx/internal/adapters/artifacts"
	cfgadapter "github.com/stratogroup/neuroforge-sdx/internal/adapters/config"
	httpadapter "github.com/stratogroup/neuroforge-sdx/internal/adapters/http"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/messaging"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/observability"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/spec"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/storage"
	"github.com/stratogroup/neuroforge-sdx/internal/usecases"
)

func main() {
	if err := run(cfgadapter.ModeMock); err != nil {
		slog.Error("mock-api exited with error", "error", err)
		os.Exit(1)
	}
}

func run(mode cfgadapter.Mode) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := cfgadapter.Load(mode)
	cfg.AllowAnonymousAuth = true
	if err := cfg.Validate(); err != nil {
		return err
	}

	logger := observability.NewLogger(cfg.ServiceName+"-mock", cfg.Environment, cfg.LogLevel)

	shutdownTelemetry, err := observability.InitTelemetry(ctx, cfg.ServiceName+"-mock", cfg.Environment)
	if err != nil {
		return err
	}
	defer shutdownTelemetry(context.Background()) //nolint:errcheck

	if _, err := spec.Load(); err != nil {
		return err
	}

	store, err := storage.NewRuntimeStore(ctx, "")
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck

	artifactService, err := artifacts.NewSignedArtifactService(cfg.ServiceName, cfg.ArtifactBaseURL, cfg.ArtifactSigningKey, cfg.ArtifactURLTTL)
	if err != nil {
		return err
	}

	application := usecases.NewService(
		store.WorkflowRepository,
		store.PRDRepository,
		store.IdempotencyRepository,
		messaging.NoopPublisher{},
		artifactService,
		cfg.ServiceName+"-mock",
		cfg.ReadinessChecks(),
		string(mode),
	)

	router := httpadapter.NewRouter(cfg, application, logger)
	server := &http.Server{
		Addr:              cfg.ListenAddress(),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("starting mock api", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("mock api listen failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down mock api")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
