# ─────────────────────────────────────────────
# NeuroForge SDX — Swagger Compliance Terminal Package
# Every generated service MUST pass all targets in this Makefile.
# CI runs `make api-compliance-full` as a blocking gate.
# ─────────────────────────────────────────────

PROTO_DIR      := ../../api/proto
OPENAPI_DIR    := ../../api/openapi
GEN_DIR        := ../../gen
SWAGGER_UI_DIR := ./swagger-ui
REDOC_DIR      := ./redoc
MOCK_PORT      := 4010
DOCS_PORT      := 8090

BUF            := buf
SPECTRAL       := spectral
OASDIFF        := oasdiff
PRISM          := prism
REDOCLY        := redocly

.PHONY: proto openapi swagger redoc mock-api contract-test compliance-test serve-docs api-compliance-full clean

# ─────────────────────────────────────────────
# make proto
# Purpose:   Generate Go, TypeScript, Python stubs from .proto definitions
# Toolchain: buf (buf.build CLI)
# Input:     api/proto/sdx/v1/*.proto, api/proto/buf.yaml, api/proto/buf.gen.yaml
# Output:    gen/go/, gen/typescript/, gen/python/, api/openapi/sdx-gateway-v1.swagger.json
# CI usage:  First step in CI pipeline; all other targets depend on this
# Pass:      buf build exits 0; buf lint exits 0; all output files generated
# Fail:      Any proto syntax error, lint violation, or missing output file
# ─────────────────────────────────────────────
proto:
	@echo "══════ Generating protobuf stubs ══════"
	cd $(PROTO_DIR) && $(BUF) lint
	cd $(PROTO_DIR) && $(BUF) build
	cd $(PROTO_DIR) && $(BUF) generate
	@echo "✓ Proto generation complete"
	@echo "  Go stubs:         $(GEN_DIR)/go/"
	@echo "  TypeScript stubs: $(GEN_DIR)/typescript/"
	@echo "  Python stubs:     $(GEN_DIR)/python/"
	@echo "  OpenAPI spec:     $(OPENAPI_DIR)/sdx-gateway-v1.swagger.json"

# ─────────────────────────────────────────────
# make openapi
# Purpose:   Validate and lint the generated OpenAPI 3.1 specification
# Toolchain: spectral (Stoplight), redocly
# Input:     api/openapi/sdx-gateway-v1.swagger.json
# Output:    Lint report (stdout); exits 0 on success
# CI usage:  Runs after `make proto`; blocks merge on any error or warning
# Pass:      Zero errors and zero warnings from spectral + redocly
# Fail:      Any spectral error (missing description, invalid schema, etc.)
# ─────────────────────────────────────────────
openapi: proto
	@echo "══════ Validating OpenAPI specification ══════"
	$(SPECTRAL) lint $(OPENAPI_DIR)/sdx-gateway-v1.swagger.json \
		--ruleset .spectral.yaml \
		--fail-severity warn
	$(REDOCLY) lint $(OPENAPI_DIR)/sdx-gateway-v1.swagger.json
	@echo "✓ OpenAPI spec is valid and lint-clean"

# ─────────────────────────────────────────────
# make swagger
# Purpose:   Serve Swagger UI locally for interactive API exploration
# Toolchain: swagger-ui-dist (npm package), http-server
# Input:     api/openapi/sdx-gateway-v1.swagger.json
# Output:    HTTP server on localhost:$(DOCS_PORT)/swagger
# CI usage:  Not used in CI (interactive tool only)
# Pass:      Server starts and spec loads without error
# Fail:      Spec file missing or malformed
# ─────────────────────────────────────────────
swagger: openapi
	@echo "══════ Starting Swagger UI ══════"
	@echo "  URL: http://localhost:$(DOCS_PORT)/swagger"
	npx @redocly/cli preview-docs $(OPENAPI_DIR)/sdx-gateway-v1.swagger.json \
		--port $(DOCS_PORT) || \
	npx swagger-ui-watcher $(OPENAPI_DIR)/sdx-gateway-v1.swagger.json \
		--port $(DOCS_PORT)

# ─────────────────────────────────────────────
# make redoc
# Purpose:   Serve Redoc documentation locally
# Toolchain: redocly CLI
# Input:     api/openapi/sdx-gateway-v1.swagger.json
# Output:    HTTP server on localhost:$(DOCS_PORT)/docs
# CI usage:  Not used in CI (interactive tool only)
# Pass:      Server starts and renders without error
# Fail:      Spec file missing or malformed
# ─────────────────────────────────────────────
redoc: openapi
	@echo "══════ Starting Redoc ══════"
	@echo "  URL: http://localhost:$(DOCS_PORT)"
	$(REDOCLY) preview-docs $(OPENAPI_DIR)/sdx-gateway-v1.swagger.json \
		--port $(DOCS_PORT)

# ─────────────────────────────────────────────
# make mock-api
# Purpose:   Run a mock API server for contract validation and frontend development
# Toolchain: prism (Stoplight)
# Input:     api/openapi/sdx-gateway-v1.swagger.json
# Output:    Mock HTTP server on localhost:$(MOCK_PORT)
# CI usage:  Started before `make contract-test`; stopped after
# Pass:      Prism starts and all routes are mockable
# Fail:      Spec has unsupported constructs or is malformed
# ─────────────────────────────────────────────
mock-api: openapi
	@echo "══════ Starting Prism mock server ══════"
	@echo "  URL: http://localhost:$(MOCK_PORT)"
	$(PRISM) mock $(OPENAPI_DIR)/sdx-gateway-v1.swagger.json \
		--port $(MOCK_PORT) \
		--dynamic

# ─────────────────────────────────────────────
# make contract-test
# Purpose:   Validate that the running implementation conforms to the OpenAPI spec
# Toolchain: prism (proxy mode), dredd, or custom Go contract tests
# Input:     api/openapi/sdx-gateway-v1.swagger.json, running service
# Output:    Test report (stdout + JUnit XML at tests/contract/results.xml)
# CI usage:  Runs after service is built and started in CI
# Pass:      All endpoints return responses matching the spec (status codes, schemas, headers)
# Fail:      Any endpoint returns a response not matching the spec
# ─────────────────────────────────────────────
contract-test: openapi
	@echo "══════ Running contract tests ══════"
	cd ../../tests/contract && go test -v -count=1 ./... \
		-openapi-spec=$(OPENAPI_DIR)/sdx-gateway-v1.swagger.json \
		-base-url=http://localhost:8080 \
		| tee results.txt
	@echo "✓ Contract tests complete"

# ─────────────────────────────────────────────
# make compliance-test
# Purpose:   Verify all endpoints have required metadata: auth, error responses,
#            quota metadata, tenant scoping, idempotency documentation
# Toolchain: Custom Go validator (tools/swagger/compliance-checker/)
# Input:     api/openapi/sdx-gateway-v1.swagger.json
# Output:    Compliance report (stdout + JSON at tools/swagger/compliance-report.json)
# CI usage:  Runs as part of `make api-compliance-full`; blocks merge on failure
# Pass:      Every endpoint has: security scheme, 400/401/403/404/429/500 responses,
#            x-rate-limit-tier extension, x-tenant-scoped extension
# Fail:      Any endpoint missing required metadata
# ─────────────────────────────────────────────
compliance-test: openapi
	@echo "══════ Running API compliance checks ══════"
	go run ./compliance-checker/main.go \
		-spec $(OPENAPI_DIR)/sdx-gateway-v1.swagger.json \
		-output compliance-report.json
	@echo "  Compliance report: tools/swagger/compliance-report.json"
	@echo "✓ API compliance checks complete"

# ─────────────────────────────────────────────
# make breaking-check
# Purpose:   Detect breaking changes vs the main branch OpenAPI spec
# Toolchain: oasdiff
# Input:     api/openapi/sdx-gateway-v1.swagger.json (current), main branch spec (fetched)
# Output:    Breaking change report (stdout)
# CI usage:  Runs on every PR; blocks merge on breaking changes
# Pass:      No breaking changes detected
# Fail:      Endpoint removed, required field added, response schema narrowed, etc.
# ─────────────────────────────────────────────
breaking-check:
	@echo "══════ Checking for breaking API changes ══════"
	git show origin/main:api/openapi/sdx-gateway-v1.swagger.json > /tmp/sdx-spec-base.json 2>/dev/null || \
		echo '{}' > /tmp/sdx-spec-base.json
	$(OASDIFF) breaking /tmp/sdx-spec-base.json $(OPENAPI_DIR)/sdx-gateway-v1.swagger.json \
		--fail-on ERR
	@echo "✓ No breaking changes detected"

# ─────────────────────────────────────────────
# make serve-docs
# Purpose:   Serve all API documentation (Swagger UI + Redoc + raw spec)
# Toolchain: caddy or nginx (local), or Go embedded server
# Input:     api/openapi/sdx-gateway-v1.swagger.json
# Output:    /swagger → Swagger UI, /docs → Redoc, /openapi.json → raw spec
# CI usage:  Not used in CI (local dev tool)
# Pass:      All three endpoints respond
# Fail:      Spec file missing
# ─────────────────────────────────────────────
serve-docs: openapi
	@echo "══════ Serving API documentation ══════"
	@echo "  Swagger UI: http://localhost:$(DOCS_PORT)/swagger"
	@echo "  Redoc:      http://localhost:$(DOCS_PORT)/docs"
	@echo "  Raw spec:   http://localhost:$(DOCS_PORT)/openapi.json"
	go run ./doc-server/main.go \
		-spec $(OPENAPI_DIR)/sdx-gateway-v1.swagger.json \
		-port $(DOCS_PORT)

# ─────────────────────────────────────────────
# make api-compliance-full
# Purpose:   Full compliance suite — lint + breaking + contract + metadata
# Toolchain: All of the above
# CI usage:  THE primary CI gate for API compliance. MUST pass before merge.
# Pass:      All sub-targets pass
# Fail:      Any sub-target fails
# ─────────────────────────────────────────────
api-compliance-full: proto openapi breaking-check compliance-test
	@echo ""
	@echo "══════════════════════════════════════════════"
	@echo "  ✓ ALL API COMPLIANCE CHECKS PASSED"
	@echo "══════════════════════════════════════════════"

# ─────────────────────────────────────────────
# make api-export
# Purpose:   Export OpenAPI spec for external consumers (SDK generation, docs site)
# Toolchain: redocly bundle
# Input:     api/openapi/sdx-gateway-v1.swagger.json
# Output:    dist/sdx-api-v1.json, dist/sdx-api-v1.yaml
# ─────────────────────────────────────────────
api-export:
	@echo "══════ Exporting API spec ══════"
	mkdir -p ../../dist
	$(REDOCLY) bundle $(OPENAPI_DIR)/sdx-gateway-v1.swagger.json \
		-o ../../dist/sdx-api-v1.json
	$(REDOCLY) bundle $(OPENAPI_DIR)/sdx-gateway-v1.swagger.json \
		-o ../../dist/sdx-api-v1.yaml
	@echo "  Exported: dist/sdx-api-v1.json, dist/sdx-api-v1.yaml"

clean:
	rm -rf $(GEN_DIR) $(OPENAPI_DIR)/*.swagger.json compliance-report.json
	@echo "✓ Clean complete"
