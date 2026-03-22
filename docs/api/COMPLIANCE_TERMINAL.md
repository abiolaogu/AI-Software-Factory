# API Compliance Terminal

The compliance terminal is the repository-native API validation environment for NeuroForge SDX. It gives every engineer one predictable set of commands for spec sync, docs hosting, mock execution, and drift detection.

## Quick Start

Run the full terminal suite:

```bash
make compliance-test
```

Run focused commands:

```bash
make proto
make openapi
make swagger
make redoc
make contract-test
make auth-test
make quota-test
make negative-test
```

Host documentation locally:

```bash
make serve-docs
```

This serves:

```text
http://localhost:8091/swagger
http://localhost:8091/docs
http://localhost:8091/openapi.json
http://localhost:8091/openapi.yaml
```

Run the mock API:

```bash
make mock-api
```

Override the mock port when you need to avoid collisions:

```bash
COMPLIANCE_MOCK_PORT=18090 make mock-api
```

By default this uses the repository mock API. To use Prism instead:

```bash
COMPLIANCE_MOCK_ENGINE=prism make mock-api
```

Override the standalone docs host address or port:

```bash
COMPLIANCE_DOCS_PORT=18091 make serve-docs
COMPLIANCE_DOCS_ADDR=127.0.0.1:18091 make serve-docs
```

## What The Compliance Terminal Enforces

- OpenAPI and JSON artifacts stay in sync.
- Proto documentation stays aligned with the canonical proto.
- Versioned runtime routes are documented.
- Swagger UI, Redoc, and raw spec routes are live.
- Positive-path requests match the OpenAPI contract.
- Auth, tenant scope, tenant-claim drift, negative responses, and idempotency are verified.
- Pagination and rate-limit headers are validated.
- Request and response examples are present and schema-valid.

## Zero-Guesswork Command Map

- `make proto`: proto drift check.
- `make openapi`: sync and validate OpenAPI artifacts.
- `make swagger`: validate Swagger UI hosting.
- `make redoc`: validate Redoc hosting.
- `make mock-api`: run the contract mock server.
- `make contract-test`: run route and response contract tests.
- `make auth-test`: run auth and tenant-scope drift tests.
- `make quota-test`: run pagination, rate-limit, and idempotency checks.
- `make negative-test`: run negative-path and example-validation checks.
- `make compliance-test`: run the complete compliance terminal suite.
- `make serve-docs`: run the standalone docs server.

## Recommended Local Flow

```bash
make proto
make openapi
make swagger
make redoc
make contract-test
make auth-test
make quota-test
make negative-test
make compliance-test
```
