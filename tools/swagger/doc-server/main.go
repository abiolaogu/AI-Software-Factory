// Package main implements a documentation server for the NeuroForge SDX OpenAPI spec.
// It serves Swagger UI, Redoc, and the raw OpenAPI JSON at /swagger, /docs, and /openapi.json.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

const (
	// Pinned Swagger UI version with SRI-like versioning.
	swaggerUIVersion = "5.17.14"

	// Pinned Redoc version.
	redocVersion = "2.1.5"
)

func main() {
	specFile := "../../api/openapi/sdx-gateway-v1.swagger.json"
	if len(os.Args) > 1 {
		specFile = os.Args[1]
	}

	port := "8081"
	if p := os.Getenv("DOC_SERVER_PORT"); p != "" {
		port = p
	}

	// Serve raw OpenAPI spec.
	http.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		http.ServeFile(w, r, specFile)
	})

	// Serve Swagger UI.
	http.HandleFunc("/swagger", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>NeuroForge SDX — Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@%s/swagger-ui.css"
        integrity=""
        crossorigin="anonymous">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@%s/swagger-ui-bundle.js"
          integrity=""
          crossorigin="anonymous"></script>
  <script>
    SwaggerUIBundle({
      url: '/openapi.json',
      dom_id: '#swagger-ui',
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: 'StandaloneLayout',
      deepLinking: true,
      tryItOutEnabled: true
    });
  </script>
</body>
</html>`, swaggerUIVersion, swaggerUIVersion)
	})

	// Serve Redoc.
	http.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>NeuroForge SDX — API Reference</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body>
  <redoc spec-url='/openapi.json'
         hide-download-button
         theme='{"colors":{"primary":{"main":"#6366f1"}}}'></redoc>
  <script src="https://cdn.redoc.ly/redoc/%s/bundles/redoc.standalone.js"
          integrity=""
          crossorigin="anonymous"></script>
</body>
</html>`, redocVersion)
	})

	// Health check for the doc server itself.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	log.Printf("Documentation server starting on :%s", port)
	log.Printf("  Swagger UI: http://localhost:%s/swagger", port)
	log.Printf("  Redoc:      http://localhost:%s/docs", port)
	log.Printf("  OpenAPI:    http://localhost:%s/openapi.json", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
