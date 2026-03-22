package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDocsListenAddr(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		t.Setenv("COMPLIANCE_DOCS_ADDR", "")
		t.Setenv("COMPLIANCE_DOCS_PORT", "")
		require.Equal(t, ":8091", docsListenAddr())
	})

	t.Run("prefers explicit address", func(t *testing.T) {
		t.Setenv("COMPLIANCE_DOCS_ADDR", ":9100")
		t.Setenv("COMPLIANCE_DOCS_PORT", "9200")
		require.Equal(t, ":9100", docsListenAddr())
	})

	t.Run("accepts bare port", func(t *testing.T) {
		t.Setenv("COMPLIANCE_DOCS_ADDR", "")
		t.Setenv("COMPLIANCE_DOCS_PORT", "9300")
		require.Equal(t, ":9300", docsListenAddr())
	})

	t.Run("accepts colon-prefixed port", func(t *testing.T) {
		t.Setenv("COMPLIANCE_DOCS_ADDR", "")
		t.Setenv("COMPLIANCE_DOCS_PORT", ":9400")
		require.Equal(t, ":9400", docsListenAddr())
	})
}
