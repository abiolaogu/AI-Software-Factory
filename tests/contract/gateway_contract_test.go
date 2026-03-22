package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// specPath resolves the OpenAPI spec relative to the repo root.
func specPath(t *testing.T) string {
	t.Helper()

	// Try environment variable first (CI).
	if p := os.Getenv("OPENAPI_SPEC"); p != "" {
		return p
	}

	// Walk up from test file to repo root.
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	return filepath.Join(repoRoot, "api", "openapi", "sdx-gateway-v1.swagger.json")
}

// loadSpec loads the OpenAPI spec as a generic map.
func loadSpec(t *testing.T) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(specPath(t))
	if err != nil {
		t.Fatalf("failed to read OpenAPI spec: %v", err)
	}
	var spec map[string]interface{}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("failed to parse OpenAPI spec: %v", err)
	}
	return spec
}

// TestGatewayEndpointsExist verifies that the generated OpenAPI spec
// contains all endpoints documented in docs/api/rest-mapping.md.
func TestGatewayEndpointsExist(t *testing.T) {
	spec := loadSpec(t)

	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("OpenAPI spec has no 'paths' object")
	}

	if len(paths) == 0 {
		t.Fatal("OpenAPI spec 'paths' is empty — google.api.http annotations are likely missing from proto files")
	}

	// Minimum expected endpoints from rest-mapping.md.
	// grpc-gateway generates paths like /api/v1/... based on the http annotations.
	expectedPrefixes := []string{
		"/api/v1/work-items",
		"/api/v1/builds",
		"/api/v1/artifacts",
		"/api/v1/tenants",
		"/api/v1/webhooks",
		"/api/v1/health",
		"/api/v1/audit-log",
		"/api/v1/compliance",
		"/api/v1/deployments",
		"/api/v1/agents",
		"/api/v1/llm",
		"/api/v1/quality",
	}

	for _, prefix := range expectedPrefixes {
		found := false
		for path := range paths {
			if strings.HasPrefix(path, prefix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected path prefix %q not found in OpenAPI spec", prefix)
		}
	}

	t.Logf("OpenAPI spec contains %d path entries", len(paths))
}

// TestGatewayWorkItemCRUD validates the WorkItem CRUD operations are present.
func TestGatewayWorkItemCRUD(t *testing.T) {
	spec := loadSpec(t)
	paths, _ := spec["paths"].(map[string]interface{})
	if paths == nil {
		t.Fatal("no paths in spec")
	}

	checks := []struct {
		path   string
		method string
	}{
		{"/api/v1/work-items", "post"},
		{"/api/v1/work-items", "get"},
		{"/api/v1/work-items/{work_item_id}", "get"},
		{"/api/v1/work-items/{work_item_id}", "delete"},
		{"/api/v1/work-items/{work_item_id}/approve", "post"},
	}

	for _, check := range checks {
		pathObj, ok := paths[check.path].(map[string]interface{})
		if !ok {
			t.Errorf("path %q not found in spec", check.path)
			continue
		}
		if _, ok := pathObj[check.method]; !ok {
			t.Errorf("method %s not found for path %q", strings.ToUpper(check.method), check.path)
		}
	}
}

// TestGatewayHealthEndpoint verifies the health check endpoint exists.
func TestGatewayHealthEndpoint(t *testing.T) {
	spec := loadSpec(t)
	paths, _ := spec["paths"].(map[string]interface{})
	if paths == nil {
		t.Fatal("no paths in spec")
	}

	healthPath, ok := paths["/api/v1/health"].(map[string]interface{})
	if !ok {
		t.Fatal("health endpoint /api/v1/health not found in spec")
	}
	if _, ok := healthPath["get"]; !ok {
		t.Error("GET method not found for /api/v1/health")
	}
}
