package http_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stratogroup/neuroforge-sdx/internal/adapters/artifacts"
	cfgadapter "github.com/stratogroup/neuroforge-sdx/internal/adapters/config"
	httpadapter "github.com/stratogroup/neuroforge-sdx/internal/adapters/http"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/messaging"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/observability"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/storage"
	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stratogroup/neuroforge-sdx/internal/usecases"
	"github.com/stretchr/testify/require"
)

func TestProtectedRoutesRequireTenantHeader(t *testing.T) {
	engine := newTestRouter(t)

	request := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBufferString(`{"prompt":"Build a production-grade multi-tenant workflow orchestration service with contract-first delivery."}`))
	request.Header.Set("Authorization", "Bearer test-token")
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "TENANT_REQUIRED")
}

func TestWorkflowCreateIsIdempotentPerTenant(t *testing.T) {
	engine := newTestRouter(t)
	requestBody := []byte(`{"prompt":"Build a production-grade multi-tenant workflow orchestration service with contract-first delivery."}`)

	first := performJSONRequest(t, engine, http.MethodPost, "/v1/workflows", requestBody, map[string]string{
		"Authorization":   "Bearer test-token",
		"Content-Type":    "application/json",
		"X-Tenant-ID":     "tenant-a",
		"Idempotency-Key": "idem-1",
	})
	require.Equal(t, http.StatusCreated, first.Code)

	second := performJSONRequest(t, engine, http.MethodPost, "/v1/workflows", requestBody, map[string]string{
		"Authorization":   "Bearer test-token",
		"Content-Type":    "application/json",
		"X-Tenant-ID":     "tenant-a",
		"Idempotency-Key": "idem-1",
	})
	require.Equal(t, http.StatusCreated, second.Code)
	require.JSONEq(t, first.Body.String(), second.Body.String())

	third := performJSONRequest(t, engine, http.MethodPost, "/v1/workflows", requestBody, map[string]string{
		"Authorization":   "Bearer test-token",
		"Content-Type":    "application/json",
		"X-Tenant-ID":     "tenant-b",
		"Idempotency-Key": "idem-1",
	})
	require.Equal(t, http.StatusCreated, third.Code)
	require.NotEqual(t, first.Body.String(), third.Body.String())
}

func TestHealthAliasesAreServed(t *testing.T) {
	engine := newTestRouter(t)

	for _, endpoint := range []string{"/healthz", "/readyz", "/livez"} {
		request := httptest.NewRequest(http.MethodGet, endpoint, nil)
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusOK, recorder.Code, endpoint)
	}
}

func TestProtectedRoutesAllowAnonymousWhenConfigured(t *testing.T) {
	cfg := cfgadapter.Load(cfgadapter.ModeAPI)
	cfg.AllowAnonymousAuth = true

	store := storage.NewMemoryStore()
	application := usecases.NewService(
		store,
		store,
		store,
		messaging.NoopPublisher{},
		newTestArtifactService(t, cfg.ServiceName),
		cfg.ServiceName,
		map[string]string{"database": "configured"},
		"test",
	)

	logger := observability.NewLogger("router-test", "test", "ERROR")
	engine := httpadapter.NewRouter(cfg, application, logger)

	request := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBufferString(`{"prompt":"Build a production-grade multi-tenant workflow orchestration service with contract-first delivery."}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Tenant-ID", "tenant-a")

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusCreated, recorder.Code)
}

func TestRouterEndpointSmoke(t *testing.T) {
	engine := newTestRouter(t)
	headers := map[string]string{
		"Authorization": "Bearer test-token",
		"X-Tenant-ID":   "tenant-a",
	}

	login := performJSONRequest(t, engine, http.MethodPost, "/v1/auth/login", []byte(`{"email":"architect@neuroforge.io","password":"secret"}`), map[string]string{
		"Content-Type": "application/json",
	})
	require.Equal(t, http.StatusOK, login.Code)

	refresh := performJSONRequest(t, engine, http.MethodPost, "/v1/auth/refresh", []byte(`{"refresh_token":"refresh-1"}`), map[string]string{
		"Content-Type": "application/json",
	})
	require.Equal(t, http.StatusOK, refresh.Code)

	workflow := performJSONRequest(t, engine, http.MethodPost, "/v1/workflows", []byte(`{"prompt":"Build a production-grade multi-tenant workflow orchestration service with contract-first delivery."}`), mergeHeaders(headers, map[string]string{
		"Content-Type":    "application/json",
		"Idempotency-Key": "idem-1",
	}))
	require.Equal(t, http.StatusCreated, workflow.Code)

	var workflowPayload map[string]any
	require.NoError(t, json.Unmarshal(workflow.Body.Bytes(), &workflowPayload))
	workflowID := workflowPayload["workflow_id"].(string)

	require.Equal(t, http.StatusOK, performJSONRequest(t, engine, http.MethodGet, "/v1/workflows?page=1&size=20", nil, headers).Code)
	require.Equal(t, http.StatusOK, performJSONRequest(t, engine, http.MethodGet, "/v1/workflows/"+workflowID, nil, headers).Code)
	require.Equal(t, http.StatusOK, performJSONRequest(t, engine, http.MethodGet, "/v1/workflows/"+workflowID+"/artifacts", nil, headers).Code)

	prd := performJSONRequest(t, engine, http.MethodPost, "/v1/prd/generate", []byte(`{"prompt":"Generate a PRD for a secure AI-native software delivery platform with tenant-aware workflow orchestration.","domain":"saas"}`), mergeHeaders(headers, map[string]string{
		"Content-Type": "application/json",
	}))
	require.Equal(t, http.StatusCreated, prd.Code)
	var prdPayload map[string]any
	require.NoError(t, json.Unmarshal(prd.Body.Bytes(), &prdPayload))
	prdID := prdPayload["prd_id"].(string)
	require.Equal(t, http.StatusOK, performJSONRequest(t, engine, http.MethodGet, "/v1/prd/"+prdID, nil, headers).Code)

	design := performJSONRequest(t, engine, http.MethodPost, "/v1/design/wireframes", []byte(`{"prd_id":"`+prdID+`","style":"professional","color_scheme":"dark"}`), mergeHeaders(headers, map[string]string{
		"Content-Type": "application/json",
	}))
	require.Equal(t, http.StatusCreated, design.Code)
	var designPayload map[string]any
	require.NoError(t, json.Unmarshal(design.Body.Bytes(), &designPayload))
	designID := designPayload["design_id"].(string)
	require.Equal(t, http.StatusOK, performJSONRequest(t, engine, http.MethodPost, "/v1/design/tokens", []byte(`{"design_id":"`+designID+`"}`), mergeHeaders(headers, map[string]string{
		"Content-Type": "application/json",
	})).Code)

	require.Equal(t, http.StatusCreated, performJSONRequest(t, engine, http.MethodPost, "/v1/code/generate", []byte(`{"specification":{"description":"Generate code"},"language":"go","framework":"gin"}`), mergeHeaders(headers, map[string]string{
		"Content-Type": "application/json",
	})).Code)
	require.Equal(t, http.StatusOK, performJSONRequest(t, engine, http.MethodPost, "/v1/code/test", []byte(`{"code":"package main","language":"go"}`), mergeHeaders(headers, map[string]string{
		"Content-Type": "application/json",
	})).Code)

	scan := performJSONRequest(t, engine, http.MethodPost, "/v1/security/scan", []byte(`{"target":"github.com/stratogroup/neuroforge-sdx"}`), mergeHeaders(headers, map[string]string{
		"Content-Type": "application/json",
	}))
	require.Equal(t, http.StatusAccepted, scan.Code)
	var scanPayload map[string]any
	require.NoError(t, json.Unmarshal(scan.Body.Bytes(), &scanPayload))
	scanID := scanPayload["scan_id"].(string)
	require.Equal(t, http.StatusOK, performJSONRequest(t, engine, http.MethodGet, "/v1/security/findings/"+scanID, nil, headers).Code)

	perf := performJSONRequest(t, engine, http.MethodPost, "/v1/performance/test", []byte(`{"endpoint":"https://api.neuroforge.local/v1/workflows"}`), mergeHeaders(headers, map[string]string{
		"Content-Type": "application/json",
	}))
	require.Equal(t, http.StatusAccepted, perf.Code)
	var perfPayload map[string]any
	require.NoError(t, json.Unmarshal(perf.Body.Bytes(), &perfPayload))
	testID := perfPayload["test_id"].(string)
	require.Equal(t, http.StatusOK, performJSONRequest(t, engine, http.MethodGet, "/v1/performance/results/"+testID, nil, headers).Code)

	require.Equal(t, http.StatusOK, performJSONRequest(t, engine, http.MethodGet, "/health/live", nil, nil).Code)
	require.Equal(t, http.StatusOK, performJSONRequest(t, engine, http.MethodGet, "/health/ready", nil, nil).Code)
	require.Equal(t, http.StatusTemporaryRedirect, performJSONRequest(t, engine, http.MethodGet, "/", nil, nil).Code)
	require.Equal(t, http.StatusOK, performJSONRequest(t, engine, http.MethodGet, "/swagger", nil, nil).Code)
	require.Equal(t, http.StatusOK, performJSONRequest(t, engine, http.MethodGet, "/docs", nil, nil).Code)
	require.Equal(t, http.StatusOK, performJSONRequest(t, engine, http.MethodGet, "/openapi.json", nil, nil).Code)
	require.Equal(t, http.StatusOK, performJSONRequest(t, engine, http.MethodGet, "/openapi.yaml", nil, nil).Code)

	cancel := performJSONRequest(t, engine, http.MethodDelete, "/v1/workflows/"+workflowID, nil, headers)
	require.Equal(t, http.StatusOK, cancel.Code)
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	cfg := cfgadapter.Load(cfgadapter.ModeAPI)
	cfg.AllowAnonymousAuth = false

	store := storage.NewMemoryStore()
	application := usecases.NewService(
		store,
		store,
		store,
		messaging.NoopPublisher{},
		newTestArtifactService(t, cfg.ServiceName),
		cfg.ServiceName,
		map[string]string{"database": "configured"},
		"test",
	)

	logger := observability.NewLogger("router-test", "test", "ERROR")
	return httpadapter.NewRouter(cfg, application, logger)
}

func performJSONRequest(t *testing.T, engine http.Handler, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code >= http.StatusBadRequest {
		payload, _ := io.ReadAll(recorder.Body)
		t.Logf("response body: %s", string(payload))
	}

	if (recorder.Code == http.StatusCreated || recorder.Code == http.StatusOK || recorder.Code == http.StatusAccepted) &&
		strings.Contains(recorder.Header().Get("Content-Type"), "application/json") {
		var payload map[string]any
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	}

	return recorder
}

func TestTenantHeaderConstantIsStable(t *testing.T) {
	require.Equal(t, "X-Tenant-ID", domain.HeaderTenantID)
}

func mergeHeaders(base map[string]string, extra map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func newTestArtifactService(t *testing.T, serviceName string) *artifacts.SignedArtifactService {
	t.Helper()

	service, err := artifacts.NewSignedArtifactService(serviceName, "https://artifacts.example.com", "signing-key", 15*time.Minute)
	require.NoError(t, err)
	return service
}
