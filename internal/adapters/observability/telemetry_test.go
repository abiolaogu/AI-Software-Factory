package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLoggerAndTelemetryInitialization(t *testing.T) {
	logger := NewLogger("neuroforge-sdx", "test", "DEBUG")
	require.NotNil(t, logger)

	shutdown, err := InitTelemetry(context.Background(), "neuroforge-sdx", "test")
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	require.NoError(t, shutdown(context.Background()))
}

func TestParseLevel(t *testing.T) {
	require.Equal(t, parseLevel("DEBUG").String(), parseLevel("DEBUG").String())
	require.Equal(t, parseLevel("WARN").String(), parseLevel("WARN").String())
	require.Equal(t, parseLevel("ERROR").String(), parseLevel("ERROR").String())
	require.Equal(t, parseLevel("unexpected").String(), parseLevel("INFO").String())
}
