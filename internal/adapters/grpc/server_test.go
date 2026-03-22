package grpc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	sdxv1 "github.com/stratogroup/neuroforge-sdx/api/proto/neuroforge/sdx/v1"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/artifacts"
	cfgadapter "github.com/stratogroup/neuroforge-sdx/internal/adapters/config"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/messaging"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/observability"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/storage"
	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stratogroup/neuroforge-sdx/internal/usecases"
	"github.com/stretchr/testify/require"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestServerHealthAndWorkflowIdempotency(t *testing.T) {
	client, cleanup := newTestClient(t, false)
	defer cleanup()

	ready, err := client.HealthReady(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, "ready", ready.Status)
	require.Contains(t, ready.Checks, "database")

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"authorization", "Bearer grpc-test-token",
		strings.ToLower(domain.HeaderTenantID), "tenant-a",
		strings.ToLower(domain.HeaderIdempotencyKey), "idem-1",
	))

	first, err := client.CreateWorkflow(ctx, &sdxv1.WorkflowRequest{
		Prompt: strings.Repeat("Build a production-grade multi-tenant workflow orchestration service. ", 2),
		Domain: "api_service",
	})
	require.NoError(t, err)
	require.NotEmpty(t, first.WorkflowId)

	second, err := client.CreateWorkflow(ctx, &sdxv1.WorkflowRequest{
		Prompt: strings.Repeat("Build a production-grade multi-tenant workflow orchestration service. ", 2),
		Domain: "api_service",
	})
	require.NoError(t, err)
	require.Equal(t, first.WorkflowId, second.WorkflowId)
}

func TestServerProtectedRPCsRequireAuthAndTenant(t *testing.T) {
	client, cleanup := newTestClient(t, false)
	defer cleanup()

	_, err := client.CreateWorkflow(context.Background(), &sdxv1.WorkflowRequest{
		Prompt: strings.Repeat("Build a production-grade multi-tenant workflow orchestration service. ", 2),
	})
	require.Error(t, err)
	require.Equal(t, codes.Unauthenticated, status.Code(err))

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"authorization", "Bearer grpc-test-token",
	))
	_, err = client.CreateWorkflow(ctx, &sdxv1.WorkflowRequest{
		Prompt: strings.Repeat("Build a production-grade multi-tenant workflow orchestration service. ", 2),
	})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServerRejectsTenantClaimDrift(t *testing.T) {
	client, cleanup := newTestClient(t, false)
	defer cleanup()

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"authorization", "Bearer "+tenantToken("tenant-b"),
		strings.ToLower(domain.HeaderTenantID), "tenant-a",
	))

	_, err := client.CreateWorkflow(ctx, &sdxv1.WorkflowRequest{
		Prompt: strings.Repeat("Build a production-grade multi-tenant workflow orchestration service. ", 2),
	})
	require.Error(t, err)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestServerExposesProtoSurface(t *testing.T) {
	client, cleanup := newTestClient(t, false)
	defer cleanup()

	login, err := client.Login(context.Background(), &sdxv1.LoginRequest{
		Email:    "platform@example.com",
		Password: "secret",
	})
	require.NoError(t, err)
	require.NotEmpty(t, login.AccessToken)

	refresh, err := client.RefreshToken(context.Background(), &sdxv1.RefreshTokenRequest{
		RefreshToken: login.RefreshToken,
	})
	require.NoError(t, err)
	require.NotEmpty(t, refresh.AccessToken)

	live, err := client.HealthLive(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, "ok", live.Status)

	ctx := outgoingContext("tenant-a", "idem-surface")

	workflowMetadata, err := structpb.NewStruct(map[string]any{
		"priority": "high",
		"module":   "grpc",
	})
	require.NoError(t, err)

	workflow, err := client.CreateWorkflow(ctx, &sdxv1.WorkflowRequest{
		Prompt:          strings.Repeat("Build a production-grade multi-tenant workflow orchestration service. ", 2),
		Domain:          "api_service",
		GitStrategy:     "trunk-based",
		TargetLanguages: []string{"go"},
		Regions:         []string{"eu-west-1"},
		Metadata:        workflowMetadata,
	})
	require.NoError(t, err)
	require.NotEmpty(t, workflow.WorkflowId)
	require.Equal(t, "queued", workflow.Status)

	list, err := client.ListWorkflows(ctx, &sdxv1.ListWorkflowsRequest{Page: 1, Size: 10})
	require.NoError(t, err)
	require.NotEmpty(t, list.Workflows)
	require.NotNil(t, list.Pagination)

	current, err := client.GetWorkflow(ctx, &sdxv1.GetWorkflowRequest{WorkflowId: workflow.WorkflowId})
	require.NoError(t, err)
	require.Equal(t, workflow.WorkflowId, current.WorkflowId)

	artifacts, err := client.GetWorkflowArtifacts(ctx, &sdxv1.GetWorkflowRequest{WorkflowId: workflow.WorkflowId})
	require.NoError(t, err)
	require.NotEmpty(t, artifacts.Artifacts)

	prd, err := client.GeneratePRD(ctx, &sdxv1.PRDRequest{
		Prompt: strings.Repeat("Generate an auditable PRD for the gRPC delivery path. ", 2),
		Domain: "platform",
	})
	require.NoError(t, err)
	require.NotEmpty(t, prd.PrdId)
	require.NotNil(t, prd.Json)

	prdLookup, err := client.GetPRD(ctx, &sdxv1.GetPRDRequest{PrdId: prd.PrdId})
	require.NoError(t, err)
	require.Equal(t, prd.PrdId, prdLookup.PrdId)

	design, err := client.GenerateWireframes(ctx, &sdxv1.DesignRequest{
		PrdId:       prd.PrdId,
		Style:       "editorial",
		ColorScheme: "graphite",
	})
	require.NoError(t, err)
	require.NotEmpty(t, design.DesignId)
	require.NotNil(t, design.Tokens)

	tokens, err := client.ExtractTokens(ctx, &sdxv1.ExtractTokensRequest{DesignId: design.DesignId})
	require.NoError(t, err)
	require.NotNil(t, tokens.Tokens)

	code, err := client.GenerateCode(ctx, &sdxv1.CodeRequest{
		Specification: &sdxv1.CodeSpecification{
			Description:  "Build a gRPC service implementation",
			Interface:    "type Service interface { Run(context.Context) error }",
			Requirements: []string{"multi-tenant", "idempotent"},
		},
		Language:  "go",
		Framework: "grpc",
	})
	require.NoError(t, err)
	require.NotEmpty(t, code.CodeId)
	require.NotNil(t, code.Validation)

	tests, err := client.GenerateTests(ctx, &sdxv1.GenerateTestsRequest{
		Code:     code.Code,
		Language: "go",
	})
	require.NoError(t, err)
	require.NotEmpty(t, tests.Tests)

	scan, err := client.SecurityScan(ctx, &sdxv1.SecurityScanRequest{
		Target:    "github.com/stratogroup/neuroforge-sdx",
		ScanTypes: []string{"sast"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, scan.ScanId)

	findings, err := client.GetSecurityFindings(ctx, &sdxv1.GetSecurityFindingsRequest{ScanId: scan.ScanId})
	require.NoError(t, err)
	require.Equal(t, scan.ScanId, findings.ScanId)
	require.NotNil(t, findings.Summary)

	performance, err := client.PerformanceTest(ctx, &sdxv1.PerformanceTestRequest{
		Endpoint:     "/v1/workflows",
		Duration:     "1m",
		VirtualUsers: 25,
		Regions:      []string{"eu-west-1"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, performance.TestId)

	results, err := client.GetPerformanceResults(ctx, &sdxv1.GetPerformanceResultsRequest{TestId: performance.TestId})
	require.NoError(t, err)
	require.Equal(t, performance.TestId, results.TestId)
	require.NotNil(t, results.Metrics)

	_, err = client.CancelWorkflow(ctx, &sdxv1.GetWorkflowRequest{WorkflowId: workflow.WorkflowId})
	require.NoError(t, err)
}

func TestServerMapsDomainErrorsToGRPCStatus(t *testing.T) {
	client, cleanup := newTestClient(t, false)
	defer cleanup()

	_, err := client.Login(context.Background(), &sdxv1.LoginRequest{})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	ctx := outgoingContext("tenant-a", "")

	_, err = client.GetWorkflow(ctx, &sdxv1.GetWorkflowRequest{WorkflowId: "missing"})
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func newTestClient(t *testing.T, allowAnonymous bool) (sdxv1.SDXServiceClient, func()) {
	t.Helper()

	cfg := cfgadapter.Load(cfgadapter.ModeAPI)
	cfg.AllowAnonymousAuth = allowAnonymous

	store := storage.NewMemoryStore()
	application := usecases.NewService(
		store,
		store,
		store,
		messaging.NoopPublisher{},
		newTestArtifactService(t, cfg.ServiceName),
		cfg.ServiceName,
		map[string]string{"database": "configured", "messaging": "configured"},
		"test",
	)
	logger := observability.NewLogger("grpc-test", "test", "ERROR")
	server := NewServer(cfg, application, logger)

	listener := bufconn.Listen(1024 * 1024)
	go func() {
		_ = server.Serve(listener)
	}()

	conn, err := gogrpc.NewClient("passthrough:///bufnet",
		gogrpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		gogrpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	return sdxv1.NewSDXServiceClient(conn), func() {
		require.NoError(t, conn.Close())
		server.Stop()
		require.NoError(t, listener.Close())
	}
}

func newTestArtifactService(t *testing.T, serviceName string) *artifacts.SignedArtifactService {
	t.Helper()

	service, err := artifacts.NewSignedArtifactService(serviceName, "https://artifacts.example.com", "signing-key", 15*time.Minute)
	require.NoError(t, err)
	return service
}

func tenantToken(tenantID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]string{"tenant_id": tenantID})
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

func outgoingContext(tenantID, idempotencyKey string) context.Context {
	entries := []string{
		"authorization", "Bearer grpc-test-token",
		strings.ToLower(domain.HeaderTenantID), tenantID,
	}
	if strings.TrimSpace(idempotencyKey) != "" {
		entries = append(entries, strings.ToLower(domain.HeaderIdempotencyKey), idempotencyKey)
	}

	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs(entries...))
}
