package usecases

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stratogroup/neuroforge-sdx/internal/adapters/artifacts"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/storage"
	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stretchr/testify/require"
)

type stubIdempotencyRepository struct {
	record     domain.IdempotencyRecord
	found      bool
	getErr     error
	saveErr    error
	saved      domain.IdempotencyRecord
	saveCalled bool
}

func (r *stubIdempotencyRepository) SaveIdempotencyRecord(_ context.Context, record domain.IdempotencyRecord) error {
	r.saveCalled = true
	r.saved = record
	return r.saveErr
}

func (r *stubIdempotencyRepository) GetIdempotencyRecord(_ context.Context, _, _, _ string) (domain.IdempotencyRecord, bool, error) {
	if r.getErr != nil {
		return domain.IdempotencyRecord{}, false, r.getErr
	}
	return r.record, r.found, nil
}

func TestWithIdempotencyReturnsFreshResponseWithoutKey(t *testing.T) {
	repository := &stubIdempotencyRepository{}

	response, err := withIdempotency(context.Background(), repository, "tenant-a", "workflow.create", http.StatusCreated, func() (map[string]string, error) {
		return map[string]string{"workflow_id": "workflow-1"}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "workflow-1", response["workflow_id"])
	require.False(t, repository.saveCalled)
}

func TestWithIdempotencyReturnsCachedResponse(t *testing.T) {
	repository := &stubIdempotencyRepository{
		record: domain.IdempotencyRecord{
			ResponseBody: []byte(`{"workflow_id":"workflow-1"}`),
		},
		found: true,
	}

	ctx := domain.WithIdempotencyKey(context.Background(), "idem-1")
	response, err := withIdempotency(ctx, repository, "tenant-a", "workflow.create", http.StatusCreated, func() (map[string]string, error) {
		return nil, errors.New("should not execute")
	})
	require.NoError(t, err)
	require.Equal(t, "workflow-1", response["workflow_id"])
	require.False(t, repository.saveCalled)
}

func TestWithIdempotencyWrapsRepositoryAndPayloadErrors(t *testing.T) {
	ctx := domain.WithIdempotencyKey(context.Background(), "idem-1")

	_, err := withIdempotency(ctx, &stubIdempotencyRepository{getErr: errors.New("db down")}, "tenant-a", "workflow.create", http.StatusCreated, func() (map[string]string, error) {
		return map[string]string{"workflow_id": "workflow-1"}, nil
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "loading idempotency record failed")

	_, err = withIdempotency(ctx, &stubIdempotencyRepository{
		record: domain.IdempotencyRecord{ResponseBody: []byte(`{`)},
		found:  true,
	}, "tenant-a", "workflow.create", http.StatusCreated, func() (map[string]string, error) {
		return map[string]string{"workflow_id": "workflow-1"}, nil
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "decoding idempotency payload failed")
}

func TestWithIdempotencyWrapsSaveErrors(t *testing.T) {
	repository := &stubIdempotencyRepository{saveErr: errors.New("write failed")}
	ctx := domain.WithIdempotencyKey(context.Background(), "idem-1")

	_, err := withIdempotency(ctx, repository, "tenant-a", "workflow.create", http.StatusCreated, func() (map[string]string, error) {
		return map[string]string{"workflow_id": "workflow-1"}, nil
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "saving idempotency record failed")
	require.True(t, repository.saveCalled)
	require.Equal(t, "tenant-a", repository.saved.TenantID)
}

func TestNormalizeRepositoryErrorPreservesKnownErrors(t *testing.T) {
	require.NoError(t, normalizeRepositoryError("workflow.get", nil))
	require.ErrorIs(t, normalizeRepositoryError("workflow.get", domain.ErrNotFound), domain.ErrNotFound)
	require.ErrorIs(t, normalizeRepositoryError("workflow.get", domain.ErrConflict), domain.ErrConflict)

	err := normalizeRepositoryError("workflow.get", errors.New("unexpected"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "repository operation failed")
}

func TestGenerateCodeBuildsSpecAwareGoScaffold(t *testing.T) {
	store := storage.NewMemoryStore()
	artifactService, err := artifacts.NewSignedArtifactService("neuroforge-sdx", "https://artifacts.example.com", "signing-key", 15*time.Minute)
	require.NoError(t, err)

	service := NewService(
		store,
		store,
		store,
		nil,
		artifactService,
		"neuroforge-sdx",
		map[string]string{"database": "configured"},
		"test",
	)
	service.newID = func() string { return "code-123" }

	ctx := domain.WithTenantID(context.Background(), "tenant-a")
	response, err := service.GenerateCode(ctx, domain.CodeRequest{
		Specification: domain.CodeSpecification{
			Description:  "Generate workflow status handler scaffold",
			Interface:    "REST",
			Requirements: []string{"otel", "tenant-safety"},
		},
		Language:  "go",
		Framework: "gin",
	})
	require.NoError(t, err)
	require.Equal(t, "code-123", response.CodeID)
	require.Equal(t, "internal/generated/go/workflow_status_handler.go", response.FilePath)
	require.NotContains(t, response.Code, "TODO")
	require.Contains(t, response.Code, "func RegisterWorkflowStatusRoutes")
	require.Contains(t, response.Code, "tenant-safety")
	require.Contains(t, response.Tests, "func TestHandleWorkflowStatus")

	_, err = parser.ParseFile(token.NewFileSet(), response.FilePath, response.Code, parser.AllErrors)
	require.NoError(t, err)
	_, err = parser.ParseFile(token.NewFileSet(), "workflow_status_handler_test.go", response.Tests, parser.AllErrors)
	require.NoError(t, err)
}

func TestCreateWorkflowAndSecurityEvidenceUseSignedArtifacts(t *testing.T) {
	store := storage.NewMemoryStore()
	artifactService, err := artifacts.NewSignedArtifactService("neuroforge-sdx", "https://artifacts.example.com/platform", "signing-key", 20*time.Minute)
	require.NoError(t, err)

	service := NewService(
		store,
		store,
		store,
		nil,
		artifactService,
		"neuroforge-sdx",
		map[string]string{"database": "configured"},
		"test",
	)
	service.newID = func() string { return "workflow-123" }
	service.now = func() time.Time {
		return time.Date(2026, 3, 22, 14, 0, 0, 0, time.UTC)
	}

	ctx := domain.WithTenantID(context.Background(), "tenant-a")

	workflow, err := service.CreateWorkflow(ctx, domain.WorkflowRequest{
		Prompt: "Build a production-grade multi-tenant workflow orchestration service with signed artifacts and compliance evidence.",
	})
	require.NoError(t, err)

	manifestURL, err := url.Parse(workflow.Artifacts["manifest"])
	require.NoError(t, err)
	require.Equal(t, "/platform/workflows/workflow-123/manifest.json", manifestURL.Path)
	require.Equal(t, "tenant-a", manifestURL.Query().Get("tenant"))
	require.NotEmpty(t, manifestURL.Query().Get("signature"))

	artifactBundle, err := service.GetWorkflowArtifacts(ctx, workflow.WorkflowID)
	require.NoError(t, err)
	require.Equal(t, workflow.Artifacts["manifest"], artifactBundle.Artifacts["manifest"].URL)

	findings, err := service.GetSecurityFindings(ctx, "scan-123")
	require.NoError(t, err)
	sbomURL, err := url.Parse(findings.SBOM)
	require.NoError(t, err)
	require.Equal(t, "/platform/compliance/scan-123/sbom.spdx.json", sbomURL.Path)
	require.Equal(t, "tenant-a", sbomURL.Query().Get("tenant"))
	require.True(t, strings.HasPrefix(findings.Signature, "cosign://rekor/neuroforge-sdx/scan-123"))
}

func TestGenerateCodeSupportsAdditionalLanguages(t *testing.T) {
	store := storage.NewMemoryStore()
	artifactService, err := artifacts.NewSignedArtifactService("neuroforge-sdx", "https://artifacts.example.com", "signing-key", 15*time.Minute)
	require.NoError(t, err)

	service := NewService(
		store,
		store,
		store,
		nil,
		artifactService,
		"neuroforge-sdx",
		map[string]string{"database": "configured"},
		"test",
	)
	service.newID = func() string { return "code-456" }

	ctx := domain.WithTenantID(context.Background(), "tenant-a")

	typescriptResponse, err := service.GenerateCode(ctx, domain.CodeRequest{
		Specification: domain.CodeSpecification{
			Description:  "Generate tenant analytics controller",
			Interface:    "REST",
			Requirements: []string{"audit", "rbac"},
		},
		Language:  "typescript",
		Framework: "express",
	})
	require.NoError(t, err)
	require.Contains(t, typescriptResponse.Code, "registerTenantAnalyticsRoutes")
	require.Contains(t, typescriptResponse.Code, "audit")
	require.Contains(t, typescriptResponse.Tests, "describe(")

	pythonResponse, err := service.GenerateCode(ctx, domain.CodeRequest{
		Specification: domain.CodeSpecification{
			Description:  "Generate release compliance service",
			Interface:    "REST",
			Requirements: []string{"evidence", "signing"},
		},
		Language: "python",
	})
	require.NoError(t, err)
	require.Contains(t, pythonResponse.Code, "handle_release_compliance")
	require.Contains(t, pythonResponse.Code, "evidence")
	require.Contains(t, pythonResponse.Tests, "def test_")
}

func TestGenerateTestsProducesLanguageSpecificSuites(t *testing.T) {
	store := storage.NewMemoryStore()
	artifactService, err := artifacts.NewSignedArtifactService("neuroforge-sdx", "https://artifacts.example.com", "signing-key", 15*time.Minute)
	require.NoError(t, err)

	service := NewService(
		store,
		store,
		store,
		nil,
		artifactService,
		"neuroforge-sdx",
		map[string]string{"database": "configured"},
		"test",
	)

	ctx := domain.WithTenantID(context.Background(), "tenant-a")

	goTests, err := service.GenerateTests(ctx, domain.GenerateTestsRequest{Code: "package generated\n\nfunc HandleWorkflowStatus() {}", Language: "go"})
	require.NoError(t, err)
	require.Contains(t, goTests.Tests, "TestHandleWorkflowStatus")

	tsTests, err := service.GenerateTests(ctx, domain.GenerateTestsRequest{Code: "export const handler = () => true;", Language: "typescript"})
	require.NoError(t, err)
	require.Contains(t, tsTests.Tests, "describe(")

	pyTests, err := service.GenerateTests(ctx, domain.GenerateTestsRequest{Code: "def handler():\n    return True\n", Language: "python"})
	require.NoError(t, err)
	require.Contains(t, pyTests.Tests, "def test_generated_suite")
}

func TestArtifactHelpersReturnContextRichErrorsWhenUnconfigured(t *testing.T) {
	service := NewService(
		storage.NewMemoryStore(),
		storage.NewMemoryStore(),
		storage.NewMemoryStore(),
		nil,
		nil,
		"neuroforge-sdx",
		map[string]string{"database": "configured"},
		"test",
	)

	_, err := service.workflowArtifacts(context.Background(), "tenant-a", "workflow-1", time.Now().UTC())
	require.Error(t, err)
	require.Contains(t, err.Error(), "generating workflow artifacts failed")

	_, err = service.designArtifacts(context.Background(), "tenant-a", "design-1", time.Now().UTC())
	require.Error(t, err)
	require.Contains(t, err.Error(), "generating design artifacts failed")

	_, err = service.securityEvidence(context.Background(), "tenant-a", "scan-1", time.Now().UTC())
	require.Error(t, err)
	require.Contains(t, err.Error(), "generating security evidence failed")
}
