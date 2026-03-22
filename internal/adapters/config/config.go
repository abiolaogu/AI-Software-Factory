package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Mode string

const (
	ModeAPI    Mode = "api"
	ModeWorker Mode = "worker"
	ModeMock   Mode = "mock"
)

type Config struct {
	Mode               Mode
	HTTPAddr           string
	GRPCAddr           string
	WorkerAddr         string
	MockHTTPAddr       string
	ServiceName        string
	Environment        string
	LogLevel           string
	ShutdownTimeout    time.Duration
	NATSURL            string
	NATSSubject        string
	NATSStream         string
	OTLPEndpoint       string
	YugabyteDSN        string
	DragonflyAddr      string
	QdrantURL          string
	ArtifactBaseURL    string
	ArtifactSigningKey string
	ArtifactURLTTL     time.Duration
	AllowAnonymousAuth bool
}

func Load(mode Mode) Config {
	return Config{
		Mode:               mode,
		HTTPAddr:           getEnv("NF_HTTP_ADDR", ":8080"),
		GRPCAddr:           getEnv("NF_GRPC_ADDR", ":9090"),
		WorkerAddr:         getEnv("NF_WORKER_ADDR", ":8081"),
		MockHTTPAddr:       getEnv("NF_MOCK_HTTP_ADDR", ":8090"),
		ServiceName:        getEnv("NF_SERVICE_NAME", "neuroforge-sdx"),
		Environment:        getEnv("NF_ENVIRONMENT", "local"),
		LogLevel:           getEnv("NF_LOG_LEVEL", "INFO"),
		ShutdownTimeout:    getDurationEnv("NF_SHUTDOWN_TIMEOUT", 15*time.Second),
		NATSURL:            getEnv("NF_NATS_URL", ""),
		NATSSubject:        getEnv("NF_NATS_SUBJECT", "neuroforge.sdx.events"),
		NATSStream:         getEnv("NF_NATS_STREAM", ""),
		OTLPEndpoint:       getEnv("NF_OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		YugabyteDSN:        getEnv("NF_YUGABYTE_DSN", ""),
		DragonflyAddr:      getEnv("NF_DRAGONFLY_ADDR", ""),
		QdrantURL:          getEnv("NF_QDRANT_URL", ""),
		ArtifactBaseURL:    getEnv("NF_ARTIFACT_BASE_URL", "https://artifacts.neuroforge.local"),
		ArtifactSigningKey: getEnv("NF_ARTIFACT_SIGNING_KEY", "neuroforge-sdx-local-signing-key"),
		ArtifactURLTTL:     getDurationEnv("NF_ARTIFACT_URL_TTL", 15*time.Minute),
		AllowAnonymousAuth: getBoolEnv("NF_ALLOW_ANONYMOUS", false),
	}
}

func (c Config) ListenAddress() string {
	switch c.Mode {
	case ModeWorker:
		return c.WorkerAddr
	case ModeMock:
		return c.MockHTTPAddr
	default:
		return c.HTTPAddr
	}
}

func (c Config) ReadinessChecks() map[string]string {
	return map[string]string{
		"database":      configured(c.YugabyteDSN),
		"messaging":     configured(c.NATSURL),
		"redis":         configured(c.DragonflyAddr),
		"memory_fabric": configured(c.QdrantURL),
		"artifacts":     configured(c.ArtifactBaseURL),
	}
}

func (c Config) Validate() error {
	validationErrors := append([]error{}, c.validateRequiredSettings()...)
	validationErrors = append(validationErrors, c.validateMessagingSettings()...)
	validationErrors = append(validationErrors, c.validateManagedInfrastructureSettings()...)

	if len(validationErrors) == 0 {
		return nil
	}

	return fmt.Errorf("configuration validation failed: %w", errors.Join(validationErrors...))
}

func (c Config) validateRequiredSettings() []error {
	var validationErrors []error

	if strings.TrimSpace(c.ArtifactBaseURL) == "" {
		validationErrors = append(validationErrors, errors.New("NF_ARTIFACT_BASE_URL is required"))
	}
	if strings.TrimSpace(c.ArtifactSigningKey) == "" {
		validationErrors = append(validationErrors, errors.New("NF_ARTIFACT_SIGNING_KEY is required"))
	}

	return validationErrors
}

func (c Config) validateMessagingSettings() []error {
	var validationErrors []error

	if strings.TrimSpace(c.NATSStream) != "" && strings.TrimSpace(c.NATSURL) == "" {
		validationErrors = append(validationErrors, errors.New("NF_NATS_URL is required when NF_NATS_STREAM is set"))
	}
	if strings.TrimSpace(c.NATSURL) != "" && strings.TrimSpace(c.NATSSubject) == "" {
		validationErrors = append(validationErrors, errors.New("NF_NATS_SUBJECT is required when NF_NATS_URL is set"))
	}

	return validationErrors
}

func (c Config) validateManagedInfrastructureSettings() []error {
	if !c.requiresManagedInfrastructure() {
		return nil
	}

	var validationErrors []error
	if strings.TrimSpace(c.YugabyteDSN) == "" {
		validationErrors = append(validationErrors, errors.New("NF_YUGABYTE_DSN is required outside local and test environments"))
	}
	if strings.TrimSpace(c.ArtifactSigningKey) == "neuroforge-sdx-local-signing-key" {
		validationErrors = append(validationErrors, errors.New("NF_ARTIFACT_SIGNING_KEY must be overridden outside local and test environments"))
	}
	if c.AllowAnonymousAuth {
		validationErrors = append(validationErrors, errors.New("NF_ALLOW_ANONYMOUS must be false outside local and test environments"))
	}

	return validationErrors
}

func (c Config) requiresManagedInfrastructure() bool {
	if c.Mode == ModeMock {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(c.Environment)) {
	case "", "local", "test":
		return false
	default:
		return true
	}
}

func configured(value string) string {
	if strings.TrimSpace(value) == "" {
		return "not_configured"
	}

	return "configured"
}

func getEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return duration
}

func getBoolEnv(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}
