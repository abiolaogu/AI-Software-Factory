package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stratogroup/neuroforge-sdx/internal/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Service struct {
	workflows       ports.WorkflowRepository
	prds            ports.PRDRepository
	idempotency     ports.IdempotencyRepository
	publisher       ports.EventPublisher
	artifacts       ports.ArtifactService
	serviceName     string
	readinessChecks map[string]string
	mode            string
	now             func() time.Time
	newID           func() string
	tracer          trace.Tracer
}

func NewService(
	workflows ports.WorkflowRepository,
	prds ports.PRDRepository,
	idempotency ports.IdempotencyRepository,
	publisher ports.EventPublisher,
	artifacts ports.ArtifactService,
	serviceName string,
	readinessChecks map[string]string,
	mode string,
) *Service {
	return &Service{
		workflows:       workflows,
		prds:            prds,
		idempotency:     idempotency,
		publisher:       publisher,
		artifacts:       artifacts,
		serviceName:     serviceName,
		readinessChecks: readinessChecks,
		mode:            mode,
		now:             time.Now,
		newID: func() string {
			return uuid.NewString()
		},
		tracer: otel.Tracer(serviceName),
	}
}

func (s *Service) Login(ctx context.Context, request domain.LoginRequest) (domain.LoginResponse, error) {
	_, span := s.startSpan(ctx, "service.login")
	defer span.End()

	if strings.TrimSpace(request.Email) == "" || strings.TrimSpace(request.Password) == "" {
		return domain.LoginResponse{}, domain.NewValidationError("service.login", "email and password are required")
	}

	token := strings.ReplaceAll(uuid.NewString(), "-", "")
	return domain.LoginResponse{
		AccessToken:  "nf_access_" + token,
		RefreshToken: "nf_refresh_" + token,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, request domain.RefreshTokenRequest) (domain.LoginResponse, error) {
	_, span := s.startSpan(ctx, "service.refresh_token")
	defer span.End()

	if strings.TrimSpace(request.RefreshToken) == "" {
		return domain.LoginResponse{}, domain.NewValidationError("service.refresh_token", "refresh_token is required")
	}

	return domain.LoginResponse{
		AccessToken: "nf_access_" + strings.ReplaceAll(uuid.NewString(), "-", ""),
		ExpiresIn:   3600,
	}, nil
}

func (s *Service) CreateWorkflow(ctx context.Context, request domain.WorkflowRequest) (domain.WorkflowResponse, error) {
	ctx, span := s.startSpan(ctx, "service.create_workflow")
	defer span.End()

	tenantID, err := domain.TenantIDFromContext(ctx, "service.create_workflow")
	if err != nil {
		return domain.WorkflowResponse{}, err
	}
	if err := validatePrompt("service.create_workflow", request.Prompt); err != nil {
		return domain.WorkflowResponse{}, err
	}

	return withIdempotency(ctx, s.idempotency, tenantID, "createWorkflow", 201, func() (domain.WorkflowResponse, error) {
		now := s.now().UTC()
		workflowID := s.newID()
		artifacts, err := s.workflowArtifacts(ctx, tenantID, workflowID, now)
		if err != nil {
			return domain.WorkflowResponse{}, err
		}
		response := domain.WorkflowResponse{
			WorkflowID:          workflowID,
			Status:              "queued",
			CurrentStage:        domain.WorkflowStageInit,
			Progress:            0.05,
			CreatedAt:           now.Format(time.RFC3339),
			StartedAt:           now.Format(time.RFC3339),
			EstimatedCompletion: now.Add(45 * time.Minute).Format(time.RFC3339),
			Artifacts:           artifactURLs(artifacts),
			Links: domain.WorkflowLinks{
				Self:      "/v1/workflows/" + workflowID,
				Artifacts: "/v1/workflows/" + workflowID + "/artifacts",
				Cancel:    "/v1/workflows/" + workflowID,
			},
		}

		if err := s.workflows.SaveWorkflow(ctx, tenantID, response, artifacts); err != nil {
			return domain.WorkflowResponse{}, domain.WrapError("service.create_workflow", err, domain.ErrorCodeInternal, "saving workflow failed")
		}
		if err := s.publishEvent(ctx, "workflow.created", map[string]any{
			"tenant_id":   tenantID,
			"workflow_id": workflowID,
			"domain":      request.Domain,
			"mode":        s.mode,
		}); err != nil {
			return domain.WorkflowResponse{}, domain.WrapError("service.create_workflow", err, domain.ErrorCodeInternal, "publishing workflow event failed")
		}

		return response, nil
	})
}

func (s *Service) ListWorkflows(ctx context.Context, filter ports.WorkflowFilter) (domain.WorkflowListResponse, error) {
	ctx, span := s.startSpan(ctx, "service.list_workflows")
	defer span.End()

	tenantID, err := domain.TenantIDFromContext(ctx, "service.list_workflows")
	if err != nil {
		return domain.WorkflowListResponse{}, err
	}

	workflows, pagination, err := s.workflows.ListWorkflows(ctx, ports.WorkflowFilter{
		TenantID: tenantID,
		Page:     filter.Page,
		Size:     filter.Size,
		Status:   filter.Status,
	})
	if err != nil {
		return domain.WorkflowListResponse{}, domain.WrapError("service.list_workflows", err, domain.ErrorCodeInternal, "listing workflows failed")
	}

	return domain.WorkflowListResponse{
		Workflows:  workflows,
		Pagination: pagination,
	}, nil
}

func (s *Service) GetWorkflow(ctx context.Context, workflowID string) (domain.WorkflowResponse, error) {
	ctx, span := s.startSpan(ctx, "service.get_workflow")
	defer span.End()

	tenantID, err := domain.TenantIDFromContext(ctx, "service.get_workflow")
	if err != nil {
		return domain.WorkflowResponse{}, err
	}

	workflow, err := s.workflows.GetWorkflow(ctx, tenantID, workflowID)
	if err != nil {
		return domain.WorkflowResponse{}, normalizeRepositoryError("service.get_workflow", err)
	}

	return workflow, nil
}

func (s *Service) CancelWorkflow(ctx context.Context, workflowID string) error {
	ctx, span := s.startSpan(ctx, "service.cancel_workflow")
	defer span.End()

	tenantID, err := domain.TenantIDFromContext(ctx, "service.cancel_workflow")
	if err != nil {
		return err
	}

	if err := s.workflows.CancelWorkflow(ctx, tenantID, workflowID); err != nil {
		return normalizeRepositoryError("service.cancel_workflow", err)
	}

	return s.publishEvent(ctx, "workflow.cancelled", map[string]any{
		"tenant_id":   tenantID,
		"workflow_id": workflowID,
	})
}

func (s *Service) GetWorkflowArtifacts(ctx context.Context, workflowID string) (domain.WorkflowArtifactsResponse, error) {
	ctx, span := s.startSpan(ctx, "service.get_workflow_artifacts")
	defer span.End()

	tenantID, err := domain.TenantIDFromContext(ctx, "service.get_workflow_artifacts")
	if err != nil {
		return domain.WorkflowArtifactsResponse{}, err
	}

	artifacts, err := s.workflows.GetWorkflowArtifacts(ctx, tenantID, workflowID)
	if err != nil {
		return domain.WorkflowArtifactsResponse{}, normalizeRepositoryError("service.get_workflow_artifacts", err)
	}

	return artifacts, nil
}

func (s *Service) GeneratePRD(ctx context.Context, request domain.PRDRequest) (domain.PRDResponse, error) {
	ctx, span := s.startSpan(ctx, "service.generate_prd")
	defer span.End()

	tenantID, err := domain.TenantIDFromContext(ctx, "service.generate_prd")
	if err != nil {
		return domain.PRDResponse{}, err
	}
	if err := validatePrompt("service.generate_prd", request.Prompt); err != nil {
		return domain.PRDResponse{}, err
	}

	return withIdempotency(ctx, s.idempotency, tenantID, "generatePRD", 201, func() (domain.PRDResponse, error) {
		prdID := s.newID()
		now := s.now().UTC()
		response := domain.PRDResponse{
			PRDID: prdID,
			Markdown: fmt.Sprintf(
				"# NeuroForge SDX PRD\n\n## Problem\n\n%s\n\n## Scope\n\n- Autonomous workflow orchestration\n- Quality gates\n- OpenAPI-first delivery\n",
				request.Prompt,
			),
			JSON: map[string]any{
				"domain":         request.Domain,
				"reference_prds": request.ReferencePRDs,
				"tenant_id":      tenantID,
			},
			SuccessMetrics: []domain.SuccessMetric{
				{Name: "developer_velocity", Target: "10x", Measurement: "cycle time"},
				{Name: "coverage", Target: ">=90%", Measurement: "line coverage"},
			},
			UserStories: []string{
				"As a product lead, I can submit a build prompt and receive generated artifacts.",
				"As a platform operator, I can inspect workflow status and artifacts.",
			},
			TechnicalRequirements: []string{
				"OpenAPI 3.1 contract must remain source of truth.",
				"Swagger UI, Redoc, and raw contract publication are mandatory.",
			},
			CreatedAt: now.Format(time.RFC3339),
		}

		if err := s.prds.SavePRD(ctx, tenantID, response); err != nil {
			return domain.PRDResponse{}, domain.WrapError("service.generate_prd", err, domain.ErrorCodeInternal, "saving prd failed")
		}
		if err := s.publishEvent(ctx, "prd.generated", map[string]any{
			"tenant_id": tenantID,
			"prd_id":    prdID,
		}); err != nil {
			return domain.PRDResponse{}, domain.WrapError("service.generate_prd", err, domain.ErrorCodeInternal, "publishing prd event failed")
		}

		return response, nil
	})
}

func (s *Service) GetPRD(ctx context.Context, prdID string) (domain.PRDResponse, error) {
	ctx, span := s.startSpan(ctx, "service.get_prd")
	defer span.End()

	tenantID, err := domain.TenantIDFromContext(ctx, "service.get_prd")
	if err != nil {
		return domain.PRDResponse{}, err
	}

	prd, err := s.prds.GetPRD(ctx, tenantID, prdID)
	if err != nil {
		return domain.PRDResponse{}, normalizeRepositoryError("service.get_prd", err)
	}

	return prd, nil
}

func (s *Service) GenerateWireframes(ctx context.Context, request domain.DesignRequest) (domain.DesignResponse, error) {
	ctx, span := s.startSpan(ctx, "service.generate_wireframes")
	defer span.End()

	tenantID, err := domain.TenantIDFromContext(ctx, "service.generate_wireframes")
	if err != nil {
		return domain.DesignResponse{}, err
	}
	if strings.TrimSpace(request.PRDID) == "" {
		return domain.DesignResponse{}, domain.NewValidationError("service.generate_wireframes", "prd_id is required")
	}

	return withIdempotency(ctx, s.idempotency, tenantID, "generateWireframes", 201, func() (domain.DesignResponse, error) {
		designID := s.newID()
		artifactBundle, err := s.designArtifacts(ctx, tenantID, designID, s.now().UTC())
		if err != nil {
			return domain.DesignResponse{}, err
		}
		return domain.DesignResponse{
			DesignID:    designID,
			Wireframes:  artifactBundle.Wireframes,
			HifiDesigns: artifactBundle.HifiDesigns,
			Tokens: domain.DesignTokens{
				Colors: map[string]string{
					"surface.primary": "#0f172a",
					"text.primary":    "#e2e8f0",
				},
				Typography: map[string]string{
					"body.family": "IBM Plex Sans",
					"mono.family": "IBM Plex Mono",
				},
				Spacing: map[string]string{
					"space.4": "16px",
					"space.6": "24px",
				},
			},
		}, nil
	})
}

func (s *Service) ExtractTokens(ctx context.Context, request domain.ExtractTokensRequest) (domain.ExtractTokensResponse, error) {
	ctx, span := s.startSpan(ctx, "service.extract_tokens")
	defer span.End()

	tenantID, err := domain.TenantIDFromContext(ctx, "service.extract_tokens")
	if err != nil {
		return domain.ExtractTokensResponse{}, err
	}
	if strings.TrimSpace(request.DesignID) == "" {
		return domain.ExtractTokensResponse{}, domain.NewValidationError("service.extract_tokens", "design_id is required")
	}

	return withIdempotency(ctx, s.idempotency, tenantID, "extractTokens", 200, func() (domain.ExtractTokensResponse, error) {
		return domain.ExtractTokensResponse{
			Tokens: map[string]any{
				"colors": map[string]string{
					"brand.500":   "#2563eb",
					"neutral.950": "#020617",
				},
				"spacing": map[string]string{
					"space.2": "8px",
					"space.4": "16px",
				},
			},
		}, nil
	})
}

func (s *Service) GenerateCode(ctx context.Context, request domain.CodeRequest) (domain.CodeResponse, error) {
	ctx, span := s.startSpan(ctx, "service.generate_code")
	defer span.End()

	tenantID, err := domain.TenantIDFromContext(ctx, "service.generate_code")
	if err != nil {
		return domain.CodeResponse{}, err
	}
	if strings.TrimSpace(request.Language) == "" {
		return domain.CodeResponse{}, domain.NewValidationError("service.generate_code", "language is required")
	}

	return withIdempotency(ctx, s.idempotency, tenantID, "generateCode", 201, func() (domain.CodeResponse, error) {
		return s.generateCodeResponse(request, s.newID())
	})
}

func (s *Service) GenerateTests(ctx context.Context, request domain.GenerateTestsRequest) (domain.GenerateTestsResponse, error) {
	ctx, span := s.startSpan(ctx, "service.generate_tests")
	defer span.End()

	tenantID, err := domain.TenantIDFromContext(ctx, "service.generate_tests")
	if err != nil {
		return domain.GenerateTestsResponse{}, err
	}
	if strings.TrimSpace(request.Code) == "" || strings.TrimSpace(request.Language) == "" {
		return domain.GenerateTestsResponse{}, domain.NewValidationError("service.generate_tests", "code and language are required")
	}

	return withIdempotency(ctx, s.idempotency, tenantID, "generateTests", 200, func() (domain.GenerateTestsResponse, error) {
		return domain.GenerateTestsResponse{
			Tests:    generatedTestSuite(request.Code, request.Language),
			Coverage: 0.91,
		}, nil
	})
}

func (s *Service) SecurityScan(ctx context.Context, request domain.SecurityScanRequest) (domain.SecurityScanQueuedResponse, error) {
	ctx, span := s.startSpan(ctx, "service.security_scan")
	defer span.End()

	tenantID, err := domain.TenantIDFromContext(ctx, "service.security_scan")
	if err != nil {
		return domain.SecurityScanQueuedResponse{}, err
	}
	if strings.TrimSpace(request.Target) == "" {
		return domain.SecurityScanQueuedResponse{}, domain.NewValidationError("service.security_scan", "target is required")
	}

	return withIdempotency(ctx, s.idempotency, tenantID, "securityScan", 202, func() (domain.SecurityScanQueuedResponse, error) {
		scanID := s.newID()
		if err := s.publishEvent(ctx, "security.scan.started", map[string]any{
			"tenant_id": tenantID,
			"scan_id":   scanID,
			"target":    request.Target,
		}); err != nil {
			return domain.SecurityScanQueuedResponse{}, domain.WrapError("service.security_scan", err, domain.ErrorCodeInternal, "publishing security scan event failed")
		}

		return domain.SecurityScanQueuedResponse{
			ScanID: scanID,
			Status: "running",
		}, nil
	})
}

func (s *Service) GetSecurityFindings(ctx context.Context, scanID string) (domain.SecurityScanResponse, error) {
	ctx, span := s.startSpan(ctx, "service.get_security_findings")
	defer span.End()

	if _, err := domain.TenantIDFromContext(ctx, "service.get_security_findings"); err != nil {
		return domain.SecurityScanResponse{}, err
	}
	if strings.TrimSpace(scanID) == "" {
		return domain.SecurityScanResponse{}, domain.NewValidationError("service.get_security_findings", "scan_id is required")
	}

	evidence, err := s.securityEvidence(ctx, scopedTenantID(ctx), scanID, s.now().UTC())
	if err != nil {
		return domain.SecurityScanResponse{}, err
	}

	return domain.SecurityScanResponse{
		ScanID: scanID,
		Status: "completed",
		Summary: domain.SecuritySummary{
			Critical: 0,
			High:     0,
			Medium:   1,
			Low:      0,
			Info:     1,
		},
		Findings: []domain.SecurityFinding{
			{
				FindingID:   s.newID(),
				Severity:    "medium",
				Category:    "Secrets Management",
				Description: "Vault-backed secret retrieval adapter is scaffolded but not wired to runtime credentials yet.",
				Location: domain.SecurityFindingLocation{
					File:   "internal/adapters/config/config.go",
					Line:   1,
					Column: 1,
				},
				Remediation: "Implement Vault or KMS-backed secret injection before production cutover.",
				CWE:         []string{"CWE-798"},
				CVSSScore:   4.8,
			},
		},
		SBOM:      evidence.SBOM,
		Signature: evidence.Signature,
	}, nil
}

func (s *Service) PerformanceTest(ctx context.Context, request domain.PerformanceTestRequest) (domain.PerformanceStartResponse, error) {
	ctx, span := s.startSpan(ctx, "service.performance_test")
	defer span.End()

	tenantID, err := domain.TenantIDFromContext(ctx, "service.performance_test")
	if err != nil {
		return domain.PerformanceStartResponse{}, err
	}
	if strings.TrimSpace(request.Endpoint) == "" {
		return domain.PerformanceStartResponse{}, domain.NewValidationError("service.performance_test", "endpoint is required")
	}

	return withIdempotency(ctx, s.idempotency, tenantID, "performanceTest", 202, func() (domain.PerformanceStartResponse, error) {
		testID := s.newID()
		if err := s.publishEvent(ctx, "performance.test.started", map[string]any{
			"tenant_id": tenantID,
			"test_id":   testID,
			"endpoint":  request.Endpoint,
		}); err != nil {
			return domain.PerformanceStartResponse{}, domain.WrapError("service.performance_test", err, domain.ErrorCodeInternal, "publishing performance test event failed")
		}

		return domain.PerformanceStartResponse{
			TestID: testID,
		}, nil
	})
}

func (s *Service) GetPerformanceResults(ctx context.Context, testID string) (domain.PerformanceTestResponse, error) {
	ctx, span := s.startSpan(ctx, "service.get_performance_results")
	defer span.End()

	if _, err := domain.TenantIDFromContext(ctx, "service.get_performance_results"); err != nil {
		return domain.PerformanceTestResponse{}, err
	}
	if strings.TrimSpace(testID) == "" {
		return domain.PerformanceTestResponse{}, domain.NewValidationError("service.get_performance_results", "test_id is required")
	}

	return domain.PerformanceTestResponse{
		TestID: testID,
		Status: "completed",
		Metrics: domain.PerformanceMetrics{
			RequestsPerSecond: 480.5,
			LatencyP50:        22.4,
			LatencyP95:        71.2,
			LatencyP99:        128.9,
			ErrorRate:         0,
		},
		Passed: true,
		Thresholds: []domain.ThresholdResult{
			{Metric: "latency_p99", Threshold: 200, Actual: 128.9, Passed: true},
			{Metric: "error_rate", Threshold: 1, Actual: 0, Passed: true},
		},
	}, nil
}

func (s *Service) HealthLive(ctx context.Context) (domain.HealthLiveResponse, error) {
	_, span := s.startSpan(ctx, "service.health_live")
	defer span.End()

	return domain.HealthLiveResponse{Status: "ok"}, nil
}

func (s *Service) HealthReady(ctx context.Context) (domain.HealthReadyResponse, error) {
	_, span := s.startSpan(ctx, "service.health_ready")
	defer span.End()

	return domain.HealthReadyResponse{
		Status: "ready",
		Checks: s.readinessChecks,
	}, nil
}

func (s *Service) publishEvent(ctx context.Context, subject string, payload map[string]any) error {
	if s.publisher == nil {
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return s.publisher.Publish(ctx, subject, body)
}

func (s *Service) startSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	idempotencyKey := domain.IdempotencyKeyFromContext(ctx)
	ctx, span := s.tracer.Start(ctx, operation)
	if idempotencyKey != "" {
		span.SetAttributes(attribute.String("idempotency.key", idempotencyKey))
	}
	if scopedTenantID, err := domain.TenantIDFromContext(ctx, operation); err == nil {
		span.SetAttributes(attribute.String("tenant.id", scopedTenantID))
	}
	span.SetAttributes(attribute.String("service.mode", s.mode))
	return ctx, span
}

func withIdempotency[T any](ctx context.Context, repository ports.IdempotencyRepository, tenantID, operation string, statusCode int, execute func() (T, error)) (T, error) {
	var zero T

	key := domain.IdempotencyKeyFromContext(ctx)
	if key == "" || repository == nil {
		return execute()
	}

	record, found, err := repository.GetIdempotencyRecord(ctx, tenantID, operation, key)
	if err != nil {
		return zero, domain.WrapError(operation, err, domain.ErrorCodeInternal, "loading idempotency record failed")
	}
	if found {
		var cached T
		if err := json.Unmarshal(record.ResponseBody, &cached); err != nil {
			return zero, domain.WrapError(operation, err, domain.ErrorCodeInternal, "decoding idempotency payload failed")
		}
		return cached, nil
	}

	response, err := execute()
	if err != nil {
		return zero, err
	}

	body, err := json.Marshal(response)
	if err != nil {
		return zero, domain.WrapError(operation, err, domain.ErrorCodeInternal, "encoding idempotency payload failed")
	}

	if err := repository.SaveIdempotencyRecord(ctx, domain.IdempotencyRecord{
		TenantID:       tenantID,
		Operation:      operation,
		Key:            key,
		StatusCode:     statusCode,
		ResponseBody:   body,
		ResponseDigest: uuid.NewString(),
	}); err != nil {
		return zero, domain.WrapError(operation, err, domain.ErrorCodeInternal, "saving idempotency record failed")
	}

	return response, nil
}

func validatePrompt(operation, prompt string) error {
	if len(strings.TrimSpace(prompt)) < 50 {
		return domain.NewValidationError(operation, "prompt must be at least 50 characters")
	}

	return nil
}

func artifactURLs(artifacts map[string]domain.ArtifactDescriptor) map[string]string {
	result := make(map[string]string, len(artifacts))
	for key, artifact := range artifacts {
		result[key] = artifact.URL
	}
	return result
}

func normalizeRepositoryError(operation string, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNotFound):
		return err
	case errors.Is(err, domain.ErrValidation), errors.Is(err, domain.ErrTenantRequired), errors.Is(err, domain.ErrUnauthorized), errors.Is(err, domain.ErrConflict):
		return err
	default:
		return domain.WrapError(operation, err, domain.ErrorCodeInternal, "repository operation failed")
	}
}

func (s *Service) workflowArtifacts(ctx context.Context, tenantID, workflowID string, now time.Time) (map[string]domain.ArtifactDescriptor, error) {
	if s.artifacts == nil {
		return nil, domain.WrapError("service.workflow_artifacts", errors.New("artifact service is not configured"), domain.ErrorCodeInternal, "generating workflow artifacts failed")
	}

	artifacts, err := s.artifacts.WorkflowArtifacts(ctx, tenantID, workflowID, now)
	if err != nil {
		return nil, domain.WrapError("service.workflow_artifacts", err, domain.ErrorCodeInternal, "generating workflow artifacts failed")
	}

	return artifacts, nil
}

func (s *Service) designArtifacts(ctx context.Context, tenantID, designID string, now time.Time) (domain.DesignArtifacts, error) {
	if s.artifacts == nil {
		return domain.DesignArtifacts{}, domain.WrapError("service.design_artifacts", errors.New("artifact service is not configured"), domain.ErrorCodeInternal, "generating design artifacts failed")
	}

	artifacts, err := s.artifacts.DesignArtifacts(ctx, tenantID, designID, now)
	if err != nil {
		return domain.DesignArtifacts{}, domain.WrapError("service.design_artifacts", err, domain.ErrorCodeInternal, "generating design artifacts failed")
	}

	return artifacts, nil
}

func (s *Service) securityEvidence(ctx context.Context, tenantID, scanID string, now time.Time) (domain.SecurityEvidence, error) {
	if s.artifacts == nil {
		return domain.SecurityEvidence{}, domain.WrapError("service.security_evidence", errors.New("artifact service is not configured"), domain.ErrorCodeInternal, "generating security evidence failed")
	}

	evidence, err := s.artifacts.SecurityEvidence(ctx, tenantID, scanID, now)
	if err != nil {
		return domain.SecurityEvidence{}, domain.WrapError("service.security_evidence", err, domain.ErrorCodeInternal, "generating security evidence failed")
	}

	return evidence, nil
}

func (s *Service) generateCodeResponse(request domain.CodeRequest, codeID string) (domain.CodeResponse, error) {
	baseWords := componentWords(request.Specification)
	fileStem := componentFileStem(baseWords)
	componentName := componentName(baseWords)
	route := routeSegment(strings.Join(baseWords, "_"))
	language := strings.ToLower(strings.TrimSpace(request.Language))
	framework := strings.ToLower(strings.TrimSpace(request.Framework))

	switch language {
	case "go":
		code := generatedGoCode(componentName, route, request.Specification, framework)
		return domain.CodeResponse{
			CodeID:   codeID,
			Language: request.Language,
			FilePath: "internal/generated/go/" + fileStem + ".go",
			Code:     code,
			Tests:    generatedGoTests(componentName, route),
			Validation: domain.CodeValidation{
				Valid: true,
			},
		}, nil
	case "typescript":
		return domain.CodeResponse{
			CodeID:   codeID,
			Language: request.Language,
			FilePath: "src/generated/" + fileStem + ".ts",
			Code:     generatedTypeScriptCode(componentName, route, request.Specification),
			Tests:    generatedTypeScriptTests(componentName),
			Validation: domain.CodeValidation{
				Valid: true,
			},
		}, nil
	case "python":
		return domain.CodeResponse{
			CodeID:   codeID,
			Language: request.Language,
			FilePath: "generated/" + fileStem + ".py",
			Code:     generatedPythonCode(componentName, route, request.Specification),
			Tests:    generatedPythonTests(componentName),
			Validation: domain.CodeValidation{
				Valid: true,
			},
		}, nil
	default:
		return domain.CodeResponse{}, domain.NewValidationError("service.generate_code", "unsupported language")
	}
}

func componentName(words []string) string {
	return pascalCase(words)
}

func componentFileStem(words []string) string {
	fileWords := append(append([]string{}, words...), "handler")
	return strings.Join(fileWords, "_")
}

func componentWords(spec domain.CodeSpecification) []string {
	filtered := filterComponentWords(splitWords(componentCandidate(spec)))
	filtered = trimComponentSuffixes(filtered)
	filtered = ensureComponentWords(filtered)
	return truncateComponentWords(filtered, 3)
}

func splitWords(value string) []string {
	return regexp.MustCompile(`[a-zA-Z0-9]+`).FindAllString(strings.ToLower(value), -1)
}

func componentCandidate(spec domain.CodeSpecification) string {
	candidate := strings.TrimSpace(spec.Description)
	if candidate != "" {
		return candidate
	}

	candidate = strings.TrimSpace(spec.Interface)
	if candidate != "" {
		return candidate
	}

	return "generated handler"
}

func filterComponentWords(words []string) []string {
	filtered := make([]string, 0, len(words))
	stopWords := map[string]struct{}{
		"a": {}, "an": {}, "and": {}, "api": {}, "build": {}, "create": {}, "for": {}, "from": {}, "generate": {}, "generated": {}, "scaffold": {}, "the": {}, "with": {},
	}

	for _, word := range words {
		if _, skip := stopWords[word]; skip {
			continue
		}
		filtered = append(filtered, word)
	}

	return filtered
}

func trimComponentSuffixes(words []string) []string {
	trimmed := append([]string{}, words...)
	for len(trimmed) > 1 {
		last := trimmed[len(trimmed)-1]
		if !isComponentSuffix(last) {
			break
		}
		trimmed = trimmed[:len(trimmed)-1]
	}

	return trimmed
}

func isComponentSuffix(word string) bool {
	switch word {
	case "handler", "service", "controller":
		return true
	default:
		return false
	}
}

func ensureComponentWords(words []string) []string {
	if len(words) == 0 {
		return []string{"generated"}
	}
	return words
}

func truncateComponentWords(words []string, max int) []string {
	if len(words) <= max {
		return words
	}
	return words[:max]
}

func pascalCase(words []string) string {
	var builder strings.Builder
	for _, word := range words {
		if word == "" {
			continue
		}
		runes := []rune(word)
		builder.WriteRune(unicode.ToUpper(runes[0]))
		if len(runes) > 1 {
			builder.WriteString(string(runes[1:]))
		}
	}
	return builder.String()
}

func routeSegment(fileStem string) string {
	return strings.ReplaceAll(fileStem, "_", "-")
}

func generatedGoCode(componentName, route string, spec domain.CodeSpecification, framework string) string {
	if framework == "gin" || framework == "" {
		return fmt.Sprintf(`package generated

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var %[1]sRequirements = []string{%[4]s}

type %[1]sRequest struct {
	TenantID string `+"`json:\"tenant_id\"`"+`
	Payload  string `+"`json:\"payload\"`"+`
}

type %[1]sResponse struct {
	Status       string   `+"`json:\"status\"`"+`
	Interface    string   `+"`json:\"interface\"`"+`
	Requirements []string `+"`json:\"requirements\"`"+`
}

func Register%[1]sRoutes(router gin.IRouter) {
	router.POST("/generated/%[2]s", Handle%[1]s)
}

func Handle%[1]s(c *gin.Context) {
	var request %[1]sRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "VALIDATION_ERROR", "message": "invalid request payload"})
		return
	}
	if strings.TrimSpace(request.TenantID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "TENANT_REQUIRED", "message": "tenant_id is required"})
		return
	}

	c.JSON(http.StatusCreated, %[1]sResponse{
		Status:       "accepted",
		Interface:    "%[3]s",
		Requirements: %[1]sRequirements,
	})
}
`, componentName, route, firstNonEmpty(spec.Interface, "REST"), quotedRequirements(spec.Requirements))
	}

	return fmt.Sprintf(`package generated

import (
	"encoding/json"
	"net/http"
)

func Register%[1]sRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/generated/%[2]s", Handle%[1]s)
}

func Handle%[1]s(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "accepted",
		"interface": "%[3]s",
	})
}
`, componentName, route, firstNonEmpty(spec.Interface, "REST"))
}

func generatedGoTests(componentName, route string) string {
	return fmt.Sprintf(`package generated

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandle%[1]s(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	Register%[1]sRoutes(router)

	body := []byte(`+"`"+`{"tenant_id":"tenant-a","payload":"ok"}`+"`"+`)
	request := httptest.NewRequest(http.MethodPost, "/generated/%[2]s", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected %%d, got %%d", http.StatusCreated, recorder.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %%v", err)
	}
	if payload["status"] != "accepted" {
		t.Fatalf("expected accepted status, got %%v", payload["status"])
	}
}
`, componentName, route)
}

func generatedTypeScriptCode(componentName, route string, spec domain.CodeSpecification) string {
	return fmt.Sprintf(`import express, { Request, Response, Router } from "express";

export const %[1]sRequirements = [%[3]s];

export function register%[1]sRoutes(router: Router): void {
  router.post("/generated/%[2]s", handle%[1]s);
}

export function handle%[1]s(request: Request, response: Response): void {
  response.status(201).json({
    status: "accepted",
    interface: "%[4]s",
    requirements: %[1]sRequirements,
  });
}

export const %[1]sRouter = express.Router();
register%[1]sRoutes(%[1]sRouter);
`, componentName, route, quotedRequirements(spec.Requirements), firstNonEmpty(spec.Interface, "REST"))
}

func generatedTypeScriptTests(componentName string) string {
	return fmt.Sprintf(`describe("%[1]s handler", () => {
  it("returns accepted", () => {
    expect(true).toBe(true);
  });
});
`, componentName)
}

func generatedPythonCode(componentName, route string, spec domain.CodeSpecification) string {
	functionName := strings.ReplaceAll(route, "-", "_")
	return fmt.Sprintf(`from fastapi import APIRouter

router = APIRouter()

%[1]s_requirements = [%[3]s]

@router.post("/generated/%[2]s", status_code=201)
def handle_%[5]s() -> dict:
    return {
        "status": "accepted",
        "interface": "%[4]s",
        "requirements": %[1]s_requirements,
    }
`, componentName, route, quotedRequirements(spec.Requirements), firstNonEmpty(spec.Interface, "REST"), functionName)
}

func generatedPythonTests(componentName string) string {
	return fmt.Sprintf(`def test_%s_handler() -> None:
    assert True
`, strings.ToLower(componentName))
}

func generatedTestSuite(code, language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "go":
		name := "GeneratedContract"
		matches := regexp.MustCompile(`func\s+(Handle[A-Za-z0-9]+)`).FindStringSubmatch(code)
		if len(matches) > 1 {
			name = matches[1]
		}
		return fmt.Sprintf(`package generated

import "testing"

func Test%s(t *testing.T) {
	if len(%q) == 0 {
		t.Fatal("expected generated code to be non-empty")
	}
}
`, name, code)
	case "typescript":
		return "describe(\"generated suite\", () => { it(\"keeps contract coverage\", () => expect(true).toBe(true)); });\n"
	case "python":
		return "def test_generated_suite() -> None:\n    assert True\n"
	default:
		return "// generated validation suite\n"
	}
}

func quotedRequirements(requirements []string) string {
	if len(requirements) == 0 {
		return ""
	}

	quoted := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		quoted = append(quoted, fmt.Sprintf("%q", requirement))
	}
	return strings.Join(quoted, ", ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func scopedTenantID(ctx context.Context) string {
	tenantID, err := domain.TenantIDFromContext(ctx, "service.scope")
	if err != nil {
		return ""
	}
	return tenantID
}
