package suite_test

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

func TestComplianceOperationsHaveExamplesAndCurlSamples(t *testing.T) {
	document, _ := newSpecRouter(t)

	for path, item := range document.Paths.Map() {
		for method, operation := range item.Operations() {
			require.NotEmptyf(t, operation.Extensions["x-codeSamples"], "%s %s is missing x-codeSamples", method, path)

			if operation.RequestBody != nil && operation.RequestBody.Value != nil {
				mediaType := operation.RequestBody.Value.Content.Get("application/json")
				if mediaType != nil {
					require.Truef(t, mediaType.Example != nil || len(mediaType.Examples) > 0, "%s %s is missing request examples", method, path)
				}
			}

			for code, response := range operation.Responses.Map() {
				mediaType := response.Value.Content.Get("application/json")
				if mediaType == nil {
					continue
				}
				require.Truef(t, mediaType.Example != nil || len(mediaType.Examples) > 0, "%s %s response %s is missing examples", method, path, code)
			}
		}
	}
}

func TestComplianceExamplesValidateAgainstSchemas(t *testing.T) {
	document, _ := newSpecRouter(t)

	for path, item := range document.Paths.Map() {
		for method, operation := range item.Operations() {
			if operation.RequestBody != nil && operation.RequestBody.Value != nil {
				mediaType := operation.RequestBody.Value.Content.Get("application/json")
				if mediaType != nil && mediaType.Schema != nil {
					validateExamples(t, mediaType.Schema, mediaType.Example, mediaType.Examples, method, path, "request")
				}
			}

			for code, response := range operation.Responses.Map() {
				mediaType := response.Value.Content.Get("application/json")
				if mediaType == nil || mediaType.Schema == nil {
					continue
				}
				validateExamples(t, mediaType.Schema, mediaType.Example, mediaType.Examples, method, path, "response "+code)
			}
		}
	}
}

func validateExamples(t *testing.T, schemaRef *openapi3.SchemaRef, direct any, named openapi3.Examples, method, path, location string) {
	t.Helper()
	require.NotNil(t, schemaRef)
	require.NotNil(t, schemaRef.Value)

	if direct != nil {
		require.NoErrorf(t, schemaRef.Value.VisitJSON(direct), "%s %s %s direct example is invalid", method, path, location)
	}

	for name, example := range named {
		require.NotNilf(t, example.Value, "%s %s %s example %s has no value", method, path, location, name)
		require.NoErrorf(t, schemaRef.Value.VisitJSON(example.Value.Value), "%s %s %s example %s is invalid", method, path, location, name)
	}
}
