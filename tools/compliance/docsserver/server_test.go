package docsserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stratogroup/neuroforge-sdx/tools/compliance/docsserver"
	"github.com/stretchr/testify/require"
)

func TestDocsServerRootRedirectsToSwagger(t *testing.T) {
	handler := docsserver.NewHandler()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
	require.Equal(t, "/swagger", recorder.Header().Get("Location"))
}

func TestDocsServerSwaggerRoute(t *testing.T) {
	assertDocsRoute(t, "/swagger", "SwaggerUIBundle")
	assertDocsRoute(t, "/swagger", "openapi.json")
}

func TestDocsServerRedocRoute(t *testing.T) {
	assertDocsRoute(t, "/docs", "redoc")
	assertDocsRoute(t, "/docs", "openapi.json")
}

func TestDocsServerSpecAssets(t *testing.T) {
	assertDocsRoute(t, "/openapi.json", `"openapi": "3.1.0"`)
	assertDocsRoute(t, "/openapi.yaml", "openapi: 3.1.0")
}

func assertDocsRoute(t *testing.T, path, body string) {
	t.Helper()

	handler := docsserver.NewHandler()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), body)
}
