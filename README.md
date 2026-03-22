# NeuroForge SDX

Production scaffold for the NeuroForge SDX service surface, generated from the attached NeuroForge SDX PRD and OpenAPI 3.1 artifact.

## What is included

- Go API server, worker, and mock API entry points
- OpenAPI 3.1 source and hosted docs endpoints
- DDD-oriented `internal/domain`, `internal/usecases`, `internal/adapters`, `internal/ports`
- Helm, KubeVela, and Fleet deployment starters
- Testcontainers integration harness
- Build, lint, test, coverage, security, compliance, and contract scripts
- GitHub Actions CI workflow skeletons

## Quick start

```bash
make setup
make run
```

The API server exposes:

- `GET /swagger`
- `GET /docs`
- `GET /openapi.json`
- gRPC `neuroforge.sdx.v1.SDXService` on `NF_GRPC_ADDR` using [api/proto/neuroforge/sdx/v1/sdx.proto](/Users/AbiolaOgunsakin1/Workflows/AI%20Software%20Factory/api/proto/neuroforge/sdx/v1/sdx.proto)
- `GET /health/live`
- `GET /health/ready`

Key runtime configuration:

- `NF_YUGABYTE_DSN` enables the YugabyteDB or PostgreSQL-compatible workflow store
- `NF_GRPC_ADDR` controls the gRPC listener for the canonical proto transport
- `NF_NATS_URL` enables regional NATS connectivity, and `NF_NATS_STREAM` opts the publisher into JetStream-backed event durability
- `NF_ARTIFACT_BASE_URL`, `NF_ARTIFACT_SIGNING_KEY`, and `NF_ARTIFACT_URL_TTL` control signed artifact and compliance evidence links

When those platform endpoints are not configured, the service still runs locally with the existing development-safe adapters and continues to expose the full OpenAPI contract. Outside `local` and `test`, startup now validates that production-safe database, artifact-signing, and auth settings are present before serving traffic.

## Source of truth

- Architecture artifact: `NeuroForge SDX PRD v1.0`
- API artifact: [api/openapi/openapi.yaml](/Users/AbiolaOgunsakin1/Workflows/AI%20Software%20Factory/api/openapi/openapi.yaml)

The attached API artifact is preserved as the repository contract baseline, including its upstream `info.title`.
