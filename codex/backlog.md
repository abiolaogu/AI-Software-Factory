# NeuroForge SDX — Codex Implementation Backlog

> Milestones M1-M8. Each task has ID, description, acceptance criteria, dependencies, and effort estimate.
> Codex processes tasks in order within each milestone. No skipping.

---

## M1: Domain Foundation

**Goal:** Implement the entire DDD domain layer in Go. All aggregates, value objects, domain events, repository interfaces, state machines, and policies. 100% unit test coverage of invariants and state transitions.

**Exit Criteria:** `go test ./domain/...` passes. All 50+ invariants enforced. All 7 state machines implemented.

| ID | Task | Acceptance Criteria | Deps | Est |
|----|------|---------------------|------|-----|
| M1-001 | **Initialise Go module and project scaffold** | `go mod init github.com/stratogroup/neuroforge-sdx` succeeds. Directory structure matches `factory.yaml`. `.golangci.yml` configured. `Makefile` has `test`, `lint`, `build` targets. | — | S |
| M1-002 | **Implement `domain/shared` package** | ULID type with `New()`, `Parse()`, `String()`, `IsZero()`. ContentHash with SHA-256 validation. TenantContext with extraction from `context.Context`. Pagination types. DomainEventEnvelope. All error sentinels (`ErrNotFound`, `ErrConflict`, `ErrInvariantViolated`, `ErrQuotaExhausted`, `ErrImmutable`, `ErrTenantMismatch`). 100% test coverage. | M1-001 | M |
| M1-003 | **Implement all value objects from `value-objects.md`** | Every VO has constructor with validation, equality method, and `String()`. Money arithmetic (same-currency only). NexoRegion, PlanTier, LanguageCode, ComplianceFramework enums with `String()` and `Parse()`. ModelID enum. GateResult VO. TokenUsage VO. All VOs are immutable (no setters). Tests verify validation rejects invalid inputs. | M1-002 | L |
| M1-004 | **Implement WorkItem aggregate** | WorkItem struct with all fields from `aggregates.md`. State machine from `state-machines.md` enforced: DRAFT→PARSED→MANIFEST_REVIEW→APPROVED→MODELLING→COMPLETED, with FAILED. All 8 invariants (WI-INV-01..WI-INV-08) enforced. Domain events emitted on every state transition. Tests for each invariant violation AND each valid transition. | M1-003 | L |
| M1-005 | **Implement Build aggregate** | Build struct with all fields. Dual-track state model (stage 1-16 + derived BuildStatus per ADR-014). `stage_to_status()` deterministic mapping. All 12 invariants (B-INV-01..B-INV-12). PipelineStageRecord and RemediationRecord entities. `AdvanceStage()`, `RecordGateResult()`, `StartRemediation()`, `Checkpoint()`, `RecoverFromCheckpoint()`, `Fail()`, `Complete()`. Domain events emitted. Tests: each invariant, each transition, remediation loop limit (3), recovery limit (5), non-remediable gates (G-07, G-14). | M1-003 | XL |
| M1-006 | **Implement Artifact aggregate** | Artifact struct. State machine: CREATED→SIGNED→PUBLISHED→DEPLOYED, with REVOKED. All 8 invariants (A-INV-01..A-INV-08). `Sign()` requires valid SignatureRef+ProvenanceRef+SBOMRef. `Publish()` requires SIGNED. Content hash immutable after creation. Write-once signature. Domain events emitted. Tests: each invariant, REVOKED is terminal, content-hash immutability. | M1-003 | M |
| M1-007 | **Implement Tenant aggregate** | Tenant struct with Workspace, WorkspaceMember, ComplianceProfile entities. State machine: PROVISIONING→ACTIVE→SUSPENDED→DEPROVISIONED. All 10 invariants (T-INV-01..T-INV-10). Quota enforcement (`ConsumeQuota()`, `ResetBillingPeriod()`). Plan downgrade validation. Language access per plan. Domain events emitted. Tests: quota exhaustion, plan downgrade rejection, SOVEREIGN on-prem enforcement, STARTER language restriction. | M1-003 | L |
| M1-008 | **Implement Agent aggregate** | Agent struct. State machine: REGISTERED→IDLE→ASSIGNED→PROCESSING→RECOVERING→UNHEALTHY→TERMINATED. All 8 invariants (AG-INV-01..AG-INV-08). Heartbeat tracking. Model routing config. SPIFFE ID validation. Domain events emitted. Tests: missed heartbeat → UNHEALTHY, SVID expiry → TERMINATED, single-build assignment. | M1-003 | M |
| M1-009 | **Implement Deployment aggregate** | Deployment struct. State machine: PENDING→APPLYING→SMOKE_TESTING→SLO_VALIDATING→DEPLOYED, with ROLLED_BACK and FAILED. All 5 invariants (D-INV-01..D-INV-05). SmokeTestResult, SLOValidation entities. Auto-rollback on SLO violation. Tests: smoke failure→rollback, SLO violation→rollback, only one active deployment per service+env. | M1-003 | M |
| M1-010 | **Implement DomainModel aggregate** | DomainModel struct. Status: DRAFT→MODELLING→READY→APPROVED. All 5 invariants (DM-INV-01..DM-INV-05). Version monotonic increment. Completeness score ≥95.0 for APPROVED. | M1-003 | S |
| M1-011 | **Implement Blueprint aggregate** | Blueprint struct. Status: DRAFT→DESIGNED→APPROVED. All 5 invariants (BP-INV-01..BP-INV-05). ServiceDefinition uniqueness. Tech stack within tenant allowance. | M1-003 | S |
| M1-012 | **Implement Contract aggregate** | Contract struct. Status: DRAFT→VALIDATED→APPROVED. All 4 invariants (C-INV-01..C-INV-04). Spec hash integrity. APPROVED is immutable. Breaking change detection delegated to oasdiff. | M1-003 | S |
| M1-013 | **Implement QualityReport, SecurityReport, CompliancePackage domain objects** | QualityReport with GateResult collection. SecurityReport with finding counts. CompliancePackage with FrameworkResult and ControlGap. All are append-only (created once, never modified). | M1-003 | M |
| M1-014 | **Implement all repository interfaces as Go interfaces** | One `repository.go` file per aggregate package. Exact method signatures from `repository-interfaces.md`. Including the 3 new repos: ContractRepository, ModelRoutingConfigRepository, LLMRequestRepository. No implementations yet (adapters come in M2). | M1-004..M1-013 | S |
| M1-015 | **Implement policy functions** | Remediation policy (gate-to-agent mapping, 3-attempt limit). Model fallback policy (circuit breaker state transitions). Quota enforcement policy. Zero Software Tax license check. Secret scan pattern matching. Escalation policy (5-recovery threshold). Data residency validation. Idempotency key validation. Webhook delivery retry schedule. Each policy is a pure function with tests. | M1-004..M1-013 | L |
| M1-016 | **Implement edge case test suite** | One test per edge case from `edge-cases.md` (28 tests). Tests use aggregate methods directly (no adapters). Test names match EC-XX format. Tests verify both detection and handling. Circular remediation detection test (EC-03). Checkpoint corruption fallback test (EC-04). Plan downgrade rejection test (EC-11). | M1-015 | L |
| M1-017 | **Proto compilation and stub generation** | `buf lint` passes. `buf build` passes. `buf generate` produces Go, TypeScript, Python stubs in `gen/`. All 13 proto files compile. Generated code committed to `gen/`. | M1-001 | M |

---

## M2: Infrastructure Adapters & API Layer

**Goal:** Implement all database adapters, messaging adapters, and the public-facing API gateway. All services can receive requests and persist data.

**Exit Criteria:** `make contract-test` passes. `make api-compliance-full` passes. All CRUD operations work against real databases (testcontainers).

| ID | Task | Acceptance Criteria | Deps | Est |
|----|------|---------------------|------|-----|
| M2-001 | **YugabyteDB adapter: WorkItem repository** | Implements WorkItemRepository interface. SQL migrations via Atlas HCL. All queries include `WHERE tenant_id = $1`. Testcontainers integration test: Save, FindByID, FindByTenant with filters, CountByTenant, soft Delete. Pagination verified (cursor-based). | M1-014, M1-017 | L |
| M2-002 | **YugabyteDB adapter: Build repository** | Implements BuildRepository. FindStaleCheckpoints returns builds with old checkpoints. CountActiveByTenant for quota. Optimistic locking via version column. Integration test covers concurrent write conflict (EC-18). | M1-014 | L |
| M2-003 | **YugabyteDB adapter: Artifact, Tenant, Agent, Deployment, Contract, Blueprint, DomainModel repos** | All remaining YugabyteDB repositories implemented. Each has migrations, testcontainers tests, tenant-scoped queries. | M1-014 | XL |
| M2-004 | **ClickHouse adapter: QualityReport, AuditLog, LLMRequest repos** | Append-only inserts. AuditLog with hash-chain verification query. LLMRequest cost aggregation queries. QualityReport gate metrics aggregation. Integration tests via testcontainers. | M1-014 | L |
| M2-005 | **DragonflyDB adapter: Rate limiter, Idempotency cache, Agent cache** | Sliding window rate limiter (per-tenant). Idempotency cache with 24h TTL. Agent health cache for fast dashboard reads. Integration tests. | M1-014 | M |
| M2-006 | **RustFS adapter: Artifact store, Evidence store** | S3-compatible client. Content-addressed storage (SHA-256 key). Pre-signed URL generation (1h TTL). Tenant-scoped paths (`/{tenant_id}/builds/{build_id}/`). Integration test with MinIO testcontainer. | M1-014 | M |
| M2-007 | **NATS JetStream adapter: Publisher, Subscriber, Stream config** | DomainEvent envelope serialisation. Publish with `Nats-Msg-Id` for dedup. Subscribe with durable consumer. WAL buffer for NATS outage (EC-08). 10 streams provisioned per `domain-events.md` config. Integration test with NATS testcontainer. | M1-014, M1-017 | L |
| M2-008 | **Implement sdx-gateway service** | gRPC server with grpc-gateway REST proxy. Authentik JWT validation middleware. Tenant extraction interceptor. Rate limiting interceptor. Idempotency interceptor. Request ID generation (ULID). OpenTelemetry tracing interceptor. Health check endpoint. Serves `/swagger`, `/docs`, `/openapi.json`. All 17 SDXGatewayService RPCs routed to domain services. Integration test: submit WorkItem → get WorkItem. | M2-001..M2-007 | XL |
| M2-009 | **Implement sdx-ingestion-svc** | gRPC IngestionService (7 RPCs). Application service layer calls WorkItem aggregate methods. Publishes domain events via NATS. Integration test: full WorkItem lifecycle (create → parse → resolve ambiguity → approve). | M2-001, M2-007 | L |
| M2-010 | **Implement sdx-tenant-svc** | gRPC TenantService (13 RPCs). Tenant provisioning (K8s namespace, Vault mount stubs). Quota enforcement. Plan changes. Workspace management. Integration test: create tenant → add workspace → add member → check quota. | M2-003, M2-007 | L |
| M2-011 | **Implement sdx-artifact-svc** | gRPC ArtifactService (9 RPCs). Cosign signing (stubbed in M2, real in M5). Provenance generation. SBOM association. Pre-signed download URLs. Integration test: create artifact → sign → publish → verify → download. | M2-003, M2-006 | L |
| M2-012 | **Implement sdx-quality-svc** | gRPC QualityService (6 RPCs). Gate evaluation logic for all 14 gates. Remediation trigger. Gate metrics aggregation. Integration test: evaluate gate → pass/fail → trigger remediation. | M2-004 | L |
| M2-013 | **Implement sdx-llm-gateway** | gRPC LLMService (5 RPCs). Circuit breaker (CLOSED→OPEN→HALF_OPEN). Model routing (primary→fallback→on-prem). Token usage tracking. Cost reporting. Integration test: complete → fallback on failure → circuit open. | M2-004, M2-005 | L |
| M2-014 | **Implement sdx-audit-svc** | gRPC AuditService (4 RPCs). Hash-chained append. Integrity verification. Evidence export. Integration test: append 100 entries → verify chain → break one → detect. | M2-004 | M |
| M2-015 | **Implement sdx-agent-orchestrator** | gRPC AgentService (9 RPCs). Agent registration, assignment, heartbeat, health monitoring. Integration test: register → assign → heartbeat → miss heartbeats → UNHEALTHY. | M2-003, M2-005 | L |
| M2-016 | **Implement sdx-deploy-svc** | gRPC DeploymentService (5 RPCs). Fleet GitOps stub (real integration in M3). Smoke test execution. SLO validation stub. Auto-rollback. Integration test: deploy → smoke pass → SLO pass → DEPLOYED. | M2-003 | M |
| M2-017 | **Contract tests and API compliance** | `make proto` passes. `make openapi` passes. `make contract-test` passes (all endpoints match spec). `make compliance-test` passes (all endpoints have auth, errors, rate-limit metadata). `make breaking-check` passes (no breaking changes vs v0.1.0). | M2-008..M2-016 | L |
| M2-018 | **Add Money and CurrencyCode to common.proto** | Proto definitions per PMR-03 from architecture review. Regenerate stubs. No breaking changes. | M1-017 | S |

---

## M3: Workflow Orchestration

**Goal:** Implement the Temporal Build Pipeline Workflow that orchestrates all 16 stages, quality gates, remediation loops, checkpointing, and recovery.

**Exit Criteria:** A complete build can run from WorkItem submission through all 16 stages to artifact signing. Gate failures trigger remediation. Recovery works from checkpoints.

| ID | Task | Acceptance Criteria | Deps | Est |
|----|------|---------------------|------|-----|
| M3-001 | **Implement BuildPipelineWorkflow** | Temporal workflow that sequences 16 stages. Each stage is a Temporal activity. Heartbeat every 30s (checkpoint). Status derived from stage via `stage_to_status()`. Workflow ID = BuildID. | M2-008 | XL |
| M3-002 | **Implement RemediationSubWorkflow** | Triggered on gate failure. Assigns responsible agent (per gate-to-agent mapping). Re-evaluates gate. Max 3 attempts. Emits remediation events. Returns to main workflow on success or escalation. | M3-001 | L |
| M3-003 | **Implement RecoveryWorkflow** | Triggered via Temporal signal on build failure. Loads checkpoint. Verifies hash. Resumes from checkpoint stage. Max 5 recovery attempts. Emits recovery events. | M3-001 | M |
| M3-004 | **Implement all 16 stage activities** | Each activity implements the corresponding interface from `internal-service-interfaces.md`. Activities call LLM gateway for AI-powered stages. Produce artifacts stored in RustFS. Emit stage events. | M3-001, M2-013 | XL |
| M3-005 | **Implement NATS → Redpanda bridge for inter-regional events** | `sdx.build.metrics`, `sdx.audit.events`, `sdx.compliance.evidence` topics replicated. Redpanda producer with idempotent writes. Consumer for learning service. | M2-007 | M |
| M3-006 | **Implement webhook delivery system** | Subscribe to webhook-eligible events. Dispatch HTTP POST with HMAC-SHA256. Retry with exponential backoff (5 attempts). Dead letter after exhaustion. | M2-007, M2-008 | M |
| M3-007 | **End-to-end pipeline test** | Submit a PRD WorkItem → all 16 stages execute (with LLM stubs) → gates evaluate → artifact signed → events emitted → audit log populated. Full test using Temporal test framework. | M3-001..M3-006 | XL |

---

## M4: Dashboard & Observability

**Goal:** Build the real-time web dashboard and observability stack.

| ID | Task | Acceptance Criteria | Deps | Est |
|----|------|---------------------|------|-----|
| M4-001 | **Implement sdx-dashboard (React + Connect-ES)** | Real-time build status. Agent mesh view. Gate results. Artifact lineage graph. Tenant quota usage. WebSocket for live updates. | M2-008 | XL |
| M4-002 | **Implement Grafana dashboards** | Build pipeline metrics. Agent utilisation. Gate pass rates. LLM cost tracking. SLO compliance. Alert rules for escalation. | M2-014 | L |
| M4-003 | **Implement OpenTelemetry instrumentation** | All services emit traces, metrics, logs. Build correlation ID (build_id + trace_id) threaded. Exemplars on metrics. | M2-008 | L |
| M4-004 | **Performance testing (k6)** | API gateway: 1000 RPS sustained. Build submission: 100 concurrent. Gate evaluation: p99 < 200ms. Results documented. | M3-007 | L |

---

## M5: Security Hardening

| ID | Task | Acceptance Criteria | Deps | Est |
|----|------|---------------------|------|-----|
| M5-001 | **SPIFFE/SPIRE integration** | All agents get SVID. mTLS between all services. SVID rotation every 1h. Expired SVID → agent TERMINATED. | M3-004 | L |
| M5-002 | **Vault dynamic credentials** | Database credentials rotated every 24h. LLM API keys stored in Vault. No secrets in environment variables. | M2-003 | M |
| M5-003 | **Real Cosign signing** | Replace signing stubs with real Fulcio/Rekor integration. SLSA Level 3 provenance generated. | M2-011 | M |
| M5-004 | **Security scanning integration** | SAST (gosec/semgrep), DAST (OWASP ZAP), SCA (Grype), Secrets (Gitleaks), IaC (Checkov) all wired to real tools. SBOM via Syft. | M3-004 | L |
| M5-005 | **Penetration testing** | OWASP Top 10 scan. API fuzzing. Auth bypass attempts. Tenant isolation verification. | M4-001 | L |

---

## M6: Compliance Certification

| ID | Task | Acceptance Criteria | Deps | Est |
|----|------|---------------------|------|-----|
| M6-001 | **SOC2 Type II evidence automation** | All SOC2 controls mapped. Evidence auto-generated from build artifacts. Control matrix produced. | M3-007 | L |
| M6-002 | **GDPR data residency enforcement** | EU tenant data stays in London region. PII pseudonymisation for cross-region audit replication. DSAR handling. | M2-010 | M |
| M6-003 | **ISO 27001, HIPAA, PCI-DSS control mapping** | Each framework's controls mapped to SDX build outputs. Evidence packages generated. Compliance badges issued. | M6-001 | L |

---

## M7: Multi-Region Deployment

| ID | Task | Acceptance Criteria | Deps | Est |
|----|------|---------------------|------|-----|
| M7-001 | **Helm charts for all services** | One chart per service. Values per region. Resource limits. HPA configured. | M3-007 | L |
| M7-002 | **Fleet GitOps configuration** | 5 NexoRegions (Lagos, London, Ashburn, Singapore, Sao Paulo). Fleet bundles per region. Automatic reconciliation. | M7-001 | L |
| M7-003 | **Multi-region NATS + Redpanda** | NATS JetStream per region (3 replicas). Redpanda cross-region for metrics and audit. Verify data residency compliance. | M7-002 | XL |
| M7-004 | **Chaos testing** | Litmus: agent pod crash, NATS partition, YugabyteDB node failure, Vault unavailability. Verify auto-recovery. | M7-003 | L |

---

## M8: GA Launch

| ID | Task | Acceptance Criteria | Deps | Est |
|----|------|---------------------|------|-----|
| M8-001 | **Billing integration** | Stripe/billing provider. Meter-based pricing. Plan enforcement. Usage reports. | M2-010 | L |
| M8-002 | **SDK generation** | Go, Python, TypeScript SDKs auto-generated from OpenAPI spec. Published to registries. | M2-017 | M |
| M8-003 | **Public API documentation site** | Hosted Redoc. Getting started guide. Authentication guide. Webhook guide. Error reference. | M2-017 | M |
| M8-004 | **Load testing at scale** | 50 concurrent builds. 500 agents. 10,000 RPS API. p99 < 500ms. Zero data loss under load. | M7-003 | XL |
| M8-005 | **GA release** | Version 1.0.0. All compliance badges green. Monitoring dashboards live. Runbooks complete. On-call rotation configured. | M8-001..M8-004 | M |

---

## Effort Key

| Symbol | Meaning | Approximate Duration |
|--------|---------|---------------------|
| S | Small | < 2 hours |
| M | Medium | 2-6 hours |
| L | Large | 6-16 hours |
| XL | Extra Large | 16-40 hours |
