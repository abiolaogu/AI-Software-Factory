package artifacts_test

import (
	"context"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stratogroup/neuroforge-sdx/internal/adapters/artifacts"
	"github.com/stretchr/testify/require"
)

func TestSignedArtifactServiceWorkflowArtifactsAreTenantScoped(t *testing.T) {
	service, err := artifacts.NewSignedArtifactService("neuroforge-sdx", "https://artifacts.example.com/platform", "signing-key", 15*time.Minute)
	require.NoError(t, err)

	now := time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC)
	workflowArtifacts, err := service.WorkflowArtifacts(context.Background(), "tenant-a", "workflow-123", now)
	require.NoError(t, err)

	manifest, ok := workflowArtifacts["manifest"]
	require.True(t, ok)

	parsed, err := url.Parse(manifest.URL)
	require.NoError(t, err)
	require.Equal(t, "/platform/workflows/workflow-123/manifest.json", parsed.Path)
	require.Equal(t, "tenant-a", parsed.Query().Get("tenant"))
	require.Equal(t, strconv.FormatInt(now.Add(15*time.Minute).Unix(), 10), parsed.Query().Get("expires"))
	require.NotEmpty(t, parsed.Query().Get("signature"))
	require.Equal(t, "application/json", manifest.Type)
}

func TestSignedArtifactServiceGeneratesComplianceEvidence(t *testing.T) {
	service, err := artifacts.NewSignedArtifactService("neuroforge-sdx", "https://artifacts.example.com", "signing-key", 30*time.Minute)
	require.NoError(t, err)

	now := time.Date(2026, 3, 22, 12, 30, 0, 0, time.UTC)
	evidence, err := service.SecurityEvidence(context.Background(), "tenant-a", "scan-123", now)
	require.NoError(t, err)

	sbomURL, err := url.Parse(evidence.SBOM)
	require.NoError(t, err)
	require.Equal(t, "/compliance/scan-123/sbom.spdx.json", sbomURL.Path)
	require.Equal(t, "tenant-a", sbomURL.Query().Get("tenant"))
	require.NotEmpty(t, sbomURL.Query().Get("signature"))

	require.Contains(t, evidence.Signature, "cosign://rekor/neuroforge-sdx/scan-123")
	require.Contains(t, evidence.Signature, "signature=")
}

func TestSignedArtifactServiceDesignArtifactsAreSigned(t *testing.T) {
	service, err := artifacts.NewSignedArtifactService("neuroforge-sdx", "https://artifacts.example.com/platform", "signing-key", 20*time.Minute)
	require.NoError(t, err)

	now := time.Date(2026, 3, 22, 13, 0, 0, 0, time.UTC)
	designArtifacts, err := service.DesignArtifacts(context.Background(), "tenant-a", "design-123", now)
	require.NoError(t, err)
	require.Len(t, designArtifacts.Wireframes, 2)
	require.Len(t, designArtifacts.HifiDesigns, 1)

	for _, asset := range append(append([]string{}, designArtifacts.Wireframes[0].URL, designArtifacts.Wireframes[1].URL), designArtifacts.HifiDesigns[0].URL) {
		parsed, err := url.Parse(asset)
		require.NoError(t, err)
		require.Equal(t, "tenant-a", parsed.Query().Get("tenant"))
		require.NotEmpty(t, parsed.Query().Get("signature"))
	}
}

func TestNewSignedArtifactServiceRejectsInvalidBaseURL(t *testing.T) {
	_, err := artifacts.NewSignedArtifactService("neuroforge-sdx", "://bad-url", "signing-key", time.Minute)
	require.Error(t, err)
}
