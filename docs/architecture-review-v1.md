# NeuroForge SDX — Final Architecture Review v1.0

**Reviewer:** Chief Software Factory Architect (Claude)
**Date:** 2026-03-21
**Phase:** Pre-Implementation Architecture Integrity Gate
**Package Version:** Initial commit `7cc5293`
**Scope:** Domain Package (A), API Contract Package (B), Swagger Compliance (C), File Output Plan (D)

---

## 0. Architecture Compliance Verdict

### CONDITIONAL PASS — 4 Must-Fix Items Before Codex Handoff

The architecture package is **structurally sound** and represents a **production-grade DDD/TDD design**. The 20-context bounded context decomposition, 9-aggregate domain model, 60+ event catalog, and 86-RPC API contract are thorough and internally consistent at the macro level.

However, the cross-reference audit reveals **4 critical gaps** that will cause Codex to block, guess, or produce inconsistent code. These must be resolved before the Codex build phase begins.

| Verdict | Details |
|---------|---------|
| **Domain Model** | PASS — 9 aggregates, 50+ invariants, 7 state machines, 28 edge cases |
| **Proto Definitions** | CONDITIONAL PASS — 10 services defined, but 12 BC services missing |
| **Event System** | FAIL — 60+ events documented, only ~10 have proto payload messages |
| **Repository Interfaces** | CONDITIONAL PASS — 12 defined, 3 critical ones missing |
| **API Contracts** | PASS — REST mapping, error catalog, auth/webhook/rate-limit contracts complete |
| **Swagger Compliance** | PASS — Makefile, Spectral rules functional after linter corrections |
| **Build State Model** | FAIL — Ambiguous dual model (16 stages vs 11 statuses) |

---

## 1. Design Drift Analysis

### 1.1 .spectral.yaml — Linter-Driven Simplification (ACCEPTED)

The `.spectral.yaml` was correctly simplified from the original StratoGroup-custom ruleset to a minimal OAS baseline. This is the **right call** for the current phase:

**Original design:**
- Custom rules for `operation-summary`, `operation-error-responses`, `operation-security`, etc.
- `fail-severity: warn` (strict mode)

**Current state:**
- Extends `spectral:oas` baseline only
- Disables rules irrelevant to OAS3 output from grpc-gateway: `oas2-api-host`, `oas2-api-schemes`, `oas2-unused-definition`
- Disables `info-contact`, `info-description` (cosmetic; re-add in M3)
- Disables `no-$ref-siblings` (grpc-gateway generates these legitimately)

**Verdict:** ACCEPTED. The custom rules (operation-error-responses, security enforcement, etc.) should be **re-added as custom Spectral functions** in M2 once the OpenAPI spec is stabilised and generating cleanly. ADR-013 should be created to document this phased approach.

### 1.2 buf.yaml — Lint Exception Expansion (ACCEPTED WITH CONDITIONS)

The `buf.yaml` lint exceptions expanded from 1 to 4:

| Exception | Reason | Verdict |
|-----------|--------|---------|
| `PACKAGE_VERSION_SUFFIX` | Original. We use `sdx.v1` not `sdx.v1beta1`. | ACCEPTED |
| `RPC_REQUEST_STANDARD_NAME` | Our RPCs reuse messages across gateway + domain services (e.g., `SubmitWorkItemRequest` used by both `IngestionService` and `SDXGatewayService`). | ACCEPTED — this is an intentional design choice to avoid message duplication |
| `RPC_RESPONSE_STANDARD_NAME` | Same as above. | ACCEPTED |
| `RPC_REQUEST_RESPONSE_UNIQUE` | Allows same request/response type in multiple RPCs. Essential for gateway service pattern. | ACCEPTED |
| `IMPORT_USED` | Suppresses warnings about imports. This **masks real issues**. | CONDITIONAL — remove once all proto files are fully fleshed out |

**Condition:** `IMPORT_USED` exception must be removed by M2. It currently hides the fact that several proto files import types they don't yet use (because those protos are incomplete). Once all services are fully defined, this lint rule becomes a safety net against dead imports.

### 1.3 Gateway Proto — Reuse Pattern (CORRECT)

`gateway.proto` correctly imports and reuses request/response types from `workitem.proto`, `build.proto`, and `artifact.proto` rather than redefining them. This is the intended Facade pattern — the Gateway delegates to domain services while presenting a unified API surface. No drift.

---

## 2. Critical Must-Fix Items (Blocks Codex)

### MF-01: Event Proto Payload Gap [CRITICAL]

**Problem:** `domain-events.md` defines 60+ domain events across 10 categories. `events.proto` only defines 10 payload messages:
- `BuildCompletedPayload`, `BuildFailedPayload`, `GateFailedPayload`
- `ArtifactSignedPayload`, `ComplianceCertifiedPayload`
- `StageCompletedPayload`, `StageFailedPayload`
- `AgentUnhealthyPayload`, `ModelFallbackPayload`, `QuotaExhaustedPayload`

**Missing (40+ payloads):**
- ALL WorkItem event payloads (6 events)
- `BuildInitiatedPayload`, `BuildCheckpointedPayload`, `BuildRecoveryStartedPayload`, `BuildRecoverySucceededPayload`
- `GateEvaluatedPayload`, `GatePassedPayload`
- `RemediationStartedPayload`, `RemediationSucceededPayload`, `RemediationExhaustedPayload`
- `ArtifactCreatedPayload`, `ArtifactPublishedPayload`, `ArtifactDeployedPayload`, `ArtifactRevokedPayload`
- ALL Tenant event payloads (9 events)
- ALL Agent event payloads except 2 (6 events)
- ALL Security event payloads (4 events)
- ALL Compliance event payloads except 1 (3 events)
- ALL Deployment event payloads (7 events)
- ALL LLM event payloads (3 events)
- ALL Webhook delivery event payloads (4 events)

**Impact:** Codex cannot generate type-safe event publishers/subscribers without payload definitions. Without proto payloads, events will be serialised as untyped `map<string, string>` or raw JSON — violating Holy Trinity correctness guarantees.

**Fix:** Add all missing payload messages to `events.proto`. Each payload must mirror the fields specified in `domain-events.md`.

**Effort:** ~200 lines of proto additions.

---

### MF-02: Missing Internal Service Protos [CRITICAL]

**Problem:** 12 of 20 bounded contexts have no gRPC service proto definitions:

| BC | Owner Service | Proto Status |
|----|---------------|-------------|
| BC-02 Domain Modelling | `sdx-domain-svc` | MISSING |
| BC-03 Architecture Design | `sdx-architect-svc` | MISSING |
| BC-04 Contract Generation | `sdx-contract-svc` | MISSING |
| BC-05 Test Specification | `sdx-test-spec-svc` | MISSING |
| BC-06 Code Generation | `sdx-codegen-svc` | MISSING |
| BC-08 Security Scanning | `sdx-security-svc` | MISSING |
| BC-10 Performance Testing | `sdx-performance-svc` | MISSING |
| BC-11 Documentation Gen | `sdx-docs-svc` | MISSING |
| BC-12 Infrastructure Prov | `sdx-iac-svc` | MISSING |
| BC-15 Learning & Opt | `sdx-learning-svc` | MISSING |
| BC-18 Observability | External (Grafana) | N/A — correct |
| BC-19 Dashboard | `sdx-dashboard` | N/A — read-only UI |

**Design Decision Required:** These services are **internal** (invoked only by Temporal activities, not exposed via sdx-gateway). Two approaches:

**Option A: Full proto definitions for all 12 services** (~12 new proto files)
- Pro: Codex generates typed gRPC stubs for all inter-service calls
- Con: High upfront cost for services that are purely Temporal-activity-driven

**Option B: Internal services use Temporal activity interfaces only** (~0 new protos)
- Pro: Simpler; Temporal activities are Go function signatures, not gRPC
- Con: No contract enforcement between services; relies on convention

**Recommendation: Option B for M1, Option A for M2.** Create ADR-013 documenting this decision. For M1, internal services communicate via Temporal activity signatures (Go interfaces defined in `domain/*/service.go`). For M2, add gRPC interfaces for services that need independent scaling or cross-language support.

**Fix for M1:** Add a new document `docs/domain/internal-service-interfaces.md` defining Go function signatures for all 12 internal services as Temporal activity contracts.

---

### MF-03: Build State Machine Ambiguity [CRITICAL]

**Problem:** The Build aggregate has two overlapping models:
1. **Stage-based** (use-cases.md): Build progresses through stages 1-16 sequentially
2. **Status-based** (state-machines.md, aggregates.md): Build transitions through named statuses (INITIATED → DOMAIN_MODELLING → ... → COMPLETED)

These are not the same:
- Stage 7 = Static Analysis, Stage 8 = Test Execution, Stage 9 = Security Scanning — but all three are collapsed into status `VERIFYING`
- Stage 10 = Compliance Mapping — but the status is `CERTIFYING`
- Stages 12-13 = Documentation + IaC — but these have no named status

**Impact:** Codex doesn't know whether to check `build.current_stage == 7` or `build.status == VERIFYING` for gate evaluation. The two models can diverge.

**Fix:** Formalise the Build as a **dual-track model**:
- `status` (BuildStatus enum): The coarse-grained state visible to the API and dashboard (11 values)
- `current_stage` (StageNumber 1-16): The fine-grained pipeline position tracked internally
- Define a **deterministic mapping**: `stage_to_status(stage_number) → BuildStatus`

```
Mapping:
  Stage 1  (Ingestion)          → INITIATED
  Stage 2  (DomainModelling)    → DOMAIN_MODELLING
  Stage 3  (ArchitectureDesign) → ARCHITECTING
  Stage 4  (ContractGeneration) → CONTRACTING
  Stage 5  (TestSpecification)  → TESTING_RED
  Stage 6  (CodeGeneration)     → CODING_GREEN
  Stage 7  (StaticAnalysis)     → VERIFYING
  Stage 8  (TestExecution)      → VERIFYING
  Stage 9  (SecurityScanning)   → VERIFYING
  Stage 10 (ComplianceMapping)  → CERTIFYING
  Stage 11 (PerformanceTesting) → CERTIFYING
  Stage 12 (Documentation)      → CERTIFYING
  Stage 13 (InfraAsCode)        → CERTIFYING
  Stage 14 (ArtifactSigning)    → SIGNING
  Stage 15 (Deployment)         → DEPLOYING
  Stage 16 (LearningLoop)       → DEPLOYING (final activity before COMPLETED)
```

Add this mapping to `state-machines.md` and `aggregates.md`. The invariant becomes: `build.status == stage_to_status(build.current_stage)` at all times.

---

### MF-04: Missing Repository Interfaces [CRITICAL]

**Problem:** Three repositories required by use-cases are not defined in `repository-interfaces.md`:

| Missing Repo | Required By | Purpose |
|-------------|-------------|---------|
| `ContractRepository` | UC-04 (Pipeline Stage 4) | Persist/load generated API contracts |
| `ModelRoutingConfigRepository` | UC-15 (Route LLM Request) | Lookup agent model routing preferences |
| `LLMRequestRepository` | UC-15, Learning (BC-15) | Persist LLM request/response for cost tracking and learning |

**Fix:** Add these three interfaces to `repository-interfaces.md`:

```go
// ContractRepository
type Repository interface {
    Save(ctx context.Context, contract *Contract) error
    FindByID(ctx context.Context, id ContractID) (*Contract, error)
    FindByBlueprint(ctx context.Context, blueprintID BlueprintID) ([]*Contract, error)
    FindByVersion(ctx context.Context, blueprintID BlueprintID, version string) (*Contract, error)
}

// ModelRoutingConfigRepository
type Repository interface {
    Save(ctx context.Context, config *ModelRoutingConfig) error
    FindByAgentType(ctx context.Context, agentType AgentType) (*ModelRoutingConfig, error)
    FindDefault() (*ModelRoutingConfig, error)
}

// LLMRequestRepository (ClickHouse — append-only)
type Repository interface {
    Append(ctx context.Context, record *LLMRequestRecord) error
    GetCostByTenant(ctx context.Context, tenantID TenantID, period DateRange) (*CostReport, error)
    GetCostByBuild(ctx context.Context, buildID BuildID) (*CostReport, error)
    GetCostByModel(ctx context.Context, model ModelID, period DateRange) (*ModelCostBreakdown, error)
}
```

---

## 3. Post-Merge Refactors (Non-Blocking)

### PMR-01: Restore StratoGroup Spectral Custom Rules [M2]

The `.spectral.yaml` was correctly simplified for grpc-gateway compatibility. After M1 OpenAPI spec stabilises:

1. Re-add `operation-summary` rule (every endpoint must have a summary)
2. Re-add `operation-security` rule (every endpoint must define security scheme)
3. Re-add `operation-error-responses` rule (every endpoint must define 400/401/500)
4. Implement as **custom Spectral functions** that understand grpc-gateway output format
5. Set `fail-severity: warn` in CI

### PMR-02: Remove `IMPORT_USED` buf Lint Exception [M2]

Once all proto files are fully fleshed out (all 12 internal services defined if Option A chosen), remove the `IMPORT_USED` exception from `buf.yaml`. This re-enables detection of dead imports.

### PMR-03: Add Money and CurrencyCode to common.proto [M2]

`value-objects.md` defines `Money` and `CurrencyCode` but they're not in the proto. If billing or cost APIs are exposed externally (they will be for Enterprise/Sovereign tenants), these need proto definitions:

```proto
enum CurrencyCode {
  CURRENCY_CODE_UNSPECIFIED = 0;
  CURRENCY_CODE_USD = 1;
  CURRENCY_CODE_EUR = 2;
  CURRENCY_CODE_GBP = 3;
  CURRENCY_CODE_NGN = 4;
  CURRENCY_CODE_SGD = 5;
  CURRENCY_CODE_BRL = 6;
}

message Money {
  int64 amount_minor_units = 1;  // Cents, pence, kobo, etc.
  CurrencyCode currency = 2;
}
```

### PMR-04: Add Detailed Metrics Proto Messages [M2]

`quality.proto` defines `CodeBLEUReport`, `CoverageReport`, `ComplexityReport` — these are correct. But `value-objects.md` also defines `MutationScore` not present in proto. Add:

```proto
message MutationScoreReport {
  double score_pct = 1;
  uint64 mutants_killed = 2;
  uint64 mutants_total = 3;
  uint64 mutants_survived = 4;
}
```

### PMR-05: Edge Case → Policy Cross-Reference Index [M2]

14 edge cases reference handling strategies not formally linked to policies. Add a cross-reference table to `policies-validations.md`:

| Edge Case | Governing Policy | Additional Handling |
|-----------|------------------|-------------------|
| EC-01 LLM Empty Response | Model Fallback Policy (#2) | Add retry-with-same-model sub-policy |
| EC-03 Circular Remediation | Remediation Policy (#1) | Add cycle detection invariant |
| EC-04 Checkpoint Corruption | Checkpoint Policy (#3) | Add fallback-to-previous-5 sub-policy |
| EC-07 Build 24h Timeout | (NEW) Timeout Policy | Define per-stage and per-build timeouts |
| EC-08 NATS Unavailable | (NEW) WAL Buffer Policy | Define local WAL semantics |
| EC-19 Fleet Stuck | (NEW) Deployment Timeout Policy | 10-min threshold → rollback |
| EC-21 Rollback Chain | (NEW) Deployment History Policy | Walk-back algorithm |

### PMR-06: WebhookConfigRepository Missing from repository-interfaces.md [M1-POST]

`repository-interfaces.md` defines `WebhookConfigRepository` — confirmed present. No action needed.

---

## 4. ADR Updates

### ADR-013: Internal Service Communication Strategy (NEW)

**Status:** Proposed
**Context:** 12 of 20 bounded contexts are internal (not exposed via sdx-gateway). The question is whether they need gRPC service definitions.
**Decision:** M1 uses Temporal activity interfaces (Go function signatures). M2 adds gRPC proto definitions for services that need independent scaling.
**Consequences:** M1 is faster to implement. Internal services cannot be called outside Temporal workflows in M1. M2 migration is additive (add proto, don't remove Go interfaces).

### ADR-014: Build State Model — Dual-Track (Stage + Status) (NEW)

**Status:** Proposed
**Context:** Build has both a 16-stage pipeline and an 11-status state machine.
**Decision:** Both are first-class. `current_stage` (1-16) is the source of truth. `status` is derived via deterministic `stage_to_status()` mapping. Status is exposed on API; stage is exposed on API and used internally.
**Consequences:** No ambiguity for Codex. Status changes are always a function of stage progression. Remediation loops and recovery do not change status directly — they revert `current_stage` and status follows.

### ADR-015: Spectral Phased Enforcement (NEW)

**Status:** Proposed
**Context:** grpc-gateway generates OpenAPI specs that don't comply with all StratoGroup custom Spectral rules (e.g., missing `info.contact`, `$ref` siblings).
**Decision:** M1 uses minimal OAS baseline. M2 re-adds custom rules as Spectral functions that understand grpc-gateway output.
**Consequences:** M1 has weaker API lint coverage. Compensated by `buf lint` on the proto side. M2 achieves full spec compliance.

### ADR-016: Event Proto Payload Coverage (UPDATE to existing event strategy)

**Status:** Proposed
**Context:** Only 10 of 60+ domain events have proto payload definitions.
**Decision:** All domain events MUST have typed proto payload messages. Events without payloads cannot be published.
**Consequences:** `events.proto` grows significantly (~200 lines). Type safety is guaranteed across NATS and Redpanda transports. Codex generates typed publishers/subscribers.

---

## 5. Merge Guidance

### Pre-Merge Checklist

| # | Item | Status | Action |
|---|------|--------|--------|
| 1 | Proto syntax validity | PASS | No action |
| 2 | Proto enum 0=UNSPECIFIED | PASS | No action |
| 3 | Event payload coverage | FAIL | Fix MF-01 before merge |
| 4 | Internal service contracts | FAIL | Fix MF-02 (add doc) before merge |
| 5 | Build state machine clarity | FAIL | Fix MF-03 (add mapping) before merge |
| 6 | Repository interface coverage | FAIL | Fix MF-04 (add 3 repos) before merge |
| 7 | Error code ↔ use-case mapping | PASS | No action |
| 8 | Value object ↔ proto mapping | PASS (advisory) | PMR-03 for M2 |
| 9 | buf.yaml lint config | PASS | Conditional (remove IMPORT_USED in M2) |
| 10 | Spectral config | PASS | Re-add custom rules in M2 |

### Merge Decision

**DO NOT merge to `main` until MF-01 through MF-04 are resolved.**

After fixes are applied:
1. Run `buf lint` — must pass with zero errors
2. Run `buf build` — must compile all proto files
3. Verify cross-references manually (event payloads ↔ domain-events.md)
4. Commit fixes as a separate commit on `main`
5. Tag as `v0.1.0-architecture` (pre-implementation baseline)

---

## 6. Next-Iteration Backlog

### M1: Foundation (Current — Pre-Codex)

| # | Task | Priority | Effort | Owner |
|---|------|----------|--------|-------|
| M1-01 | Fix MF-01: Add 40+ event payload messages to events.proto | P0 | 4h | Architect |
| M1-02 | Fix MF-02: Create internal-service-interfaces.md (Temporal activity contracts) | P0 | 3h | Architect |
| M1-03 | Fix MF-03: Add stage-to-status mapping to state-machines.md and aggregates.md | P0 | 1h | Architect |
| M1-04 | Fix MF-04: Add ContractRepository, ModelRoutingConfigRepository, LLMRequestRepository | P0 | 1h | Architect |
| M1-05 | Create ADR-013, ADR-014, ADR-015, ADR-016 as formal docs | P1 | 2h | Architect |
| M1-06 | Hand off to Codex: domain layer implementation (aggregates, VOs, events) | P0 | Codex | Builder |
| M1-07 | Hand off to Codex: proto compilation and stub generation | P0 | Codex | Builder |
| M1-08 | Antigravity verification of domain layer | P0 | Auto | Verifier |

### M2: API & Integration (Post-Codex Domain Layer)

| # | Task | Priority | Effort |
|---|------|----------|--------|
| M2-01 | Implement sdx-gateway gRPC service + REST proxy | P0 | Codex |
| M2-02 | Implement sdx-ingestion-svc (WorkItem aggregate + parsing) | P0 | Codex |
| M2-03 | Implement sdx-tenant-svc (Tenant aggregate + provisioning) | P0 | Codex |
| M2-04 | Implement sdx-artifact-svc (signing, provenance, SBOM) | P0 | Codex |
| M2-05 | Implement sdx-quality-svc (gate evaluation) | P0 | Codex |
| M2-06 | Implement sdx-llm-gateway (routing, circuit breaker) | P0 | Codex |
| M2-07 | Restore Spectral custom rules (PMR-01) | P1 | Architect |
| M2-08 | Remove IMPORT_USED buf exception (PMR-02) | P1 | Architect |
| M2-09 | Add Money/CurrencyCode to proto (PMR-03) | P1 | Architect |
| M2-10 | Add MutationScoreReport to proto (PMR-04) | P2 | Architect |
| M2-11 | Contract tests: `make contract-test` green | P0 | Codex |
| M2-12 | `make api-compliance-full` green in CI | P0 | DevOps |

### M3: Orchestration & Deployment

| # | Task | Priority |
|---|------|----------|
| M3-01 | Implement Temporal Build Pipeline Workflow | P0 |
| M3-02 | Implement sdx-agent-orchestrator (agent mesh, health, routing) | P0 |
| M3-03 | Implement sdx-deploy-svc (Fleet GitOps, smoke tests, SLO) | P0 |
| M3-04 | Implement sdx-compliance-svc (evidence generation) | P0 |
| M3-05 | Implement sdx-security-svc (SAST, DAST, SCA, secrets, SBOM) | P0 |
| M3-06 | Implement sdx-audit-svc (hash-chained audit log) | P0 |
| M3-07 | NATS JetStream stream provisioning (10 streams) | P0 |
| M3-08 | Redpanda inter-regional replication setup | P1 |
| M3-09 | End-to-end pipeline test: submit PRD → deployed artifact | P0 |

### M4: Dashboard & Observability

| # | Task | Priority |
|---|------|----------|
| M4-01 | Implement sdx-dashboard (React + Connect-ES) | P0 |
| M4-02 | Implement webhook delivery system | P1 |
| M4-03 | Grafana dashboards (build metrics, agent health, gate pass rates) | P1 |
| M4-04 | Performance testing (k6 load tests) | P1 |
| M4-05 | Chaos testing (Litmus: agent crash, NATS partition) | P2 |

### M5-M8: Hardening, Compliance, Scale, Launch

| Milestone | Focus |
|-----------|-------|
| M5 | Security hardening: SPIFFE rotation, Vault dynamic creds, pen testing |
| M6 | Compliance certification: SOC2 evidence automation, GDPR data residency validation |
| M7 | Multi-region: Lagos, London, Ashburn, Singapore, Sao Paulo deployment |
| M8 | GA Launch: billing integration, SDK generation, public API documentation |

---

## 7. StratoGroup Factory Principle Compliance

| Principle | Status | Evidence |
|-----------|--------|----------|
| **XP (Iteration)** | COMPLIANT | Milestones M1-M8 with clear iteration boundaries |
| **TDD (Correctness)** | COMPLIANT | Edge cases define test names; red-green cycle enforced by pipeline |
| **DDD (Domain)** | COMPLIANT | 20 bounded contexts, 9 aggregates, ubiquitous language, context map |
| **AIDD (AI-first)** | COMPLIANT | Architect/Builder/Verifier separation maintained |
| **Holy Trinity + AIDD** | COMPLIANT | All four pillars present in design artifacts |
| **Swagger/OpenAPI as First-Class** | COMPLIANT | Makefile with 10 targets; Spectral linting; CI gate (`make api-compliance-full`) |
| **Zero Software Tax** | COMPLIANT | License policy documented; gate G-09 enforces |
| **SLSA Level 3** | COMPLIANT | Provenance, Cosign, Rekor in artifact aggregate |
| **Multi-tenancy** | COMPLIANT | 5-layer tenant propagation; quota enforcement; data residency |
| **Observability** | COMPLIANT | OTel integration; OpenObserve + ClickHouse audit; correlation IDs |

---

## 8. Final Statement

The NeuroForge SDX architecture package is **90% Codex-ready**. The remaining 10% is four focused fixes (MF-01 through MF-04) that can be completed in a single work session. No architectural redesign is needed — only gap-filling.

The design is decisive, production-grade, and aligned with StratoGroup factory principles. Swagger/OpenAPI compliance is a permanent first-class feature. No shortcuts that weaken long-term extensibility were detected.

**Next action:** Resolve MF-01 through MF-04, commit as a fix commit, tag `v0.1.0-architecture`, and hand off to Codex.
