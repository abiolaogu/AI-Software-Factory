package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stratogroup/neuroforge-sdx/test/integration/harness"
	"github.com/stretchr/testify/require"
)

func TestIntegrationHarnessBootsPlatformDependencies(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("set RUN_INTEGRATION=1 to execute the testcontainers integration harness")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	stack, err := harness.Start(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, stack.YugabyteDSN)
	require.NotEmpty(t, stack.DragonflyAddr)
	require.NotEmpty(t, stack.NATSURL)
	require.NotEmpty(t, stack.QdrantURL)

	t.Cleanup(func() {
		terminateCtx, terminateCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer terminateCancel()
		require.NoError(t, stack.Terminate(terminateCtx))
	})
}
