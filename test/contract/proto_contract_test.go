package contract_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtoDocumentationCopyMatchesCanonicalProto(t *testing.T) {
	root := filepath.Join("..", "..")

	canonical, err := os.ReadFile(filepath.Join(root, "api/proto/neuroforge/sdx/v1/sdx.proto"))
	require.NoError(t, err)

	documented, err := os.ReadFile(filepath.Join(root, "docs/api/neuroforge-sdx.proto"))
	require.NoError(t, err)

	require.Equal(t, string(canonical), string(documented))
}

func TestCanonicalProtoCoversRuntimeOperations(t *testing.T) {
	root := filepath.Join("..", "..")

	canonical, err := os.ReadFile(filepath.Join(root, "api/proto/neuroforge/sdx/v1/sdx.proto"))
	require.NoError(t, err)

	rpcs := extractRPCNames(string(canonical))
	require.Subset(t, rpcs, []string{
		"Login",
		"RefreshToken",
		"HealthLive",
		"HealthReady",
		"CreateWorkflow",
		"GetWorkflow",
		"ListWorkflows",
		"CancelWorkflow",
		"GetWorkflowArtifacts",
		"GeneratePRD",
		"GetPRD",
		"GenerateWireframes",
		"ExtractTokens",
		"GenerateCode",
		"GenerateTests",
		"SecurityScan",
		"GetSecurityFindings",
		"PerformanceTest",
		"GetPerformanceResults",
	})
}

func extractRPCNames(proto string) []string {
	matches := regexp.MustCompile(`(?m)^\s*rpc\s+([A-Za-z0-9_]+)\s*\(`).FindAllStringSubmatch(proto, -1)
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			names = append(names, match[1])
		}
	}
	return names
}
