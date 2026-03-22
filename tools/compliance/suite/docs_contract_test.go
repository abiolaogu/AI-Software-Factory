package suite_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/stratogroup/neuroforge-sdx/tools/compliance/docsserver"
	"github.com/stretchr/testify/require"
)

func TestComplianceDocsRoutesServe(t *testing.T) {
	handler := docsserver.NewHandler()

	for _, endpoint := range []string{"/swagger", "/docs", "/openapi.json", "/openapi.yaml"} {
		request := httptest.NewRequest(http.MethodGet, endpoint, nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code, endpoint)
		require.NotEmpty(t, recorder.Body.String(), endpoint)
	}
}

func TestComplianceVersionedRoutesAreDocumented(t *testing.T) {
	engine := newComplianceEngine(t)
	document, _ := newSpecRouter(t)

	documented := make(map[string]struct{})
	for path, item := range document.Paths.Map() {
		for method := range item.Operations() {
			documented[strings.ToUpper(method)+" "+path] = struct{}{}
		}
	}

	ignored := map[string]struct{}{
		"GET /v1/health/live":  {},
		"GET /v1/health/ready": {},
		"GET /v1/livez":        {},
		"GET /v1/readyz":       {},
		"GET /v1/healthz":      {},
	}

	undocumented := make([]string, 0)
	for _, route := range engine.Routes() {
		if !strings.HasPrefix(route.Path, "/v1/") {
			continue
		}
		key := route.Method + " " + normalizeGinPath(strings.TrimPrefix(route.Path, "/v1"))
		if _, skip := ignored[route.Method+" "+route.Path]; skip {
			continue
		}
		if _, ok := documented[key]; !ok {
			undocumented = append(undocumented, route.Method+" "+route.Path)
		}
	}

	slices.Sort(undocumented)
	require.Empty(t, undocumented)
}
