package usecases_test

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/stratogroup/neuroforge-sdx/internal/adapters/artifacts"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/storage"
	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stratogroup/neuroforge-sdx/internal/ports"
	"github.com/stratogroup/neuroforge-sdx/internal/usecases"
	"github.com/stretchr/testify/require"
)

type capturingPublisher struct {
	subject string
	body    []byte
}

func (c *capturingPublisher) Publish(_ context.Context, subject string, body []byte) error {
	c.subject = subject
	c.body = body
	return nil
}

func (c *capturingPublisher) Close() error {
	return nil
}

func TestCreateWorkflowRequiresTenantScope(t *testing.T) {
	service := usecases.NewService(
		storage.NewMemoryStore(),
		storage.NewMemoryStore(),
		storage.NewMemoryStore(),
		nil,
		newTestArtifactService(t),
		"neuroforge-sdx",
		map[string]string{"database": "configured"},
		"test",
	)

	_, err := service.CreateWorkflow(context.Background(), domain.WorkflowRequest{
		Prompt: "Build a production-grade multi-tenant workflow orchestration service with contract-first delivery.",
	})
	require.Error(t, err)

	appErr := new(domain.AppError)
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, domain.ErrorCodeTenantRequired, appErr.Code)
}

func TestWorkflowAccessIsTenantScoped(t *testing.T) {
	store := storage.NewMemoryStore()
	service := usecases.NewService(
		store,
		store,
		store,
		nil,
		newTestArtifactService(t),
		"neuroforge-sdx",
		map[string]string{"database": "configured"},
		"test",
	)

	ctxA := domain.WithTenantID(context.Background(), "tenant-a")
	ctxB := domain.WithTenantID(context.Background(), "tenant-b")

	workflow, err := service.CreateWorkflow(ctxA, domain.WorkflowRequest{
		Prompt: "Build a production-grade multi-tenant workflow orchestration service with contract-first delivery.",
	})
	require.NoError(t, err)

	listA, err := service.ListWorkflows(ctxA, ports.WorkflowFilter{Page: 1, Size: 20})
	require.NoError(t, err)
	require.Len(t, listA.Workflows, 1)

	listB, err := service.ListWorkflows(ctxB, ports.WorkflowFilter{Page: 1, Size: 20})
	require.NoError(t, err)
	require.Empty(t, listB.Workflows)

	_, err = service.GetWorkflow(ctxB, workflow.WorkflowID)
	require.Error(t, err)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCreateWorkflowPublishesTenantAwareEvent(t *testing.T) {
	publisher := &capturingPublisher{}
	store := storage.NewMemoryStore()
	service := usecases.NewService(
		store,
		store,
		store,
		publisher,
		newTestArtifactService(t),
		"neuroforge-sdx",
		map[string]string{"database": "configured"},
		"test",
	)

	_, err := service.CreateWorkflow(domain.WithTenantID(context.Background(), "tenant-a"), domain.WorkflowRequest{
		Prompt: "Build a production-grade multi-tenant workflow orchestration service with contract-first delivery.",
	})
	require.NoError(t, err)
	require.Equal(t, "workflow.created", publisher.subject)

	var event map[string]any
	require.NoError(t, json.Unmarshal(publisher.body, &event))
	require.Equal(t, "tenant-a", event["tenant_id"])
}

func TestServiceHappyPathOperations(t *testing.T) {
	store := storage.NewMemoryStore()
	service := usecases.NewService(
		store,
		store,
		store,
		nil,
		newTestArtifactService(t),
		"neuroforge-sdx",
		map[string]string{"database": "configured", "redis": "configured"},
		"test",
	)

	ctx := domain.WithTenantID(context.Background(), "tenant-a")
	ctx = domain.WithIdempotencyKey(ctx, "idem-1")

	login, err := service.Login(context.Background(), domain.LoginRequest{Email: "architect@neuroforge.io", Password: "secret"})
	require.NoError(t, err)
	require.NotEmpty(t, login.AccessToken)

	refresh, err := service.RefreshToken(context.Background(), domain.RefreshTokenRequest{RefreshToken: "token"})
	require.NoError(t, err)
	require.NotEmpty(t, refresh.AccessToken)

	workflow, err := service.CreateWorkflow(ctx, domain.WorkflowRequest{
		Prompt: "Build a production-grade multi-tenant workflow orchestration service with contract-first delivery.",
	})
	require.NoError(t, err)

	artifacts, err := service.GetWorkflowArtifacts(ctx, workflow.WorkflowID)
	require.NoError(t, err)
	require.NotEmpty(t, artifacts.Artifacts)

	require.NoError(t, service.CancelWorkflow(ctx, workflow.WorkflowID))

	prd, err := service.GeneratePRD(ctx, domain.PRDRequest{
		Prompt: "Generate a PRD for a secure AI-native software delivery platform with tenant-aware workflow orchestration.",
		Domain: "saas",
	})
	require.NoError(t, err)

	loadedPRD, err := service.GetPRD(ctx, prd.PRDID)
	require.NoError(t, err)
	require.Equal(t, prd.PRDID, loadedPRD.PRDID)

	design, err := service.GenerateWireframes(ctx, domain.DesignRequest{PRDID: prd.PRDID, Style: "professional"})
	require.NoError(t, err)
	require.NotEmpty(t, design.DesignID)

	tokens, err := service.ExtractTokens(ctx, domain.ExtractTokensRequest{DesignID: design.DesignID})
	require.NoError(t, err)
	require.NotEmpty(t, tokens.Tokens)

	code, err := service.GenerateCode(ctx, domain.CodeRequest{Language: "go"})
	require.NoError(t, err)
	require.Equal(t, "go", code.Language)

	tests, err := service.GenerateTests(ctx, domain.GenerateTestsRequest{Code: "package main", Language: "go"})
	require.NoError(t, err)
	require.Greater(t, tests.Coverage, 0.8)

	scan, err := service.SecurityScan(ctx, domain.SecurityScanRequest{Target: "github.com/stratogroup/neuroforge-sdx"})
	require.NoError(t, err)
	findings, err := service.GetSecurityFindings(ctx, scan.ScanID)
	require.NoError(t, err)
	require.Equal(t, scan.ScanID, findings.ScanID)
	_, err = url.Parse(findings.SBOM)
	require.NoError(t, err)
	require.NotEmpty(t, findings.Signature)

	perf, err := service.PerformanceTest(ctx, domain.PerformanceTestRequest{Endpoint: "https://api.neuroforge.local/v1/workflows"})
	require.NoError(t, err)
	results, err := service.GetPerformanceResults(ctx, perf.TestID)
	require.NoError(t, err)
	require.True(t, results.Passed)

	live, err := service.HealthLive(context.Background())
	require.NoError(t, err)
	require.Equal(t, "ok", live.Status)

	ready, err := service.HealthReady(context.Background())
	require.NoError(t, err)
	require.Equal(t, "ready", ready.Status)
}

func TestServiceValidationAndLookupErrors(t *testing.T) {
	store := storage.NewMemoryStore()
	service := usecases.NewService(
		store,
		store,
		store,
		nil,
		newTestArtifactService(t),
		"neuroforge-sdx",
		map[string]string{"database": "configured"},
		"test",
	)

	ctx := domain.WithTenantID(context.Background(), "tenant-a")

	_, err := service.GeneratePRD(ctx, domain.PRDRequest{Prompt: "too short"})
	require.ErrorIs(t, err, domain.ErrValidation)

	_, err = service.GetPRD(ctx, "missing")
	require.ErrorIs(t, err, domain.ErrNotFound)

	_, err = service.GetWorkflow(ctx, "missing")
	require.ErrorIs(t, err, domain.ErrNotFound)

	_, err = service.GenerateWireframes(ctx, domain.DesignRequest{})
	require.ErrorIs(t, err, domain.ErrValidation)

	_, err = service.ExtractTokens(ctx, domain.ExtractTokensRequest{})
	require.ErrorIs(t, err, domain.ErrValidation)

	_, err = service.GenerateCode(ctx, domain.CodeRequest{})
	require.ErrorIs(t, err, domain.ErrValidation)

	_, err = service.GenerateTests(ctx, domain.GenerateTestsRequest{})
	require.ErrorIs(t, err, domain.ErrValidation)

	_, err = service.SecurityScan(ctx, domain.SecurityScanRequest{})
	require.ErrorIs(t, err, domain.ErrValidation)

	_, err = service.GetSecurityFindings(ctx, "")
	require.ErrorIs(t, err, domain.ErrValidation)

	_, err = service.PerformanceTest(ctx, domain.PerformanceTestRequest{})
	require.ErrorIs(t, err, domain.ErrValidation)

	_, err = service.GetPerformanceResults(ctx, "")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func newTestArtifactService(t *testing.T) ports.ArtifactService {
	t.Helper()

	service, err := artifacts.NewSignedArtifactService("neuroforge-sdx", "https://artifacts.example.com", "signing-key", 15*time.Minute)
	require.NoError(t, err)
	return service
}
