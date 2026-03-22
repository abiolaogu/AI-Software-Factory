package contract

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestProtoFilesExist verifies that all expected proto files are present.
func TestProtoFilesExist(t *testing.T) {
	// Resolve repo root.
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	protoDir := filepath.Join(repoRoot, "api", "proto", "sdx", "v1")

	expectedProtos := []string{
		"common.proto",
		"gateway.proto",
		"workitem.proto",
		"build.proto",
		"artifact.proto",
		"tenant.proto",
		"audit.proto",
		"compliance.proto",
		"deploy.proto",
		"agent.proto",
		"llm.proto",
		"quality.proto",
		"events.proto",
	}

	for _, proto := range expectedProtos {
		path := filepath.Join(protoDir, proto)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected proto file missing: %s", proto)
		}
	}
}

// TestBufYamlExists verifies that buf configuration is present.
func TestBufYamlExists(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..")

	configs := []string{
		filepath.Join(repoRoot, "api", "proto", "buf.yaml"),
		filepath.Join(repoRoot, "api", "proto", "buf.gen.yaml"),
	}

	for _, cfg := range configs {
		if _, err := os.Stat(cfg); os.IsNotExist(err) {
			t.Errorf("buf config missing: %s", filepath.Base(cfg))
		}
	}
}

// TestProtoHasHTTPAnnotations does a basic check that the gateway proto
// imports annotations (a prerequisite for grpc-gateway OpenAPI generation).
func TestProtoHasHTTPAnnotations(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	gatewayProto := filepath.Join(repoRoot, "api", "proto", "sdx", "v1", "gateway.proto")

	data, err := os.ReadFile(gatewayProto)
	if err != nil {
		t.Fatalf("cannot read gateway.proto: %v", err)
	}

	content := string(data)

	checks := []struct {
		needle      string
		description string
	}{
		{`google/api/annotations.proto`, "imports google.api.http annotations"},
		{`google.api.http`, "has HTTP annotations on RPCs"},
		{`protoc_gen_openapiv2`, "has OpenAPI swagger options"},
	}

	for _, check := range checks {
		if !containsString(content, check.needle) {
			t.Errorf("gateway.proto: expected to find %q (%s)", check.needle, check.description)
		}
	}
}

func containsString(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && filepath.Base(haystack) != "" && // avoid import
		stringContains(haystack, needle)
}

func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
