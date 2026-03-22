APP_NAME := neuroforge-sdx
GO := go
BIN_DIR := bin

.PHONY: help setup tidy build build-api build-worker build-mock run run-worker run-mock fmt lint test test-unit test-integration test-contract coverage security compliance swagger docker-build docker-build-api docker-build-worker ci clean proto openapi redoc mock-api contract-test auth-test quota-test negative-test compliance-test serve-docs complexity helm-validate buf-lint buf-build buf-generate api-compliance

help:
	@printf "Available targets:\n"
	@printf "  setup            Install dependencies and sync generated artifacts\n"
	@printf "  proto            Validate proto drift and documentation copies\n"
	@printf "  openapi          Sync and validate OpenAPI artifacts\n"
	@printf "  swagger          Validate Swagger UI hosting assets\n"
	@printf "  redoc            Validate Redoc hosting assets\n"
	@printf "  mock-api         Run the compliance mock API\n"
	@printf "  contract-test    Run API contract and route drift tests\n"
	@printf "  auth-test        Run auth and tenant enforcement tests\n"
	@printf "  quota-test       Run rate-limit, pagination, and idempotency tests\n"
	@printf "  negative-test    Run negative-path and example validation tests\n"
	@printf "  compliance-test  Run the full compliance terminal suite\n"
	@printf "  serve-docs       Run the standalone Swagger/Redoc docs server\n"
	@printf "  build            Build api-server, worker, and mock-api binaries\n"
	@printf "  run              Run the API server locally\n"
	@printf "  run-worker       Run the worker locally\n"
	@printf "  run-mock         Run the mock API locally\n"
	@printf "  fmt              Format Go and shell sources\n"
	@printf "  lint             Run static analysis\n"
	@printf "  complexity       Enforce cyclomatic complexity threshold\n"
	@printf "  test             Run the default test suite\n"
	@printf "  test-integration Run the testcontainers integration harness\n"
	@printf "  test-contract    Run OpenAPI drift checks against the implementation\n"
	@printf "  coverage         Generate a coverage report\n"
	@printf "  security         Run security scanners\n"
	@printf "  helm-validate    Validate the Helm chart renders cleanly\n"
	@printf "  compliance       Run compliance and repository structure checks\n"
	@printf "  swagger          Sync and validate OpenAPI artifacts\n"
	@printf "  docker-build     Build all container images\n"
	@printf "  ci               Run the full CI-local verification suite\n"

setup: tidy proto openapi

tidy:
	$(GO) mod tidy

build: build-api build-worker build-mock

build-api:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/api-server ./cmd/api-server

build-worker:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/worker ./cmd/worker

build-mock:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/mock-api ./cmd/mock-api

proto:
	bash tools/compliance/proto.sh

openapi:
	bash tools/compliance/openapi.sh

swagger:
	bash tools/compliance/swagger.sh

redoc:
	bash tools/compliance/redoc.sh

mock-api:
	bash tools/compliance/mock-api.sh

run:
	$(GO) run ./cmd/api-server

run-worker:
	$(GO) run ./cmd/worker

run-mock:
	./scripts/contract/mock.sh

serve-docs:
	bash tools/compliance/serve-docs.sh

fmt:
	bash scripts/ci/fmt.sh

lint:
	bash scripts/ci/lint.sh

complexity:
	bash scripts/ci/complexity.sh

test: test-unit

test-unit:
	bash scripts/ci/test.sh

test-integration:
	RUN_INTEGRATION=1 $(GO) test ./test/integration/... -count=1 -v

test-contract:
	bash scripts/contract/test.sh

contract-test:
	bash tools/compliance/contract-test.sh

auth-test:
	bash tools/compliance/auth-test.sh

quota-test:
	bash tools/compliance/quota-test.sh

negative-test:
	bash tools/compliance/negative-test.sh

compliance-test:
	bash tools/compliance/compliance-test.sh

coverage:
	bash scripts/ci/coverage.sh

security:
	bash scripts/security/scan.sh

helm-validate:
	bash scripts/compliance/helm-validate.sh

compliance:
	bash scripts/compliance/check.sh

docker-build: docker-build-api docker-build-worker

docker-build-api:
	docker build -f Dockerfile.api -t $(APP_NAME)-api:local .

docker-build-worker:
	docker build -f Dockerfile.worker -t $(APP_NAME)-worker:local .

ci: fmt lint complexity test coverage test-contract contract-test auth-test quota-test negative-test security helm-validate compliance build

buf-lint:
	cd api/proto && buf lint

buf-build:
	cd api/proto && buf build

buf-generate:
	cd api/proto && buf generate

api-compliance:
	@if [ -d tools/swagger ] && [ -f tools/swagger/Makefile ]; then \
		$(MAKE) -C tools/swagger api-compliance-full; \
	else \
		echo "Swagger compliance tooling not configured"; \
	fi

clean:
	rm -rf $(BIN_DIR) coverage.out coverage.html
