package storage_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stratogroup/neuroforge-sdx/internal/adapters/storage"
	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stratogroup/neuroforge-sdx/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestMemoryStoreIsolatesTenantData(t *testing.T) {
	store := storage.NewMemoryStore()
	ctx := context.Background()

	workflow := domain.WorkflowResponse{
		WorkflowID:   "workflow-1",
		Status:       "queued",
		CurrentStage: domain.WorkflowStageInit,
		CreatedAt:    "2026-03-21T00:00:00Z",
	}

	err := store.SaveWorkflow(ctx, "tenant-a", workflow, map[string]domain.ArtifactDescriptor{
		"manifest": {URL: "https://example.com/manifest.json"},
	})
	require.NoError(t, err)

	listA, pagination, err := store.ListWorkflows(ctx, ports.WorkflowFilter{TenantID: "tenant-a", Page: 1, Size: 20})
	require.NoError(t, err)
	require.Len(t, listA, 1)
	require.Equal(t, 1, pagination.Total)

	listB, pagination, err := store.ListWorkflows(ctx, ports.WorkflowFilter{TenantID: "tenant-b", Page: 1, Size: 20})
	require.NoError(t, err)
	require.Empty(t, listB)
	require.Equal(t, 0, pagination.Total)

	_, err = store.GetWorkflow(ctx, "tenant-b", workflow.WorkflowID)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestMemoryStoreIdempotencyRecordsAreTenantScoped(t *testing.T) {
	store := storage.NewMemoryStore()
	ctx := context.Background()

	record := domain.IdempotencyRecord{
		TenantID:       "tenant-a",
		Operation:      "createWorkflow",
		Key:            "idem-1",
		StatusCode:     http.StatusCreated,
		ResponseBody:   []byte(`{"workflow_id":"workflow-1"}`),
		ResponseDigest: "digest-1",
	}

	require.NoError(t, store.SaveIdempotencyRecord(ctx, record))

	saved, found, err := store.GetIdempotencyRecord(ctx, "tenant-a", "createWorkflow", "idem-1")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, http.StatusCreated, saved.StatusCode)

	_, found, err = store.GetIdempotencyRecord(ctx, "tenant-b", "createWorkflow", "idem-1")
	require.NoError(t, err)
	require.False(t, found)
}

func TestMemoryStoreSupportsArtifactsCancellationAndPRDs(t *testing.T) {
	store := storage.NewMemoryStore()
	ctx := context.Background()

	workflow := domain.WorkflowResponse{
		WorkflowID:   "workflow-1",
		Status:       "queued",
		CurrentStage: domain.WorkflowStageInit,
		CreatedAt:    "2026-03-21T00:00:00Z",
	}
	artifacts := map[string]domain.ArtifactDescriptor{
		"manifest": {URL: "https://example.com/manifest.json", Type: "application/json"},
	}

	require.NoError(t, store.SaveWorkflow(ctx, "tenant-a", workflow, artifacts))

	artifactResponse, err := store.GetWorkflowArtifacts(ctx, "tenant-a", "workflow-1")
	require.NoError(t, err)
	require.Equal(t, "https://example.com/manifest.json", artifactResponse.Artifacts["manifest"].URL)

	require.NoError(t, store.CancelWorkflow(ctx, "tenant-a", "workflow-1"))

	cancelled, err := store.GetWorkflow(ctx, "tenant-a", "workflow-1")
	require.NoError(t, err)
	require.Equal(t, "failed", cancelled.Status)
	require.Contains(t, cancelled.Errors, "workflow cancelled by operator")

	require.ErrorIs(t, store.CancelWorkflow(ctx, "tenant-a", "missing"), domain.ErrNotFound)
	_, err = store.GetWorkflowArtifacts(ctx, "tenant-a", "missing")
	require.ErrorIs(t, err, domain.ErrNotFound)

	prd := domain.PRDResponse{
		PRDID:                 "prd-1",
		Markdown:              "# PRD",
		JSON:                  map[string]any{"tenant_id": "tenant-a"},
		SuccessMetrics:        []domain.SuccessMetric{{Name: "coverage", Target: ">=80%", Measurement: "line"}},
		UserStories:           []string{"As an engineer, I can create tenant-safe workflows."},
		TechnicalRequirements: []string{"Prepared statements only."},
		CreatedAt:             "2026-03-21T00:00:00Z",
	}

	require.NoError(t, store.SavePRD(ctx, "tenant-a", prd))

	loadedPRD, err := store.GetPRD(ctx, "tenant-a", "prd-1")
	require.NoError(t, err)
	require.Equal(t, prd.PRDID, loadedPRD.PRDID)
	require.Equal(t, "tenant-a", loadedPRD.JSON["tenant_id"])

	_, err = store.GetPRD(ctx, "tenant-b", "prd-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}
