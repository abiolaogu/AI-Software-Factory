package main

import (
	"encoding/json"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	const yamlPath = "api/openapi/openapi.yaml"
	const jsonPath = "api/openapi/openapi.json"

	yamlBody, err := os.ReadFile(yamlPath)
	if err != nil {
		log.Fatalf("reading %s: %v", yamlPath, err)
	}

	var document any
	if err := yaml.Unmarshal(yamlBody, &document); err != nil {
		log.Fatalf("parsing %s: %v", yamlPath, err)
	}

	jsonBody, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		log.Fatalf("serializing %s: %v", jsonPath, err)
	}

	jsonBody = append(jsonBody, '\n')
	if err := os.WriteFile(jsonPath, jsonBody, 0o600); err != nil {
		log.Fatalf("writing %s: %v", jsonPath, err)
	}
}
