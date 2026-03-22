package suite_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComplianceContractSmokeForDocumentedEndpoints(t *testing.T) {
	engine := newComplianceEngine(t)
	_, specRouter := newSpecRouter(t)

	headers := complianceHeaders("tenant-a")

	validateExchangeWithSpec(t, engine, specRouter, http.MethodPost, "/v1/auth/login", map[string]any{
		"email":    "architect@neuroforge.io",
		"password": "correct-horse-battery-staple",
	}, headers, http.StatusOK)

	validateExchangeWithSpec(t, engine, specRouter, http.MethodPost, "/v1/auth/refresh", map[string]any{
		"refresh_token": "refresh-1",
	}, headers, http.StatusOK)

	workflow := validateExchangeWithSpec(t, engine, specRouter, http.MethodPost, "/v1/workflows", map[string]any{
		"prompt":           "Build a production-grade autonomous workflow orchestration service with audit-ready controls and strict contract testing.",
		"domain":           "api_service",
		"git_strategy":     "trunk-based",
		"target_languages": []string{"go"},
		"regions":          []string{"lagos"},
	}, mergeHeaders(headers, map[string]string{"Idempotency-Key": "idem-1"}), http.StatusCreated)

	workflowID := workflow["workflow_id"].(string)
	validateExchangeWithSpec(t, engine, specRouter, http.MethodGet, "/v1/workflows?page=1&size=20", nil, headers, http.StatusOK)
	validateExchangeWithSpec(t, engine, specRouter, http.MethodGet, "/v1/workflows/"+workflowID, nil, headers, http.StatusOK)
	validateExchangeWithSpec(t, engine, specRouter, http.MethodGet, "/v1/workflows/"+workflowID+"/artifacts", nil, headers, http.StatusOK)

	prd := validateExchangeWithSpec(t, engine, specRouter, http.MethodPost, "/v1/prd/generate", map[string]any{
		"prompt": "Generate a PRD for a secure AI-native software delivery platform with tenant-aware workflow orchestration and strong compliance gates.",
		"domain": "saas",
	}, mergeHeaders(headers, map[string]string{"Idempotency-Key": "idem-prd"}), http.StatusCreated)

	prdID := prd["prd_id"].(string)
	validateExchangeWithSpec(t, engine, specRouter, http.MethodGet, "/v1/prd/"+prdID, nil, headers, http.StatusOK)

	design := validateExchangeWithSpec(t, engine, specRouter, http.MethodPost, "/v1/design/wireframes", map[string]any{
		"prd_id":       prdID,
		"style":        "professional",
		"color_scheme": "dark",
	}, mergeHeaders(headers, map[string]string{"Idempotency-Key": "idem-design"}), http.StatusCreated)

	designID := design["design_id"].(string)
	validateExchangeWithSpec(t, engine, specRouter, http.MethodPost, "/v1/design/tokens", map[string]any{
		"design_id": designID,
	}, mergeHeaders(headers, map[string]string{"Idempotency-Key": "idem-tokens"}), http.StatusOK)

	validateExchangeWithSpec(t, engine, specRouter, http.MethodPost, "/v1/code/generate", map[string]any{
		"specification": map[string]any{"description": "Generate scaffold"},
		"language":      "go",
		"framework":     "gin",
	}, mergeHeaders(headers, map[string]string{"Idempotency-Key": "idem-code"}), http.StatusCreated)

	validateExchangeWithSpec(t, engine, specRouter, http.MethodPost, "/v1/code/test", map[string]any{
		"code":     "package main\nfunc main() {}\n",
		"language": "go",
	}, mergeHeaders(headers, map[string]string{"Idempotency-Key": "idem-tests"}), http.StatusOK)

	scan := validateExchangeWithSpec(t, engine, specRouter, http.MethodPost, "/v1/security/scan", map[string]any{
		"target": "github.com/stratogroup/neuroforge-sdx",
	}, mergeHeaders(headers, map[string]string{"Idempotency-Key": "idem-scan"}), http.StatusAccepted)
	validateExchangeWithSpec(t, engine, specRouter, http.MethodGet, "/v1/security/findings/"+scan["scan_id"].(string), nil, headers, http.StatusOK)

	perf := validateExchangeWithSpec(t, engine, specRouter, http.MethodPost, "/v1/performance/test", map[string]any{
		"endpoint": "https://api.neuroforge.local/v1/workflows",
	}, mergeHeaders(headers, map[string]string{"Idempotency-Key": "idem-perf"}), http.StatusAccepted)
	validateExchangeWithSpec(t, engine, specRouter, http.MethodGet, "/v1/performance/results/"+perf["test_id"].(string), nil, headers, http.StatusOK)
}

func TestComplianceAuthAndTenantHeadersAreEnforced(t *testing.T) {
	engine := newComplianceEngine(t)

	noAuth := performRequest(t, engine, http.MethodPost, "/v1/workflows", map[string]any{
		"prompt": "Build a production-grade autonomous workflow orchestration service with audit-ready controls and strict contract testing.",
	}, map[string]string{})
	require.Equal(t, http.StatusUnauthorized, noAuth.Code)

	noTenant := performRequest(t, engine, http.MethodPost, "/v1/workflows", map[string]any{
		"prompt": "Build a production-grade autonomous workflow orchestration service with audit-ready controls and strict contract testing.",
	}, map[string]string{"Authorization": "Bearer opaque-token"})
	require.Equal(t, http.StatusBadRequest, noTenant.Code)
	validateErrorSchema(t, noTenant.Body.Bytes())
}

func TestComplianceTenantClaimDriftIsRejected(t *testing.T) {
	engine := newComplianceEngine(t)

	response := performRequest(t, engine, http.MethodPost, "/v1/workflows", map[string]any{
		"prompt": "Build a production-grade autonomous workflow orchestration service with audit-ready controls and strict contract testing.",
	}, map[string]string{
		"Authorization": "Bearer " + tenantToken("tenant-a"),
		"X-Tenant-ID":   "tenant-b",
	})

	require.Equal(t, http.StatusUnauthorized, response.Code)
	validateErrorSchema(t, response.Body.Bytes())
}

func TestCompliancePaginationRateLimitAndIdempotency(t *testing.T) {
	engine := newComplianceEngine(t)
	headers := complianceHeaders("tenant-a")

	first := performRequest(t, engine, http.MethodPost, "/v1/workflows", map[string]any{
		"prompt": "Build a production-grade autonomous workflow orchestration service with audit-ready controls and strict contract testing.",
	}, mergeHeaders(headers, map[string]string{"Idempotency-Key": "idem-1"}))
	require.Equal(t, http.StatusCreated, first.Code)

	second := performRequest(t, engine, http.MethodPost, "/v1/workflows", map[string]any{
		"prompt": "Build a production-grade autonomous workflow orchestration service with audit-ready controls and strict contract testing.",
	}, mergeHeaders(headers, map[string]string{"Idempotency-Key": "idem-1"}))
	require.Equal(t, http.StatusCreated, second.Code)
	require.JSONEq(t, first.Body.String(), second.Body.String())

	list := performRequest(t, engine, http.MethodGet, "/v1/workflows?page=1&size=20", nil, headers)
	require.Equal(t, http.StatusOK, list.Code)
	require.NotEmpty(t, list.Header().Get("X-RateLimit-Limit"))
	require.NotEmpty(t, list.Header().Get("X-RateLimit-Remaining"))
	require.NotEmpty(t, list.Header().Get("X-RateLimit-Reset"))

	var payload struct {
		Pagination map[string]any `json:"pagination"`
	}
	require.NoError(t, json.Unmarshal(list.Body.Bytes(), &payload))
	require.Equal(t, float64(1), payload.Pagination["page"])
	require.Equal(t, float64(20), payload.Pagination["size"])
}

func TestComplianceNegativeErrorModelsMatchSpec(t *testing.T) {
	engine := newComplianceEngine(t)
	headers := complianceHeaders("tenant-a")

	notFound := performRequest(t, engine, http.MethodGet, "/v1/workflows/missing", nil, headers)
	require.Equal(t, http.StatusNotFound, notFound.Code)
	validateErrorSchema(t, notFound.Body.Bytes())

	validation := performRequest(t, engine, http.MethodPost, "/v1/prd/generate", map[string]any{
		"prompt": "too short",
	}, headers)
	require.Equal(t, http.StatusBadRequest, validation.Code)
	validateErrorSchema(t, validation.Body.Bytes())
}

func validateErrorSchema(t *testing.T, body []byte) {
	t.Helper()

	document, _ := newSpecRouter(t)
	schemaRef := document.Components.Schemas["Error"]
	require.NotNil(t, schemaRef)
	require.NotNil(t, schemaRef.Value)

	var payload any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.NoError(t, schemaRef.Value.VisitJSON(payload))
	assertDocumentedFields(t, schemaRef, payload)
}

func mergeHeaders(base, extra map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}
