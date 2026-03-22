package grpc

import (
	"encoding/json"
	"time"

	sdxv1 "github.com/stratogroup/neuroforge-sdx/api/proto/neuroforge/sdx/v1"
	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func workflowRequestFromProto(request *sdxv1.WorkflowRequest) domain.WorkflowRequest {
	if request == nil {
		return domain.WorkflowRequest{}
	}

	return domain.WorkflowRequest{
		Prompt:          request.GetPrompt(),
		Domain:          request.GetDomain(),
		GitStrategy:     domain.GitStrategy(request.GetGitStrategy()),
		TargetLanguages: append([]string(nil), request.GetTargetLanguages()...),
		Regions:         append([]string(nil), request.GetRegions()...),
		Metadata:        structToMap(request.GetMetadata()),
	}
}

func prdRequestFromProto(request *sdxv1.PRDRequest) domain.PRDRequest {
	if request == nil {
		return domain.PRDRequest{}
	}

	return domain.PRDRequest{
		Prompt:        request.GetPrompt(),
		Domain:        request.GetDomain(),
		ReferencePRDs: append([]string(nil), request.GetReferencePrds()...),
	}
}

func designRequestFromProto(request *sdxv1.DesignRequest) domain.DesignRequest {
	if request == nil {
		return domain.DesignRequest{}
	}

	return domain.DesignRequest{
		PRDID:       request.GetPrdId(),
		Style:       request.GetStyle(),
		ColorScheme: request.GetColorScheme(),
	}
}

func codeRequestFromProto(request *sdxv1.CodeRequest) domain.CodeRequest {
	if request == nil {
		return domain.CodeRequest{}
	}

	return domain.CodeRequest{
		Specification: codeSpecificationFromProto(request.GetSpecification()),
		Language:      request.GetLanguage(),
		Framework:     request.GetFramework(),
	}
}

func codeSpecificationFromProto(specification *sdxv1.CodeSpecification) domain.CodeSpecification {
	if specification == nil {
		return domain.CodeSpecification{}
	}

	return domain.CodeSpecification{
		Description:  specification.GetDescription(),
		Interface:    specification.GetInterface(),
		Requirements: append([]string(nil), specification.GetRequirements()...),
	}
}

func toProtoLoginResponse(response domain.LoginResponse) *sdxv1.LoginResponse {
	return &sdxv1.LoginResponse{
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
		ExpiresIn:    int32(response.ExpiresIn),
		TokenType:    response.TokenType,
	}
}

func workflowResponseToProto(response domain.WorkflowResponse) *sdxv1.WorkflowResponse {
	return &sdxv1.WorkflowResponse{
		WorkflowId:          response.WorkflowID,
		Status:              response.Status,
		CurrentStage:        string(response.CurrentStage),
		Progress:            response.Progress,
		CreatedAt:           timestampProto(response.CreatedAt),
		StartedAt:           timestampProto(response.StartedAt),
		CompletedAt:         timestampProto(response.CompletedAt),
		EstimatedCompletion: timestampProto(response.EstimatedCompletion),
		Artifacts:           response.Artifacts,
		Errors:              append([]string(nil), response.Errors...),
		Links: &sdxv1.WorkflowLinks{
			Self:      response.Links.Self,
			Artifacts: response.Links.Artifacts,
			Cancel:    response.Links.Cancel,
		},
	}
}

func workflowListResponseToProto(response domain.WorkflowListResponse) *sdxv1.ListWorkflowsResponse {
	workflows := make([]*sdxv1.WorkflowResponse, 0, len(response.Workflows))
	for _, workflow := range response.Workflows {
		workflows = append(workflows, workflowResponseToProto(workflow))
	}

	return &sdxv1.ListWorkflowsResponse{
		Workflows: workflows,
		Pagination: &sdxv1.Pagination{
			Page:  int32(response.Pagination.Page),
			Size:  int32(response.Pagination.Size),
			Total: int32(response.Pagination.Total),
			Pages: int32(response.Pagination.Pages),
		},
	}
}

func workflowArtifactsResponseToProto(response domain.WorkflowArtifactsResponse) *sdxv1.WorkflowArtifactsResponse {
	artifacts := make(map[string]*sdxv1.ArtifactDescriptor, len(response.Artifacts))
	for name, artifact := range response.Artifacts {
		artifacts[name] = &sdxv1.ArtifactDescriptor{
			Url:       artifact.URL,
			Type:      artifact.Type,
			Size:      int32(artifact.Size),
			CreatedAt: timestampProto(artifact.CreatedAt),
		}
	}

	return &sdxv1.WorkflowArtifactsResponse{Artifacts: artifacts}
}

func prdResponseToProto(response domain.PRDResponse) (*sdxv1.PRDResponse, error) {
	payload, err := mapToStruct(response.JSON)
	if err != nil {
		return nil, err
	}

	return &sdxv1.PRDResponse{
		PrdId:                 response.PRDID,
		Markdown:              response.Markdown,
		Json:                  payload,
		SuccessMetrics:        successMetricsToProto(response.SuccessMetrics),
		UserStories:           append([]string(nil), response.UserStories...),
		TechnicalRequirements: append([]string(nil), response.TechnicalRequirements...),
		CreatedAt:             timestampProto(response.CreatedAt),
	}, nil
}

func successMetricsToProto(metrics []domain.SuccessMetric) []*sdxv1.SuccessMetric {
	items := make([]*sdxv1.SuccessMetric, 0, len(metrics))
	for _, metric := range metrics {
		items = append(items, &sdxv1.SuccessMetric{
			Name:        metric.Name,
			Target:      metric.Target,
			Measurement: metric.Measurement,
		})
	}

	return items
}

func designResponseToProto(response domain.DesignResponse) *sdxv1.DesignResponse {
	return &sdxv1.DesignResponse{
		DesignId:    response.DesignID,
		Wireframes:  designAssetsToProto(response.Wireframes),
		HifiDesigns: designAssetsToProto(response.HifiDesigns),
		Tokens: &sdxv1.DesignTokens{
			Colors:     response.Tokens.Colors,
			Typography: response.Tokens.Typography,
			Spacing:    response.Tokens.Spacing,
		},
	}
}

func designAssetsToProto(assets []domain.DesignAsset) []*sdxv1.DesignAsset {
	items := make([]*sdxv1.DesignAsset, 0, len(assets))
	for _, asset := range assets {
		items = append(items, &sdxv1.DesignAsset{
			ScreenName: asset.ScreenName,
			Url:        asset.URL,
		})
	}

	return items
}

func extractTokensResponseToProto(response domain.ExtractTokensResponse) (*sdxv1.ExtractTokensResponse, error) {
	payload, err := mapToStruct(response.Tokens)
	if err != nil {
		return nil, err
	}

	return &sdxv1.ExtractTokensResponse{Tokens: payload}, nil
}

func codeResponseToProto(response domain.CodeResponse) *sdxv1.CodeResponse {
	return &sdxv1.CodeResponse{
		CodeId:   response.CodeID,
		Code:     response.Code,
		Language: response.Language,
		FilePath: response.FilePath,
		Tests:    response.Tests,
		Validation: &sdxv1.CodeValidation{
			Valid:  response.Validation.Valid,
			Errors: append([]string(nil), response.Validation.Errors...),
		},
	}
}

func securityScanResponseToProto(response domain.SecurityScanResponse) *sdxv1.SecurityScanResponse {
	findings := make([]*sdxv1.SecurityFinding, 0, len(response.Findings))
	for _, finding := range response.Findings {
		findings = append(findings, &sdxv1.SecurityFinding{
			FindingId:   finding.FindingID,
			Severity:    finding.Severity,
			Category:    finding.Category,
			Description: finding.Description,
			Location: &sdxv1.SecurityFindingLocation{
				File:   finding.Location.File,
				Line:   int32(finding.Location.Line),
				Column: int32(finding.Location.Column),
			},
			Remediation: finding.Remediation,
			Cwe:         append([]string(nil), finding.CWE...),
			CvssScore:   finding.CVSSScore,
		})
	}

	return &sdxv1.SecurityScanResponse{
		ScanId:   response.ScanID,
		Status:   response.Status,
		Findings: findings,
		Summary: &sdxv1.SecuritySummary{
			Critical: int32(response.Summary.Critical),
			High:     int32(response.Summary.High),
			Medium:   int32(response.Summary.Medium),
			Low:      int32(response.Summary.Low),
			Info:     int32(response.Summary.Info),
		},
		Sbom:      response.SBOM,
		Signature: response.Signature,
	}
}

func performanceTestResponseToProto(response domain.PerformanceTestResponse) *sdxv1.PerformanceTestResponse {
	thresholds := make([]*sdxv1.ThresholdResult, 0, len(response.Thresholds))
	for _, threshold := range response.Thresholds {
		thresholds = append(thresholds, &sdxv1.ThresholdResult{
			Metric:    threshold.Metric,
			Threshold: threshold.Threshold,
			Actual:    threshold.Actual,
			Passed:    threshold.Passed,
		})
	}

	return &sdxv1.PerformanceTestResponse{
		TestId: response.TestID,
		Status: response.Status,
		Metrics: &sdxv1.PerformanceMetrics{
			RequestsPerSecond: response.Metrics.RequestsPerSecond,
			LatencyP50:        response.Metrics.LatencyP50,
			LatencyP95:        response.Metrics.LatencyP95,
			LatencyP99:        response.Metrics.LatencyP99,
			ErrorRate:         response.Metrics.ErrorRate,
		},
		Passed:     response.Passed,
		Thresholds: thresholds,
	}
}

func mapToStruct(value map[string]any) (*structpb.Struct, error) {
	if len(value) == 0 {
		return nil, nil
	}

	normalized := map[string]any{}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &normalized); err != nil {
		return nil, err
	}

	return structpb.NewStruct(normalized)
}

func structToMap(value *structpb.Struct) map[string]any {
	if value == nil {
		return nil
	}

	return value.AsMap()
}

func timestampProto(value string) *timestamppb.Timestamp {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}

	return timestamppb.New(parsed)
}
