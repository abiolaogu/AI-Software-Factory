# NeuroForge SDX — Acceptance Criteria

> Every criterion is a testable assertion. Codex MUST write a failing test for each before implementation.
> Format: `[AGGREGATE-AC-XX]` for domain layer, `[SVC-AC-XX]` for service layer, `[E2E-AC-XX]` for end-to-end.

---

## WorkItem Aggregate

| ID | Criterion | Test Type |
|----|-----------|-----------|
| WI-AC-01 | Creating a WorkItem with empty raw_content returns `ErrInvariantViolated` | Unit |
| WI-AC-02 | Creating a WorkItem with raw_content > 5MB returns `ErrInvariantViolated` | Unit |
| WI-AC-03 | `MarkAsParsed()` with 0 requirements returns `ErrInvariantViolated` (WI-INV-01) | Unit |
| WI-AC-04 | `MarkAsParsed()` with ≥1 requirement succeeds and emits `WorkItemParsedPayload` | Unit |
| WI-AC-05 | `ApproveManifest()` with unresolved ambiguities returns `ErrInvariantViolated` (WI-INV-02) | Unit |
| WI-AC-06 | `ApproveManifest()` with 0 languages returns `ErrInvariantViolated` (WI-INV-03) | Unit |
| WI-AC-07 | `ApproveManifest()` with language outside tenant allowance returns `ErrInvariantViolated` (WI-INV-07) | Unit |
| WI-AC-08 | Attempting DRAFT→APPROVED (skipping PARSED) returns `ErrInvariantViolated` (WI-INV-04) | Unit |
| WI-AC-09 | FAILED→DRAFT restart resets requirements and ambiguities | Unit |
| WI-AC-10 | COMPLETED WorkItem cannot transition to any state | Unit |
| WI-AC-11 | `tenant_id` is immutable after creation (WI-INV-06) | Unit |
| WI-AC-12 | Full lifecycle: DRAFT→PARSED→MANIFEST_REVIEW→APPROVED emits 4 events in order | Unit |

## Build Aggregate

| ID | Criterion | Test Type |
|----|-----------|-----------|
| B-AC-01 | `AdvanceStage(7)` with failing BLOCKING gate returns `ErrInvariantViolated` (B-INV-01) | Unit |
| B-AC-02 | `AdvanceStage(7)` with passing gate succeeds and status becomes VERIFYING | Unit |
| B-AC-03 | `stage_to_status()` maps stage 1→INITIATED, 7→VERIFYING, 10→CERTIFYING, 14→SIGNING, 15→DEPLOYING | Unit |
| B-AC-04 | Status is always derived from current_stage (ADR-014 invariant) | Unit |
| B-AC-05 | `RecoverFromCheckpoint()` with recovery_attempts=5 returns `ErrInvariantViolated` (B-INV-03) | Unit |
| B-AC-06 | `RecoverFromCheckpoint()` with recovery_attempts=4 succeeds and increments to 5 | Unit |
| B-AC-07 | `RecordGateResult(G-07)` with CRITICAL finding sets build to FAILED, not remediable (B-INV-05) | Unit |
| B-AC-08 | `RecordGateResult(G-14)` with any detection sets build to FAILED (B-INV-06) | Unit |
| B-AC-09 | COMPLETED build rejects all mutations (B-INV-07) | Unit |
| B-AC-10 | `stages` is append-only; cannot modify past stages (B-INV-09) | Unit |
| B-AC-11 | `StartRemediation()` with attempt=4 for same gate returns `ErrInvariantViolated` (B-INV-12) | Unit |
| B-AC-12 | Checkpoint hash must be refreshed within 30s invariant is documentable (test heartbeat logic) | Unit |
| B-AC-13 | Circular remediation detection: same gate, same or worse value on 2nd attempt → escalate (EC-03) | Unit |
| B-AC-14 | Build MUST have SBOM artifact before entering stage 14 (B-INV-04) | Unit |
| B-AC-15 | `Fail()` from any active state succeeds; `Fail()` from COMPLETED fails | Unit |

## Artifact Aggregate

| ID | Criterion | Test Type |
|----|-----------|-----------|
| A-AC-01 | content_hash immutable after creation (A-INV-01) | Unit |
| A-AC-02 | `Sign()` without valid SignatureRef returns error (A-INV-02) | Unit |
| A-AC-03 | `Publish()` without SIGNED status returns error (A-INV-03) | Unit |
| A-AC-04 | `Publish()` without SBOM returns error (A-INV-04) | Unit |
| A-AC-05 | ProvenanceRef must have slsa_level=3 (A-INV-05) | Unit |
| A-AC-06 | REVOKED artifact cannot transition to any state (A-INV-06) | Unit |
| A-AC-07 | storage_uri must include tenant_id (A-INV-07) | Unit |
| A-AC-08 | signature_ref is write-once (A-INV-08) | Unit |
| A-AC-09 | Full lifecycle: CREATED→SIGNED→PUBLISHED→DEPLOYED emits 4 events | Unit |

## Tenant Aggregate

| ID | Criterion | Test Type |
|----|-----------|-----------|
| T-AC-01 | `ConsumeQuota()` when current_value ≥ limit returns `ErrQuotaExhausted` (T-INV-01) | Unit |
| T-AC-02 | STARTER tenant accessing Rust language returns error (T-INV-02) | Unit |
| T-AC-03 | SOVEREIGN tenant with cloud LLM routing returns error (T-INV-03) | Unit |
| T-AC-04 | SUSPENDED tenant cannot create WorkItems (T-INV-05) | Unit |
| T-AC-05 | Plan downgrade from ENTERPRISE (unlimited) to STARTER with 20 pipelines used returns error (T-INV-10) | Unit |
| T-AC-06 | `nexo_region` is immutable (T-INV-08) | Unit |
| T-AC-07 | `ResetBillingPeriod()` sets current_value to 0 (T-INV-09) | Unit |
| T-AC-08 | Adding workspace member beyond plan limit returns error (T-INV-07) | Unit |
| T-AC-09 | Full lifecycle: PROVISIONING→ACTIVE→SUSPENDED→ACTIVE (reactivate) works | Unit |

## Agent Aggregate

| ID | Criterion | Test Type |
|----|-----------|-----------|
| AG-AC-01 | 3 missed heartbeats (30s) transitions to UNHEALTHY (AG-INV-01) | Unit |
| AG-AC-02 | Expired SPIFFE SVID transitions to TERMINATED (AG-INV-02) | Unit |
| AG-AC-03 | PROCESSING agent must have non-null build_id (AG-INV-03) | Unit |
| AG-AC-04 | IDLE agent must have null build_id (AG-INV-04) | Unit |
| AG-AC-05 | TERMINATED agent cannot be reassigned (AG-INV-06) | Unit |
| AG-AC-06 | Only one build assignment at a time (AG-INV-07) | Unit |
| AG-AC-07 | Agent type immutable after registration (AG-INV-08) | Unit |

## Deployment Aggregate

| ID | Criterion | Test Type |
|----|-----------|-----------|
| D-AC-01 | Deploy with unsigned artifact returns error (D-INV-01) | Unit |
| D-AC-02 | PROD deployment with failed smoke test triggers rollback (D-INV-02) | Unit |
| D-AC-03 | SLO violation within 15 min triggers auto-rollback (D-INV-03) | Unit |
| D-AC-04 | ROLLED_BACK records previous deployment ID (D-INV-04) | Unit |
| D-AC-05 | Only one active deployment per tenant+service+env (D-INV-05) | Unit |

---

## Service-Level Acceptance Criteria

### sdx-gateway

| ID | Criterion | Test Type |
|----|-----------|-----------|
| GW-AC-01 | Unauthenticated request returns 401 with SDX-1001 | Integration |
| GW-AC-02 | Request for another tenant's resource returns 403 with SDX-1003 | Integration |
| GW-AC-03 | Rate limit exceeded returns 429 with SDX-9001 and Retry-After header | Integration |
| GW-AC-04 | Idempotency key returns cached response on duplicate request (same body) | Integration |
| GW-AC-05 | Idempotency key with different body returns FIRST request's response | Integration |
| GW-AC-06 | `/health` returns SERVING with component statuses | Integration |
| GW-AC-07 | `/openapi.json` returns valid OpenAPI 3.1 spec | Integration |
| GW-AC-08 | All 17 RPCs route correctly to downstream services | Integration |
| GW-AC-09 | X-Request-ID header present on every response (ULID format) | Integration |
| GW-AC-10 | X-RateLimit-* headers present on every response | Integration |

### sdx-ingestion-svc

| ID | Criterion | Test Type |
|----|-----------|-----------|
| ING-AC-01 | SubmitWorkItem creates WorkItem with DRAFT status | Integration |
| ING-AC-02 | SubmitWorkItem with invalid source_type returns SDX-2001 | Integration |
| ING-AC-03 | ResolveAmbiguity with non-existent ambiguity_id returns SDX-4002 | Integration |
| ING-AC-04 | ApproveManifest triggers Build creation and returns build_id | Integration |
| ING-AC-05 | ListWorkItems with status filter returns correct subset | Integration |
| ING-AC-06 | Cursor-based pagination works correctly across pages | Integration |
| ING-AC-07 | DeleteWorkItem on APPROVED status returns SDX-3016 | Integration |

### sdx-llm-gateway

| ID | Criterion | Test Type |
|----|-----------|-----------|
| LLM-AC-01 | Complete routes to primary model when circuit CLOSED | Integration |
| LLM-AC-02 | 3 consecutive failures open circuit breaker | Integration |
| LLM-AC-03 | OPEN circuit routes to fallback model, emits ModelFallbackPayload | Integration |
| LLM-AC-04 | HALF_OPEN after 30s cooldown sends probe to primary | Integration |
| LLM-AC-05 | 2 probe successes close circuit | Integration |
| LLM-AC-06 | Empty LLM response triggers retry (up to 3) (EC-01) | Integration |
| LLM-AC-07 | Token budget exceeded splits into chunks (EC-24) | Integration |
| LLM-AC-08 | All models unavailable returns SDX-6001 after 5min queue (EC-22) | Integration |
| LLM-AC-09 | Cost report aggregates by tenant, build, and model | Integration |

### sdx-audit-svc

| ID | Criterion | Test Type |
|----|-----------|-----------|
| AUD-AC-01 | Append creates entry with prev_hash chaining to previous entry | Integration |
| AUD-AC-02 | VerifyIntegrity detects broken hash chain (EC-25) | Integration |
| AUD-AC-03 | 100 entries appended and verified in < 1 second | Integration |
| AUD-AC-04 | QueryAuditLog with AUDIT_READER role succeeds | Integration |
| AUD-AC-05 | QueryAuditLog without AUDIT_READER role returns SDX-1002 | Integration |

### Webhook Delivery

| ID | Criterion | Test Type |
|----|-----------|-----------|
| WH-AC-01 | build.completed event triggers webhook delivery to configured URL | Integration |
| WH-AC-02 | X-SDX-Signature contains valid HMAC-SHA256 | Integration |
| WH-AC-03 | Failed delivery retries 5 times with exponential backoff | Integration |
| WH-AC-04 | After 5 failures, event moves to dead letter queue | Integration |
| WH-AC-05 | 301 redirect is NOT followed (EC-27) | Integration |
| WH-AC-06 | TestWebhook sends ping and reports result | Integration |

---

## End-to-End Acceptance Criteria

| ID | Criterion | Test Type |
|----|-----------|-----------|
| E2E-AC-01 | Submit PRD → WorkItem PARSED → ambiguities resolved → manifest APPROVED → Build INITIATED | E2E |
| E2E-AC-02 | Build progresses through all 16 stages with LLM stubs → COMPLETED | E2E |
| E2E-AC-03 | Gate G-01 failure at stage 7 → remediation loop → agent fixes → gate passes → continues | E2E |
| E2E-AC-04 | Gate G-07 CRITICAL at stage 9 → build FAILED immediately → escalation triggered | E2E |
| E2E-AC-05 | Agent crash mid-stage → Temporal retries on different agent → build continues (EC-05) | E2E |
| E2E-AC-06 | Build checkpoint → simulate crash → recover from checkpoint → build completes (EC-04) | E2E |
| E2E-AC-07 | Tenant quota exhaustion → new build rejected with SDX-3002 → existing build continues (EC-09) | E2E |
| E2E-AC-08 | COMPLETED build produces: source code, binary, container image, SBOM, compliance package, docs — all SIGNED with SLSA L3 provenance | E2E |
| E2E-AC-09 | Full audit trail: every action in the build has a hash-chained audit entry | E2E |
| E2E-AC-10 | Webhook fires for build.completed with valid HMAC signature | E2E |
| E2E-AC-11 | Two tenants cannot see each other's builds, artifacts, or audit entries | E2E |
| E2E-AC-12 | `make api-compliance-full` passes: proto lint + OpenAPI lint + breaking check + compliance check | E2E |

---

## Antigravity Verification Checklist

> Antigravity (Verifier role) runs these checks after each Codex implementation milestone.

| # | Check | Automated | Gate |
|---|-------|-----------|------|
| 1 | All unit tests pass (`go test ./domain/...`) | Yes | M1 |
| 2 | All invariants from aggregates.md have corresponding test assertions | Manual | M1 |
| 3 | All state machine transitions from state-machines.md are enforced | Manual | M1 |
| 4 | All 28 edge cases from edge-cases.md have tests | Manual | M1 |
| 5 | All integration tests pass (`go test ./internal/...`) | Yes | M2 |
| 6 | All contract tests pass (`make contract-test`) | Yes | M2 |
| 7 | API compliance passes (`make api-compliance-full`) | Yes | M2 |
| 8 | E2E pipeline test passes | Yes | M3 |
| 9 | No GPL/AGPL dependencies | Yes (license checker) | All |
| 10 | All proto files lint clean (`buf lint`) | Yes | All |
| 11 | No breaking proto changes (`buf breaking`) | Yes | M2+ |
| 12 | golangci-lint clean | Yes | All |
| 13 | Coverage ≥ 90% line, ≥ 85% branch on domain layer | Yes | M1 |
| 14 | No hardcoded secrets (gitleaks) | Yes | All |
| 15 | Cyclomatic complexity ≤ 10 per function | Yes | All |
