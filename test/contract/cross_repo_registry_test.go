package contract_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

type crossRepoRegistry struct {
	SchemaVersion string `json:"schemaVersion"`
	Authority     string `json:"authority"`
	Contracts     []struct {
		ContractID string   `json:"contractId"`
		Version    string   `json:"version"`
		SchemaPath string   `json:"schemaPath"`
		Producer   string   `json:"producer"`
		Consumers  []string `json:"consumers"`
	} `json:"contracts"`
}

func TestCrossRepoContractRegistryIsCompleteAndVersioned(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	registryBytes, err := os.ReadFile(filepath.Join(root, "compatibility", "cross-repo-contracts.json"))
	require.NoError(t, err)

	var registry crossRepoRegistry
	require.NoError(t, json.Unmarshal(registryBytes, &registry))
	require.Equal(t, "1.0.0", registry.SchemaVersion)
	require.Equal(t, "ai-software-factory", registry.Authority)
	require.Len(t, registry.Contracts, 5)

	semver := regexp.MustCompile(`^[1-9][0-9]*\.[0-9]+\.[0-9]+$`)
	seen := map[string]struct{}{}
	for _, contract := range registry.Contracts {
		require.NotEmpty(t, contract.ContractID)
		require.True(t, semver.MatchString(contract.Version), contract.ContractID)
		require.NotEmpty(t, contract.Producer, contract.ContractID)
		require.NotEmpty(t, contract.Consumers, contract.ContractID)
		_, duplicate := seen[contract.ContractID]
		require.False(t, duplicate, contract.ContractID)
		seen[contract.ContractID] = struct{}{}

		schemaBytes, readErr := os.ReadFile(filepath.Join(root, contract.SchemaPath))
		require.NoError(t, readErr, contract.ContractID)
		var schema map[string]any
		require.NoError(t, json.Unmarshal(schemaBytes, &schema), contract.ContractID)
		require.Equal(t, "https://json-schema.org/draft/2020-12/schema", schema["$schema"], contract.ContractID)
		require.Equal(t, "object", schema["type"], contract.ContractID)
		require.NotEmpty(t, schema["required"], contract.ContractID)
		require.Equal(t, false, schema["additionalProperties"], contract.ContractID)
	}
}
