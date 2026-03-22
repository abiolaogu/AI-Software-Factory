package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/stratogroup/neuroforge-sdx/internal/adapters/artifacts"
	cfgadapter "github.com/stratogroup/neuroforge-sdx/internal/adapters/config"
	grpcadapter "github.com/stratogroup/neuroforge-sdx/internal/adapters/grpc"
	httpadapter "github.com/stratogroup/neuroforge-sdx/internal/adapters/http"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/messaging"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/observability"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/spec"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/storage"
	"github.com/stratogroup/neuroforge-sdx/internal/usecases"
)

func main() {
	if err := run(cfgadapter.ModeAPI); err != nil {
		slog.Error("api-server exited with error", "error", err)
		os.Exit(1)
	}
}

func run(mode cfgadapter.Mode) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := cfgadapter.Load(mode)
	if err := cfg.Validate(); err != nil {
		return err
	}
	logger := observability.NewLogger(cfg.ServiceName, cfg.Environment, cfg.LogLevel)

	shutdownTelemetry, err := observability.InitTelemetry(ctx, cfg.ServiceName, cfg.Environment)
	if err != nil {
		return err
	}
	defer shutdownTelemetry(context.Background()) //nolint:errcheck

	if _, err := spec.Load(); err != nil {
		return err
	}

	publisher, err := messaging.New(cfg.NATSURL, cfg.NATSSubject, messaging.WithJetStreamStream(cfg.NATSStream))
	if err != nil {
		logger.Warn("falling back to noop event publisher", "error", err)
		publisher = messaging.NoopPublisher{}
	}
	defer publisher.Close() //nolint:errcheck

	store, err := storage.NewRuntimeStore(ctx, cfg.YugabyteDSN)
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
		publisher,
		artifactService,
		cfg.ServiceName,
		cfg.ReadinessChecks(),
		string(mode),
	)

	router := httpadapter.NewRouter(cfg, application, logger)
	httpServer := &http.Server{
		Addr:              cfg.ListenAddress(),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	grpcServer := grpcadapter.NewServer(cfg, application, logger)

	startHTTPServer(logger, httpServer, stop)

	grpcListener, err := startGRPCServer(logger, cfg.GRPCAddr, grpcServer, stop)
	if err != nil {
		return err
	}

	<-ctx.Done()
	logger.Info("shutting down api server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	return errors.Join(
		httpServer.Shutdown(shutdownCtx),
		shutdownGRPCServer(shutdownCtx, grpcServer, grpcListener),
	)
}

func startHTTPServer(logger *slog.Logger, server *http.Server, stop context.CancelFunc) {
	go func() {
		logger.Info("starting api server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api server listen failed", "error", err)
			stop()
		}
	}()
}

func startGRPCServer(logger *slog.Logger, addr string, server interface {
	Serve(net.Listener) error
}, stop context.CancelFunc) (net.Listener, error) {
	if strings.TrimSpace(addr) == "" {
		logger.Warn("grpc transport disabled")
		return nil, nil
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	go func() {
		logger.Info("starting grpc server", "addr", addr)
		if err := server.Serve(listener); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Error("grpc server listen failed", "error", err)
			stop()
		}
	}()

	return listener, nil
}

func shutdownGRPCServer(ctx context.Context, server interface {
	GracefulStop()
	Stop()
}, listener net.Listener) error {
	if server == nil {
		return nil
	}

	done := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		if listener != nil {
			_ = listener.Close()
		}
		return nil
	case <-ctx.Done():
		server.Stop()
	}

	if listener != nil {
		_ = listener.Close()
	}

	return nil
}
