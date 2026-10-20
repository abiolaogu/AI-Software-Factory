package integration_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stratogroup/neuroforge-sdx/internal/adapters/storage"
	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stratogroup/neuroforge-sdx/internal/ports"
	"github.com/stratogroup/neuroforge-sdx/test/integration/harness"
	"github.com/stretchr/testify/require"
)

func TestPostgresStorePersistsTenantScopedData(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("set RUN_INTEGRATION=1 to execute the testcontainers integration harness")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	stack, err := harness.Start(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		terminateCtx, terminateCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer terminateCancel()
		require.NoError(t, stack.Terminate(terminateCtx))
	})

	var store *storage.PostgresStore
	for attempt := 0; attempt < 30; attempt++ {
		store, err = storage.NewPostgresStore(ctx, stack.YugabyteDSN)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	workflow := domain.WorkflowResponse{
		WorkflowID:   "workflow-1",
		Status:       "queued",
		CurrentStage: domain.WorkflowStageInit,
		Progress:     0.1,
		CreatedAt:    "2026-03-21T00:00:00Z",
	}
	require.NoError(t, store.SaveWorkflow(ctx, "tenant-a", workflow, map[string]domain.ArtifactDescriptor{
		"manifest": {
			URL:       "https://example.com/manifest.json",
			Type:      "application/json",
			Size:      123,
			CreatedAt: "2026-03-21T00:00:00Z",
		},
	}))

	foundWorkflow, err := store.GetWorkflow(ctx, "tenant-a", "workflow-1")
	require.NoError(t, err)
	require.Equal(t, "workflow-1", foundWorkflow.WorkflowID)

	_, err = store.GetWorkflow(ctx, "tenant-b", "workflow-1")
	require.ErrorIs(t, err, domain.ErrNotFound)

	workflows, pagination, err := store.ListWorkflows(ctx, ports.WorkflowFilter{TenantID: "tenant-a", Page: 1, Size: 20})
	require.NoError(t, err)
	require.Len(t, workflows, 1)
	require.Equal(t, 1, pagination.Total)

	prd := domain.PRDResponse{
		PRDID:     "prd-1",
		Markdown:  "# PRD",
		JSON:      map[string]any{"tenant_id": "tenant-a"},
		CreatedAt: "2026-03-21T00:00:00Z",
	}
	require.NoError(t, store.SavePRD(ctx, "tenant-a", prd))

	foundPRD, err := store.GetPRD(ctx, "tenant-a", "prd-1")
	require.NoError(t, err)
	require.Equal(t, "prd-1", foundPRD.PRDID)

	record := domain.IdempotencyRecord{
		TenantID:       "tenant-a",
		Operation:      "createWorkflow",
		Key:            "idem-1",
		StatusCode:     http.StatusCreated,
		ResponseBody:   []byte(`{"workflow_id":"workflow-1"}`),
		ResponseDigest: "digest-1",
	}
	require.NoError(t, store.SaveIdempotencyRecord(ctx, record))

	foundRecord, found, err := store.GetIdempotencyRecord(ctx, "tenant-a", "createWorkflow", "idem-1")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, http.StatusCreated, foundRecord.StatusCode)
}
