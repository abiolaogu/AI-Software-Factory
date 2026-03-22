package contract_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationsAvoidUnsafeStatements(t *testing.T) {
	pattern := regexp.MustCompile(`(?i)\bDROP\s+TABLE\b|\bDROP\s+COLUMN\b|\bTRUNCATE\b`)

	root := filepath.Join("..", "..")
	files, err := filepath.Glob(filepath.Join(root, "internal/adapters/storage/migrations/*.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		body, err := os.ReadFile(file)
		require.NoError(t, err)
		require.Falsef(t, pattern.Match(body), "migration %s contains an unsafe statement", file)
	}
}
