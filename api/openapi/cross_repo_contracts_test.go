package openapi_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

type consumerLockFile struct {
	SchemaVersion       string `json:"schemaVersion"`
	AuthorityRepository string `json:"authorityRepository"`
	AuthorityRevision   string `json:"authorityRevision"`
	Contracts           []struct {
		ContractID         string   `json:"contractId"`
		Version            string   `json:"version"`
		SchemaPath         string   `json:"schemaPath"`
		ConsumerSchemaPath string   `json:"consumerSchemaPath"`
		Producer           string   `json:"producer"`
		Consumers          []string `json:"consumers"`
		SHA256             string   `json:"sha256"`
	} `json:"contracts"`
}

type compatibilityManifest struct {
	RepoVersion         string `json:"repoVersion"`
	ConsumerLockPath    string `json:"consumerLockPath"`
	CompatibleConsumers []struct {
		Repo              string   `json:"repo"`
		GitHubRepo        string   `json:"githubRepo"`
		ConsumerLockPath  string   `json:"consumerLockPath"`
		ProducesContracts []string `json:"producesContracts"`
		ConsumesContracts []string `json:"consumesContracts"`
	} `json:"compatibleConsumers"`
}

func TestCrossRepoConsumerLockDigestsAndBidirectionalRegistry(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))

	lockBytes, err := os.ReadFile(filepath.Join(root, "compatibility", "cross-repo-consumer-lock.json"))
	if err != nil {
		t.Fatalf("read cross-repo-consumer-lock.json: %v", err)
	}
	var lock consumerLockFile
	if err := json.Unmarshal(lockBytes, &lock); err != nil {
		t.Fatalf("unmarshal cross-repo-consumer-lock.json: %v", err)
	}
	if len(lock.Contracts) != 5 {
		t.Fatalf("expected 5 cross-repo boundary contracts, got %d", len(lock.Contracts))
	}

	compatBytes, err := os.ReadFile(filepath.Join(root, "compatibility", "manifest.json"))
	if err != nil {
		t.Fatalf("read compatibility/manifest.json: %v", err)
	}
	var compat compatibilityManifest
	if err := json.Unmarshal(compatBytes, &compat); err != nil {
		t.Fatalf("unmarshal compatibility/manifest.json: %v", err)
	}

	byRepo := map[string]struct {
		Produces []string
		Consumes []string
	}{}
	for _, c := range compat.CompatibleConsumers {
		byRepo[c.Repo] = struct {
			Produces []string
			Consumes []string
		}{
			Produces: c.ProducesContracts,
			Consumes: c.ConsumesContracts,
		}
	}

	for _, contract := range lock.Contracts {
		schemaBytes, err := os.ReadFile(filepath.Join(root, contract.SchemaPath))
		if err != nil {
			t.Fatalf("read schema %s: %v", contract.SchemaPath, err)
		}
		sum := sha256.Sum256(schemaBytes)
		gotSHA := hex.EncodeToString(sum[:])
		if gotSHA != contract.SHA256 {
			t.Fatalf("schema %s digest drift: got %s, want %s", contract.ContractID, gotSHA, contract.SHA256)
		}

		var schemaDoc map[string]any
		if err := json.Unmarshal(schemaBytes, &schemaDoc); err != nil {
			t.Fatalf("parse schema %s: %v", contract.ContractID, err)
		}
		req, ok := schemaDoc["required"].([]any)
		if !ok || len(req) == 0 {
			t.Fatalf("schema %s must declare non-empty required fields", contract.ContractID)
		}

		prod, exists := byRepo[contract.Producer]
		if !exists || !slices.Contains(prod.Produces, contract.ContractID) {
			t.Fatalf("producer %s missing producesContracts entry for %s", contract.Producer, contract.ContractID)
		}
		for _, consumer := range contract.Consumers {
			cons, exists := byRepo[consumer]
			if !exists || !slices.Contains(cons.Consumes, contract.ContractID) {
				t.Fatalf("consumer %s missing consumesContracts entry for %s", consumer, contract.ContractID)
			}
		}
	}
}
