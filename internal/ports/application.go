package ports

import (
	"context"

	"github.com/stratogroup/neuroforge-sdx/internal/domain"
)

type Application interface {
	Login(context.Context, domain.LoginRequest) (domain.LoginResponse, error)
	RefreshToken(context.Context, domain.RefreshTokenRequest) (domain.LoginResponse, error)
	CreateWorkflow(context.Context, domain.WorkflowRequest) (domain.WorkflowResponse, error)
	ListWorkflows(context.Context, WorkflowFilter) (domain.WorkflowListResponse, error)
	GetWorkflow(context.Context, string) (domain.WorkflowResponse, error)
	CancelWorkflow(context.Context, string) error
	GetWorkflowArtifacts(context.Context, string) (domain.WorkflowArtifactsResponse, error)
	GeneratePRD(context.Context, domain.PRDRequest) (domain.PRDResponse, error)
	GetPRD(context.Context, string) (domain.PRDResponse, error)
	GenerateWireframes(context.Context, domain.DesignRequest) (domain.DesignResponse, error)
	ExtractTokens(context.Context, domain.ExtractTokensRequest) (domain.ExtractTokensResponse, error)
	GenerateCode(context.Context, domain.CodeRequest) (domain.CodeResponse, error)
	GenerateTests(context.Context, domain.GenerateTestsRequest) (domain.GenerateTestsResponse, error)
	SecurityScan(context.Context, domain.SecurityScanRequest) (domain.SecurityScanQueuedResponse, error)
	GetSecurityFindings(context.Context, string) (domain.SecurityScanResponse, error)
	PerformanceTest(context.Context, domain.PerformanceTestRequest) (domain.PerformanceStartResponse, error)
	GetPerformanceResults(context.Context, string) (domain.PerformanceTestResponse, error)
	HealthLive(context.Context) (domain.HealthLiveResponse, error)
	HealthReady(context.Context) (domain.HealthReadyResponse, error)
}
