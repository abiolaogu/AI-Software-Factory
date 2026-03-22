package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/stratogroup/neuroforge-sdx/internal/adapters/config"
	"github.com/stretchr/testify/require"
)

func TestLoadAndHelpers(t *testing.T) {
	t.Setenv("NF_HTTP_ADDR", ":9000")
	t.Setenv("NF_GRPC_ADDR", ":9443")
	t.Setenv("NF_WORKER_ADDR", ":9001")
	t.Setenv("NF_MOCK_HTTP_ADDR", ":9002")
	t.Setenv("NF_SERVICE_NAME", "test-service")
	t.Setenv("NF_ENVIRONMENT", "test")
	t.Setenv("NF_LOG_LEVEL", "DEBUG")
	t.Setenv("NF_SHUTDOWN_TIMEOUT", "30s")
	t.Setenv("NF_NATS_URL", "nats://localhost:4222")
	t.Setenv("NF_NATS_SUBJECT", "test.subject")
	t.Setenv("NF_NATS_STREAM", "regional-events")
	t.Setenv("NF_OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	t.Setenv("NF_YUGABYTE_DSN", "postgres://localhost/db")
	t.Setenv("NF_DRAGONFLY_ADDR", "localhost:6379")
	t.Setenv("NF_QDRANT_URL", "http://localhost:6333")
	t.Setenv("NF_ARTIFACT_BASE_URL", "https://artifacts.example.com")
	t.Setenv("NF_ARTIFACT_SIGNING_KEY", "signing-key")
	t.Setenv("NF_ARTIFACT_URL_TTL", "45m")
	t.Setenv("NF_ALLOW_ANONYMOUS", "true")

	cfg := config.Load(config.ModeAPI)
	require.Equal(t, ":9000", cfg.HTTPAddr)
	require.Equal(t, ":9443", cfg.GRPCAddr)
	require.Equal(t, "test-service", cfg.ServiceName)
	require.Equal(t, 30*time.Second, cfg.ShutdownTimeout)
	require.Equal(t, "regional-events", cfg.NATSStream)
	require.Equal(t, "https://artifacts.example.com", cfg.ArtifactBaseURL)
	require.Equal(t, "signing-key", cfg.ArtifactSigningKey)
	require.Equal(t, 45*time.Minute, cfg.ArtifactURLTTL)
	require.True(t, cfg.AllowAnonymousAuth)
	require.Equal(t, ":9000", cfg.ListenAddress())

	cfg.Mode = config.ModeWorker
	require.Equal(t, ":9001", cfg.ListenAddress())
	cfg.Mode = config.ModeMock
	require.Equal(t, ":9002", cfg.ListenAddress())

	checks := cfg.ReadinessChecks()
	require.Equal(t, "configured", checks["database"])
	require.Equal(t, "configured", checks["redis"])
	require.Equal(t, "configured", checks["memory_fabric"])
	require.Equal(t, "configured", checks["messaging"])
	require.Equal(t, "configured", checks["artifacts"])
}

func TestLoadFallsBackOnInvalidEnvironmentValues(t *testing.T) {
	_ = os.Unsetenv("NF_HTTP_ADDR")
	t.Setenv("NF_SHUTDOWN_TIMEOUT", "invalid")
	t.Setenv("NF_ARTIFACT_URL_TTL", "invalid")
	t.Setenv("NF_ALLOW_ANONYMOUS", "invalid")

	cfg := config.Load(config.ModeAPI)
	require.Equal(t, ":8080", cfg.HTTPAddr)
	require.Equal(t, 15*time.Second, cfg.ShutdownTimeout)
	require.Equal(t, 15*time.Minute, cfg.ArtifactURLTTL)
	require.False(t, cfg.AllowAnonymousAuth)
}

func TestValidateAllowsLocalDevelopmentDefaults(t *testing.T) {
	cfg := config.Load(config.ModeAPI)
	require.NoError(t, cfg.Validate())
}

func TestValidateRejectsUnsafeProductionDefaults(t *testing.T) {
	cfg := config.Config{
		Mode:               config.ModeAPI,
		Environment:        "production",
		NATSStream:         "regional-events",
		ArtifactBaseURL:    "https://artifacts.example.com",
		ArtifactSigningKey: "neuroforge-sdx-local-signing-key",
		AllowAnonymousAuth: true,
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.ErrorContains(t, err, "NF_YUGABYTE_DSN is required outside local and test environments")
	require.ErrorContains(t, err, "NF_NATS_URL is required when NF_NATS_STREAM is set")
	require.ErrorContains(t, err, "NF_ARTIFACT_SIGNING_KEY must be overridden outside local and test environments")
	require.ErrorContains(t, err, "NF_ALLOW_ANONYMOUS must be false outside local and test environments")
}

func TestValidateAllowsMockModeAnonymousAndOptionalInfrastructure(t *testing.T) {
	cfg := config.Config{
		Mode:               config.ModeMock,
		Environment:        "production",
		ArtifactBaseURL:    "https://artifacts.example.com",
		ArtifactSigningKey: "managed-signing-key",
		AllowAnonymousAuth: true,
	}

	require.NoError(t, cfg.Validate())
}
