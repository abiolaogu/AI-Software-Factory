package contract

import (
	"strings"
	"testing"
)

// TestOpenAPISpecHasInfo validates the OpenAPI spec info metadata.
func TestOpenAPISpecHasInfo(t *testing.T) {
	spec := loadSpec(t)

	info, ok := spec["info"].(map[string]interface{})
	if !ok {
		t.Fatal("OpenAPI spec has no 'info' object")
	}

	// Title must not be a proto file path.
	title, _ := info["title"].(string)
	if title == "" {
		t.Error("info.title is empty")
	}
	if strings.HasSuffix(title, ".proto") {
		t.Errorf("info.title should not be a proto path, got: %s", title)
	}

	// Version must be set.
	version, _ := info["version"].(string)
	if version == "" || version == "version not set" {
		t.Errorf("info.version is not set, got: %q", version)
	}
}

// TestOpenAPISpecHasSecurityDefinitions validates security definitions exist.
func TestOpenAPISpecHasSecurityDefinitions(t *testing.T) {
	spec := loadSpec(t)

	secDefs, ok := spec["securityDefinitions"].(map[string]interface{})
	if !ok {
		t.Fatal("OpenAPI spec has no 'securityDefinitions' — auth is not documented")
	}

	if len(secDefs) == 0 {
		t.Error("securityDefinitions is empty")
	}

	// Expect BearerAuth at minimum.
	if _, ok := secDefs["BearerAuth"]; !ok {
		t.Error("expected 'BearerAuth' security definition not found")
	}

	t.Logf("Found %d security definitions", len(secDefs))
}

// TestOpenAPIEndpointsHaveOperationId checks every endpoint has operationId.
func TestOpenAPIEndpointsHaveOperationId(t *testing.T) {
	spec := loadSpec(t)
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok || len(paths) == 0 {
		t.Skip("no paths to validate")
	}

	methods := []string{"get", "post", "put", "patch", "delete"}
	missing := 0

	for path, pathItem := range paths {
		pathObj, ok := pathItem.(map[string]interface{})
		if !ok {
			continue
		}
		for _, method := range methods {
			op, ok := pathObj[method].(map[string]interface{})
			if !ok {
				continue
			}
			opId, _ := op["operationId"].(string)
			if opId == "" {
				t.Errorf("%s %s: missing operationId", strings.ToUpper(method), path)
				missing++
			}
		}
	}

	if missing > 0 {
		t.Errorf("%d endpoints missing operationId", missing)
	}
}

// TestOpenAPIEndpointsHaveTags checks that endpoints are grouped by tags.
func TestOpenAPIEndpointsHaveTags(t *testing.T) {
	spec := loadSpec(t)
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok || len(paths) == 0 {
		t.Skip("no paths to validate")
	}

	methods := []string{"get", "post", "put", "patch", "delete"}
	missing := 0

	for path, pathItem := range paths {
		pathObj, ok := pathItem.(map[string]interface{})
		if !ok {
			continue
		}
		for _, method := range methods {
			op, ok := pathObj[method].(map[string]interface{})
			if !ok {
				continue
			}
			tags, _ := op["tags"].([]interface{})
			if len(tags) == 0 {
				t.Errorf("%s %s: missing tags", strings.ToUpper(method), path)
				missing++
			}
		}
	}

	if missing > 0 {
		t.Errorf("%d endpoints missing tags", missing)
	}
}

// TestOpenAPIEndpointsHaveResponses verifies each endpoint defines response schemas.
func TestOpenAPIEndpointsHaveResponses(t *testing.T) {
	spec := loadSpec(t)
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok || len(paths) == 0 {
		t.Skip("no paths to validate")
	}

	methods := []string{"get", "post", "put", "patch", "delete"}

	for path, pathItem := range paths {
		pathObj, ok := pathItem.(map[string]interface{})
		if !ok {
			continue
		}
		for _, method := range methods {
			op, ok := pathObj[method].(map[string]interface{})
			if !ok {
				continue
			}
			responses, ok := op["responses"].(map[string]interface{})
			if !ok || len(responses) == 0 {
				t.Errorf("%s %s: no responses defined", strings.ToUpper(method), path)
			}
		}
	}
}
