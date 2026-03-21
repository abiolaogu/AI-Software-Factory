# ADR-013: Internal Service Communication Strategy

**Status:** Accepted
**Date:** 2026-03-21
**Deciders:** Chief Software Factory Architect

## Context

NeuroForge SDX has 20 bounded contexts. 8 have public-facing gRPC service protos (exposed via sdx-gateway). 12 are internal — invoked only by the Temporal Build Pipeline Workflow. The question is whether internal services need gRPC proto definitions.

## Decision

**M1: Temporal activity interfaces (Go function signatures).** Internal services define Go interfaces in `docs/domain/internal-service-interfaces.md`. Temporal activities implement these interfaces. No gRPC overhead for purely internal orchestration.

**M2: Add gRPC proto definitions for services requiring independent scaling or cross-language support.** This is additive — Go interfaces remain, gRPC is layered on top.

## Consequences

- **M1 is faster:** No proto generation for 12 services; Codex builds Go interfaces directly.
- **Trade-off:** Internal services cannot be called outside Temporal workflows in M1.
- **Migration path:** M2 adds gRPC without removing Go interfaces. Both coexist.
- **Affected services:** sdx-domain-svc, sdx-architect-svc, sdx-contract-svc, sdx-test-spec-svc, sdx-codegen-svc, sdx-security-svc, sdx-performance-svc, sdx-docs-svc, sdx-iac-svc, sdx-learning-svc.
