# ADR-016: Event Proto Payload Coverage

**Status:** Accepted
**Date:** 2026-03-21
**Deciders:** Chief Software Factory Architect

## Context

The domain event catalog defines 60+ events. The initial events.proto only defined 10 payload messages. Without typed proto payloads, events would be serialised as untyped maps or raw JSON — violating Holy Trinity correctness guarantees and preventing Codex from generating type-safe event publishers/subscribers.

## Decision

**All domain events MUST have typed proto payload messages in events.proto.** Events without payloads cannot be published. The DomainEvent envelope's `payload` field (google.protobuf.Any) wraps the specific payload proto.

## Implementation

events.proto now defines 50+ payload messages covering:
- 6 WorkItem events
- 8 Build lifecycle events (including initiated, checkpointed, recovery, escalation)
- 5 Quality gate events (evaluated, passed, failed, remediation start/succeed/exhaust)
- 4 Artifact events (created, published, deployed, revoked)
- 7 Tenant events (created, plan changed, suspended, reactivated, quota, workspace)
- 5 Agent events (registered, assigned, released, recovered, terminated)
- 4 Security events (scan completed, vulnerability, SBOM, cleared)
- 3 Compliance events (mapping, evidence, gap)
- 7 Deployment events (started, smoke pass/fail, SLO, completed, rollback)
- 3 LLM events (request completed, circuit opened/closed)
- 4 Webhook delivery events (dispatched, delivered, failed, dead letter)

## Consequences

- **Type safety guaranteed** across NATS and Redpanda transports.
- **Codex generates typed publishers/subscribers** from proto stubs.
- **Schema evolution** managed via proto field numbering (additive only).
- **events.proto is the largest proto file** (~400 lines). Acceptable — all payloads share a single package.
