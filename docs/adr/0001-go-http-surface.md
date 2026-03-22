# ADR-0001: Go API Server and Worker Surface

## Status

Accepted

## Context

The attached NeuroForge SDX PRD routes high-concurrency APIs and microservices to Go with canonical cloud-native toolchains, and it explicitly requires Docker, Helm, Fleet, GitHub Actions, OpenAPI, observability, and a worker process from day one.

## Decision

The repository scaffold uses Go as the primary runtime for the HTTP API server, worker, and contract-test mock API. The service surface is organized around DDD-oriented `domain`, `usecases`, `ports`, and `adapters` packages, with OpenAPI 3.1 retained as the contract baseline and served directly from the runtime.

## Consequences

- The repository can compile, run, and expose the full API contract immediately.
- Business workflows remain behind use case boundaries and repository interfaces.
- Tenant-scoped persistence, NATS-backed event publishing, and signed artifact and compliance evidence generation ship behind stable ports and adapters from day one.
- Platform integrations remain replaceable so the service can migrate cleanly to shared hosted Hasura, JetStream, Vault or KMS, and evidence-signing services without reshaping the domain layer.
