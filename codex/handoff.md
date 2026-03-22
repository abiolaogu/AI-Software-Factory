# NeuroForge SDX — Codex Handoff Package

**From:** Chief Software Factory Architect (Claude)
**To:** Builder (Codex)
**Date:** 2026-03-22
**Architecture Baseline:** `v0.1.0-architecture` (tag on commit `c9e7c17`)

---

## 1. What You're Building

NeuroForge SDX is an autonomous AI software factory. It takes a PRD as input and produces production-grade, signed, compliance-certified software as output — through a 16-stage pipeline orchestrated by Temporal, powered by LLM agents, and governed by 14 quality gates.

You are implementing the **Go backend** for this system: domain layer, infrastructure adapters, gRPC services, Temporal workflows, and the API gateway.

---

## 2. Where Everything Is

| Artifact | Path | What It Contains |
|----------|------|-----------------|
| **factory.yaml** | `codex/factory.yaml` | Build manifest: tech stack, services, conventions, Codex rules |
| **Backlog** | `codex/backlog.md` | M1-M8 milestones with 70+ tasks, acceptance criteria, effort estimates |
| **Service Matrix** | `codex/service-matrix.md` | 21 services: ports, protos, databases, dependencies, implementation order |
| **Acceptance Criteria** | `codex/acceptance-criteria.md` | 100+ testable assertions per aggregate, service, and E2E scenario |
| **Aggregates** | `docs/domain/aggregates.md` | 9 aggregates with fields, invariants, entities |
| **Value Objects** | `docs/domain/value-objects.md` | 35+ VOs with validation rules |
| **State Machines** | `docs/domain/state-machines.md` | 7 state machines with transition tables and stage-to-status mapping |
| **Domain Events** | `docs/domain/domain-events.md` | 60+ events with NATS subjects and JetStream config |
| **Use Cases** | `docs/domain/use-cases.md` | 20 use cases with step-by-step flows |
| **Repository Interfaces** | `docs/domain/repository-interfaces.md` | 15 Go interfaces |
| **Policies** | `docs/domain/policies-validations.md` | 10 policies, 20+ validation rules |
| **Edge Cases** | `docs/domain/edge-cases.md` | 28 failure modes with handling strategies |
| **Internal Services** | `docs/domain/internal-service-interfaces.md` | 10 Temporal activity Go interfaces |
| **Proto Definitions** | `api/proto/sdx/v1/*.proto` | 13 proto files, 86 RPCs, 100+ messages |
| **REST Mapping** | `docs/api/rest-mapping.md` | All 86 RPCs mapped to REST endpoints |
| **Error Catalog** | `docs/api/error-catalog.md` | 60+ error codes (SDX-1xxx through SDX-9xxx) |
| **API Contracts** | `docs/api/auth-tenant-webhook-ratelimit-idempotency.md` | Auth, tenant, webhook, rate-limit, idempotency specs |
| **ADRs** | `docs/adr/ADR-013..016` | Internal services, Build state model, Spectral, Event payloads |
| **Architecture Review** | `docs/architecture-review-v1.md` | Full review with compliance verdict |

---

## 3. How to Start (M1 Step-by-Step)

### Step 1: Initialise Project

```bash
mkdir -p cmd domain/shared domain/workitem domain/build domain/artifact \
  domain/tenant domain/agent domain/deployment domain/quality domain/compliance \
  internal/adapter internal/workflow internal/gateway gen tests
go mod init github.com/stratogroup/neuroforge-sdx
```

### Step 2: Implement `domain/shared/`

Read `docs/domain/value-objects.md` section "Identity Value Objects" and "Common Types" from `docs/domain/repository-interfaces.md`. Implement:
- `types.go` — ULID, ContentHash, TenantContext, Pagination
- `errors.go` — All error sentinels
- `event_envelope.go` — DomainEventEnvelope struct

### Step 3: Implement Value Objects

Read `docs/domain/value-objects.md` entirely. Implement every VO as a Go struct with:
- Constructor that validates
- Equality method
- `String()` for logging

### Step 4: Implement Aggregates (TDD)

For each aggregate (WorkItem, Build, Artifact, Tenant, Agent, Deployment, DomainModel, Blueprint, Contract):

1. **Read** the aggregate section in `docs/domain/aggregates.md`
2. **Read** the state machine in `docs/domain/state-machines.md`
3. **Write failing tests** for every invariant (from `codex/acceptance-criteria.md`)
4. **Implement** the aggregate to make tests pass
5. **Verify** domain events are emitted on every state transition

### Step 5: Implement Policies

Read `docs/domain/policies-validations.md`. Implement each policy as a pure function.

### Step 6: Compile Protos

```bash
cd api/proto && buf lint && buf build && buf generate
```

### Step 7: Run Tests

```bash
go test ./domain/... -v -count=1 -cover
```

**M1 is complete when all domain tests pass with ≥90% coverage.**

---

## 4. Critical Rules (Do Not Violate)

1. **TDD is mandatory.** Write the failing test FIRST. Then implement. Then refactor.
2. **Every invariant must have a test.** If aggregates.md says "MUST NOT", there must be a test proving it.
3. **No business logic outside `domain/`.** Adapters are dumb translators.
4. **No ORM.** Raw SQL with `sqlx` for YugabyteDB. ClickHouse Go driver for analytics.
5. **Every SQL query has `WHERE tenant_id = $1`.** No exceptions.
6. **ULID for all IDs.** `github.com/oklog/ulid/v2`. No UUID. No auto-increment.
7. **Apache-2.0 / MIT / BSD dependencies only.** No GPL. No AGPL. Check every `go get`.
8. **Structured logging only.** `slog` with JSON output. Always include `tenant_id`, `build_id`, `trace_id`.
9. **Context propagation.** `tenant_id` flows from gRPC metadata → `context.Context` → all layers.
10. **Proto payloads for events.** Use `google.protobuf.Any` wrapping the typed payload from `events.proto`.

---

## 5. Verification Handoff to Antigravity

After each milestone, Antigravity verifies:

| Milestone | Antigravity Checks |
|-----------|-------------------|
| M1 | All invariant tests pass. All state machine tests pass. All 28 edge case tests exist. Coverage ≥90%. No cyclomatic complexity > 10. |
| M2 | All integration tests pass with testcontainers. Contract tests pass. API compliance passes. No breaking proto changes. |
| M3 | E2E pipeline test passes. Remediation loop works. Recovery from checkpoint works. |
| M4+ | Performance targets met. Security scans clean. Compliance evidence generated. |

---

## 6. Definition of Done (Per Task)

A task is DONE when:
- [ ] Failing tests written first (TDD)
- [ ] Implementation makes all tests green
- [ ] GoDoc comments on all public functions
- [ ] `golangci-lint run` passes
- [ ] No hardcoded secrets
- [ ] All invariants from architecture docs are enforced and tested
- [ ] Domain events emitted for every state change
- [ ] Error codes from error-catalog.md used (not generic errors)

---

## 7. What NOT to Do

- Do NOT create your own error codes. Use only codes from `docs/api/error-catalog.md`.
- Do NOT invent new aggregate states. Use only states from `docs/domain/state-machines.md`.
- Do NOT skip proto compilation. All event payloads must be proto-generated types.
- Do NOT use `interface{}` or `any` for domain types. Everything is strongly typed.
- Do NOT add dependencies without checking the license.
- Do NOT implement M2 before M1 is 100% complete and verified.

---

**Start with M1-001. Proceed in order. Ask Architect (Claude) if anything is ambiguous. Do not guess.**
