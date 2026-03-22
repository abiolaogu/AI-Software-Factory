# API Testing Playbook

## Purpose

This playbook is the operating guide for validating API correctness, documentation integrity, and runtime drift in NeuroForge SDX.

## Recommended Workflow

1. Sync the contract artifacts.

```bash
make proto
make openapi
```

2. Validate hosted documentation.

```bash
make swagger
make redoc
```

3. Run the compliance checks in layers.

```bash
make contract-test
make auth-test
make quota-test
make negative-test
```

4. Run the full compliance terminal before merging.

```bash
make compliance-test
```

## Mocking Strategy

- Default mock engine: repository-native mock API via `cmd/mock-api`.
- Optional standards mock engine: Prism through `COMPLIANCE_MOCK_ENGINE=prism`.
- Use the internal mock when you need a zero-dependency local workflow.
- Use Prism when you want spec-only mock behavior.
- Override the mock port with `COMPLIANCE_MOCK_PORT`.
- Override the docs host with `COMPLIANCE_DOCS_PORT` or `COMPLIANCE_DOCS_ADDR`.

## What Each Test Layer Covers

- Contract: documented routes, request validation, response validation, and docs routes.
- Auth: missing Authorization header, missing tenant header, tenant-claim drift.
- Quota: rate-limit headers, pagination behavior, idempotency replay semantics.
- Negative: error schema validity, status-code drift, and example correctness.

## Troubleshooting

- If `make openapi` fails, regenerate and inspect `api/openapi/openapi.yaml` and `api/openapi/openapi.json`.
- If `make contract-test` fails, compare runtime routes with the OpenAPI paths first.
- If `make auth-test` fails, inspect [router.go](/Users/AbiolaOgunsakin1/Workflows/AI%20Software%20Factory/internal/adapters/http/router.go).
- If `make negative-test` fails, inspect request and response examples in [openapi.yaml](/Users/AbiolaOgunsakin1/Workflows/AI%20Software%20Factory/api/openapi/openapi.yaml).
- If docs checks fail, run `make serve-docs` and verify `/swagger`, `/docs`, and `/openapi.json`.
- If the mock server port is already taken, rerun with `COMPLIANCE_MOCK_PORT=18090 make mock-api`.
