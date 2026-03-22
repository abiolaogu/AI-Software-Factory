package suite_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/gin-gonic/gin"
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

func newComplianceEngine(t *testing.T) *gin.Engine {
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

	logger := observability.NewLogger("compliance-test", "test", "ERROR")
	return httpadapter.NewRouter(cfg, application, logger)
}

func newTestArtifactService(t *testing.T, serviceName string) *artifacts.SignedArtifactService {
	t.Helper()

	service, err := artifacts.NewSignedArtifactService(serviceName, "https://artifacts.example.com", "signing-key", 15*time.Minute)
	require.NoError(t, err)
	return service
}

func newSpecRouter(t *testing.T) (*openapi3.T, routers.Router) {
	t.Helper()

	document, err := spec.Load()
	require.NoError(t, err)

	specRouter, err := legacy.NewRouter(document)
	require.NoError(t, err)
	return document, specRouter
}

func performRequest(t *testing.T, engine http.Handler, method, path string, payload any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		require.NoError(t, err)
	}

	request := httptest.NewRequest(method, "http://localhost:8080"+path, bytes.NewReader(body))
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder
}

func validateExchangeWithSpec(
	t *testing.T,
	engine http.Handler,
	specRouter routers.Router,
	method, path string,
	payload any,
	headers map[string]string,
	expectedStatus int,
) map[string]any {
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
		Body:                   http.NoBody,
	}
	if recorder.Body.Len() > 0 {
		responseValidationInput.Body = ioNopCloser(recorder.Body.Bytes())
	}
	require.NoError(t, openapi3filter.ValidateResponse(context.Background(), responseValidationInput))

	if recorder.Body.Len() == 0 {
		return map[string]any{}
	}

	var response map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assertDocumentedFields(t, route.Operation.Responses.Status(expectedStatus).Value.Content.Get("application/json").Schema, response)
	return response
}

func ioNopCloser(body []byte) *nopCloser {
	return &nopCloser{reader: bytes.NewReader(body)}
}

type nopCloser struct {
	reader *bytes.Reader
}

func (n *nopCloser) Read(p []byte) (int, error) {
	return n.reader.Read(p)
}

func (n *nopCloser) Close() error {
	return nil
}

func assertDocumentedFields(t *testing.T, schemaRef *openapi3.SchemaRef, value any) {
	t.Helper()
	if schemaRef == nil || schemaRef.Value == nil || value == nil {
		return
	}

	schema := schemaRef.Value
	switch typed := value.(type) {
	case map[string]any:
		if len(schema.Properties) == 0 && schema.AdditionalProperties.Schema == nil && schema.AdditionalProperties.Has == nil {
			return
		}

		if schema.AdditionalProperties.Schema != nil && len(schema.Properties) == 0 {
			for _, nested := range typed {
				assertDocumentedFields(t, schema.AdditionalProperties.Schema, nested)
			}
			return
		}

		allowed := make([]string, 0, len(schema.Properties))
		for name := range schema.Properties {
			allowed = append(allowed, name)
		}
		slices.Sort(allowed)

		for key, nested := range typed {
			propertySchema, ok := schema.Properties[key]
			if !ok {
				if schema.AdditionalProperties.Has != nil && *schema.AdditionalProperties.Has {
					continue
				}
				if schema.AdditionalProperties.Schema != nil {
					assertDocumentedFields(t, schema.AdditionalProperties.Schema, nested)
					continue
				}
				t.Fatalf("undocumented field %q, allowed fields: %v", key, allowed)
			}
			assertDocumentedFields(t, propertySchema, nested)
		}
	case []any:
		if schema.Items == nil {
			return
		}
		for _, item := range typed {
			assertDocumentedFields(t, schema.Items, item)
		}
	}
}

func complianceHeaders(tenantID string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + tenantToken(tenantID),
		"X-Tenant-ID":   tenantID,
	}
}

func tenantToken(tenantID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"tenant_id":"` + tenantID + `"}`))
	return header + "." + payload + ".signature"
}

func normalizeGinPath(path string) string {
	matcher := regexp.MustCompile(`:([A-Za-z0-9_]+)`)
	return matcher.ReplaceAllString(path, `{$1}`)
}
