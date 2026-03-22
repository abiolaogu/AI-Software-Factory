package spec_test

import (
	"testing"

	"github.com/stratogroup/neuroforge-sdx/internal/adapters/spec"
	"github.com/stretchr/testify/require"
)

func TestSpecProviderLoadsEmbeddedArtifacts(t *testing.T) {
	yamlBytes, err := spec.YAML()
	require.NoError(t, err)
	require.NotEmpty(t, yamlBytes)

	jsonBytes, err := spec.JSON()
	require.NoError(t, err)
	require.NotEmpty(t, jsonBytes)

	document, err := spec.Load()
	require.NoError(t, err)
	require.NotNil(t, document)
	require.Equal(t, "3.1.0", document.OpenAPI)
}
