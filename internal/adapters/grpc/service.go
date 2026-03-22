package grpc

import (
	"context"

	sdxv1 "github.com/stratogroup/neuroforge-sdx/api/proto/neuroforge/sdx/v1"
	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stratogroup/neuroforge-sdx/internal/ports"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *Server) Login(ctx context.Context, request *sdxv1.LoginRequest) (*sdxv1.LoginResponse, error) {
	response, err := s.application.Login(ctx, domain.LoginRequest{
		Email:    request.GetEmail(),
		Password: request.GetPassword(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return toProtoLoginResponse(response), nil
}

func (s *Server) RefreshToken(ctx context.Context, request *sdxv1.RefreshTokenRequest) (*sdxv1.LoginResponse, error) {
	response, err := s.application.RefreshToken(ctx, domain.RefreshTokenRequest{
		RefreshToken: request.GetRefreshToken(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return toProtoLoginResponse(response), nil
}

func (s *Server) HealthLive(ctx context.Context, _ *emptypb.Empty) (*sdxv1.HealthResponse, error) {
	response, err := s.application.HealthLive(ctx)
	if err != nil {
		return nil, toStatusError(err)
	}

	return &sdxv1.HealthResponse{Status: response.Status}, nil
}

func (s *Server) HealthReady(ctx context.Context, _ *emptypb.Empty) (*sdxv1.ReadinessResponse, error) {
	response, err := s.application.HealthReady(ctx)
	if err != nil {
		return nil, toStatusError(err)
	}

	return &sdxv1.ReadinessResponse{
		Status: response.Status,
		Checks: response.Checks,
	}, nil
}

func (s *Server) CreateWorkflow(ctx context.Context, request *sdxv1.WorkflowRequest) (*sdxv1.WorkflowResponse, error) {
	response, err := s.application.CreateWorkflow(ctx, workflowRequestFromProto(request))
	if err != nil {
		return nil, toStatusError(err)
	}

	return workflowResponseToProto(response), nil
}

func (s *Server) GetWorkflow(ctx context.Context, request *sdxv1.GetWorkflowRequest) (*sdxv1.WorkflowResponse, error) {
	response, err := s.application.GetWorkflow(ctx, request.GetWorkflowId())
	if err != nil {
		return nil, toStatusError(err)
	}

	return workflowResponseToProto(response), nil
}

func (s *Server) ListWorkflows(ctx context.Context, request *sdxv1.ListWorkflowsRequest) (*sdxv1.ListWorkflowsResponse, error) {
	response, err := s.application.ListWorkflows(ctx, ports.WorkflowFilter{
		Page:   int(request.GetPage()),
		Size:   int(request.GetSize()),
		Status: request.GetStatus(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return workflowListResponseToProto(response), nil
}

func (s *Server) CancelWorkflow(ctx context.Context, request *sdxv1.GetWorkflowRequest) (*emptypb.Empty, error) {
	if err := s.application.CancelWorkflow(ctx, request.GetWorkflowId()); err != nil {
		return nil, toStatusError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) GetWorkflowArtifacts(ctx context.Context, request *sdxv1.GetWorkflowRequest) (*sdxv1.WorkflowArtifactsResponse, error) {
	response, err := s.application.GetWorkflowArtifacts(ctx, request.GetWorkflowId())
	if err != nil {
		return nil, toStatusError(err)
	}

	return workflowArtifactsResponseToProto(response), nil
}

func (s *Server) GeneratePRD(ctx context.Context, request *sdxv1.PRDRequest) (*sdxv1.PRDResponse, error) {
	response, err := s.application.GeneratePRD(ctx, prdRequestFromProto(request))
	if err != nil {
		return nil, toStatusError(err)
	}

	message, conversionErr := prdResponseToProto(response)
	if conversionErr != nil {
		return nil, toStatusError(conversionErr)
	}

	return message, nil
}

func (s *Server) GetPRD(ctx context.Context, request *sdxv1.GetPRDRequest) (*sdxv1.PRDResponse, error) {
	response, err := s.application.GetPRD(ctx, request.GetPrdId())
	if err != nil {
		return nil, toStatusError(err)
	}

	message, conversionErr := prdResponseToProto(response)
	if conversionErr != nil {
		return nil, toStatusError(conversionErr)
	}

	return message, nil
}

func (s *Server) GenerateWireframes(ctx context.Context, request *sdxv1.DesignRequest) (*sdxv1.DesignResponse, error) {
	response, err := s.application.GenerateWireframes(ctx, designRequestFromProto(request))
	if err != nil {
		return nil, toStatusError(err)
	}

	return designResponseToProto(response), nil
}

func (s *Server) ExtractTokens(ctx context.Context, request *sdxv1.ExtractTokensRequest) (*sdxv1.ExtractTokensResponse, error) {
	response, err := s.application.ExtractTokens(ctx, domain.ExtractTokensRequest{
		DesignID: request.GetDesignId(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	message, conversionErr := extractTokensResponseToProto(response)
	if conversionErr != nil {
		return nil, toStatusError(conversionErr)
	}

	return message, nil
}

func (s *Server) GenerateCode(ctx context.Context, request *sdxv1.CodeRequest) (*sdxv1.CodeResponse, error) {
	response, err := s.application.GenerateCode(ctx, codeRequestFromProto(request))
	if err != nil {
		return nil, toStatusError(err)
	}

	return codeResponseToProto(response), nil
}

func (s *Server) GenerateTests(ctx context.Context, request *sdxv1.GenerateTestsRequest) (*sdxv1.GenerateTestsResponse, error) {
	response, err := s.application.GenerateTests(ctx, domain.GenerateTestsRequest{
		Code:     request.GetCode(),
		Language: request.GetLanguage(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &sdxv1.GenerateTestsResponse{
		Tests:    response.Tests,
		Coverage: response.Coverage,
	}, nil
}

func (s *Server) SecurityScan(ctx context.Context, request *sdxv1.SecurityScanRequest) (*sdxv1.SecurityScanQueuedResponse, error) {
	response, err := s.application.SecurityScan(ctx, domain.SecurityScanRequest{
		Target:    request.GetTarget(),
		ScanTypes: append([]string(nil), request.GetScanTypes()...),
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &sdxv1.SecurityScanQueuedResponse{
		ScanId: response.ScanID,
		Status: response.Status,
	}, nil
}

func (s *Server) GetSecurityFindings(ctx context.Context, request *sdxv1.GetSecurityFindingsRequest) (*sdxv1.SecurityScanResponse, error) {
	response, err := s.application.GetSecurityFindings(ctx, request.GetScanId())
	if err != nil {
		return nil, toStatusError(err)
	}

	return securityScanResponseToProto(response), nil
}

func (s *Server) PerformanceTest(ctx context.Context, request *sdxv1.PerformanceTestRequest) (*sdxv1.PerformanceStartResponse, error) {
	response, err := s.application.PerformanceTest(ctx, domain.PerformanceTestRequest{
		Endpoint:     request.GetEndpoint(),
		Duration:     request.GetDuration(),
		VirtualUsers: int(request.GetVirtualUsers()),
		Regions:      append([]string(nil), request.GetRegions()...),
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &sdxv1.PerformanceStartResponse{TestId: response.TestID}, nil
}

func (s *Server) GetPerformanceResults(ctx context.Context, request *sdxv1.GetPerformanceResultsRequest) (*sdxv1.PerformanceTestResponse, error) {
	response, err := s.application.GetPerformanceResults(ctx, request.GetTestId())
	if err != nil {
		return nil, toStatusError(err)
	}

	return performanceTestResponseToProto(response), nil
}
