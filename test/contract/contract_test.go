package contract_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/artifacts"
	cfgadapter "github.com/stratogroup/neuroforge-sdx/internal/adapters/config"
	httpadapter "github.com/stratogroup/neuroforge-sdx/internal/adapters/http"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/messaging"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/observability"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/spec"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/storage"
	"github.com/stratogroup/neuroforge-sdx/internal/usecases"
	"github.com/stretchr/testify/require"
)

func TestOpenAPISpecIsValid(t *testing.T) {
	document, err := spec.Load()
	require.NoError(t, err)
	require.Equal(t, "3.1.0", document.OpenAPI)
}

func TestDocumentationEndpoints(t *testing.T) {
	engine := newTestEngine(t)

	for _, endpoint := range []string{"/swagger", "/docs", "/openapi.json", "/openapi.yaml"} {
		request := httptest.NewRequest(http.MethodGet, endpoint, nil)
		recorder := httptest.NewRecorder()

		engine.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code, endpoint)
		require.NotEmpty(t, recorder.Body.String(), endpoint)
	}
}

func TestContractRoutesMatchSpec(t *testing.T) {
	engine := newTestEngine(t)
	document, err := spec.Load()
	require.NoError(t, err)

	specRouter, err := legacy.NewRouter(document)
	require.NoError(t, err)

	authHeader := map[string]string{
		"Authorization": "Bearer contract-token",
		"X-Tenant-ID":   "contract-tenant",
	}

	loginBody, loginResponse := validateExchange(t, engine, specRouter, http.MethodPost, "/v1/auth/login", map[string]string{
		"email":    "contract@example.com",
		"password": "correct-horse-battery-staple",
	}, authHeader, http.StatusOK)
	require.NotEmpty(t, loginBody)
	require.NotEmpty(t, loginResponse["access_token"])

	validateExchange(t, engine, specRouter, http.MethodPost, "/v1/auth/refresh", map[string]string{
		"refresh_token": "contract-refresh-token",
	}, authHeader, http.StatusOK)

	workflowBody, workflowResponse := validateExchange(t, engine, specRouter, http.MethodPost, "/v1/workflows", map[string]any{
		"prompt":           strings.Repeat("Build a production-ready autonomous workflow service. ", 2),
		"domain":           "api_service",
		"git_strategy":     "trunk-based",
		"target_languages": []string{"go", "typescript"},
		"regions":          []string{"lagos", "london"},
		"metadata": map[string]any{
			"requested_by": "contract-test",
		},
	}, authHeader, http.StatusCreated)
	require.NotEmpty(t, workflowBody)
	workflowID, ok := workflowResponse["workflow_id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, workflowID)

	validateExchange(t, engine, specRouter, http.MethodGet, "/v1/workflows?page=1&size=20", nil, authHeader, http.StatusOK)
	validateExchange(t, engine, specRouter, http.MethodGet, "/v1/workflows/"+workflowID, nil, authHeader, http.StatusOK)
	validateExchange(t, engine, specRouter, http.MethodGet, "/v1/workflows/"+workflowID+"/artifacts", nil, authHeader, http.StatusOK)

	_, prdResponse := validateExchange(t, engine, specRouter, http.MethodPost, "/v1/prd/generate", map[string]any{
		"prompt": strings.Repeat("Generate a PRD for a secure AI-native software delivery platform. ", 2),
		"domain": "saas",
	}, authHeader, http.StatusCreated)
	prdID, ok := prdResponse["prd_id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, prdID)

	validateExchange(t, engine, specRouter, http.MethodGet, "/v1/prd/"+prdID, nil, authHeader, http.StatusOK)

	_, designResponse := validateExchange(t, engine, specRouter, http.MethodPost, "/v1/design/wireframes", map[string]any{
		"prd_id":       prdID,
		"style":        "professional",
		"color_scheme": "dark",
	}, authHeader, http.StatusCreated)
	designID, ok := designResponse["design_id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, designID)

	validateExchange(t, engine, specRouter, http.MethodPost, "/v1/design/tokens", map[string]any{
		"design_id": designID,
	}, authHeader, http.StatusOK)

	validateExchange(t, engine, specRouter, http.MethodPost, "/v1/code/generate", map[string]any{
		"specification": map[string]any{
			"description":  "Generate a code scaffold",
			"interface":    "REST",
			"requirements": []string{"telemetry", "contract tests"},
		},
		"language":  "go",
		"framework": "gin",
	}, authHeader, http.StatusCreated)

	validateExchange(t, engine, specRouter, http.MethodPost, "/v1/code/test", map[string]any{
		"code":     "package main\nfunc main() {}\n",
		"language": "go",
	}, authHeader, http.StatusOK)

	_, scanResponse := validateExchange(t, engine, specRouter, http.MethodPost, "/v1/security/scan", map[string]any{
		"target":     "github.com/stratogroup/neuroforge-sdx",
		"scan_types": []string{"sast", "sca"},
	}, authHeader, http.StatusAccepted)
	scanID, ok := scanResponse["scan_id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, scanID)

	validateExchange(t, engine, specRouter, http.MethodGet, "/v1/security/findings/"+scanID, nil, authHeader, http.StatusOK)

	_, performanceResponse := validateExchange(t, engine, specRouter, http.MethodPost, "/v1/performance/test", map[string]any{
		"endpoint":      "https://api.neuroforge.local/v1/workflows",
		"duration":      "5m",
		"virtual_users": 100,
		"regions":       []string{"lagos"},
	}, authHeader, http.StatusAccepted)
	testID, ok := performanceResponse["test_id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, testID)

	validateExchange(t, engine, specRouter, http.MethodGet, "/v1/performance/results/"+testID, nil, authHeader, http.StatusOK)
	validateExchange(t, engine, specRouter, http.MethodGet, "/v1/health/live", nil, authHeader, http.StatusOK)
	validateExchange(t, engine, specRouter, http.MethodGet, "/v1/health/ready", nil, authHeader, http.StatusOK)
	validateExchange(t, engine, specRouter, http.MethodDelete, "/v1/workflows/"+workflowID, nil, authHeader, http.StatusOK)
}

func newTestEngine(t *testing.T) http.Handler {
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
		cfg.ReadinessChecks(),
		string(cfgadapter.ModeAPI),
	)

	logger := observability.NewLogger("contract-test", "test", "ERROR")
	return httpadapter.NewRouter(cfg, application, logger)
}

func newTestArtifactService(t *testing.T, serviceName string) *artifacts.SignedArtifactService {
	t.Helper()

	service, err := artifacts.NewSignedArtifactService(serviceName, "https://artifacts.example.com", "signing-key", 15*time.Minute)
	require.NoError(t, err)
	return service
}

func validateExchange(
	t *testing.T,
	engine http.Handler,
	specRouter routers.Router,
	method string,
	path string,
	payload any,
	headers map[string]string,
	expectedStatus int,
) ([]byte, map[string]any) {
	t.Helper()

	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		require.NoError(t, err)
	}

	requestURL := "http://localhost:8000" + path
	validationRequest := httptest.NewRequest(method, requestURL, bytes.NewReader(body))
	serverRequest := httptest.NewRequest(method, requestURL, bytes.NewReader(body))

	if payload != nil {
		validationRequest.Header.Set("Content-Type", "application/json")
		serverRequest.Header.Set("Content-Type", "application/json")
	}

	for key, value := range headers {
		validationRequest.Header.Set(key, value)
		serverRequest.Header.Set(key, value)
	}

	route, pathParams, err := specRouter.FindRoute(validationRequest)
	require.NoError(t, err)

	requestValidationInput := &openapi3filter.RequestValidationInput{
		Request:    validationRequest,
		Route:      route,
		PathParams: pathParams,
		Options: &openapi3filter.Options{
			AuthenticationFunc: func(_ context.Context, input *openapi3filter.AuthenticationInput) error {
				if input.RequestValidationInput.Request.Header.Get("Authorization") == "" {
					return errors.New("missing Authorization header")
				}
				return nil
			},
		},
	}
	require.NoError(t, openapi3filter.ValidateRequest(context.Background(), requestValidationInput))

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, serverRequest)
	require.Equal(t, expectedStatus, recorder.Code, path)

	responseValidationInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: requestValidationInput,
		Status:                 recorder.Code,
		Header:                 recorder.Header(),
		Body:                   io.NopCloser(bytes.NewReader(recorder.Body.Bytes())),
	}
	require.NoError(t, openapi3filter.ValidateResponse(context.Background(), responseValidationInput))

	if recorder.Body.Len() == 0 {
		return nil, map[string]any{}
	}

	var response map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	return recorder.Body.Bytes(), response
}
