# NeuroForge SDX — Use-Case Catalog

> Each use case maps to one or more gRPC RPCs and REST endpoints.
> Codex implements each as an application service method backed by domain aggregates.

---

## UC-01: Submit WorkItem

**Actor:** Human (Product Lead) or API Client
**Preconditions:** Tenant is ACTIVE. Quota not exhausted.
**Input:** Raw content (PRD text, user story, API spec, codebase URL), source_type, compliance_frameworks[], target_languages[]
**Flow:**
1. sdx-gateway validates auth (JWT), extracts tenant_id, checks rate limit
2. sdx-gateway checks idempotency key in DragonflyDB
3. sdx-ingestion-svc creates WorkItem aggregate (status=DRAFT)
4. SpecAgent parses content, extracts Requirements, identifies Ambiguities
5. WorkItem transitions to PARSED
6. If ambiguities exist, WorkItem awaits resolution (async via dashboard or API)
7. Once all ambiguities resolved, manifest is assembled
8. Human or ArchitectAgent approves manifest → WorkItem transitions to APPROVED
9. Emit `sdx.workitem.manifest.approved` → triggers Build creation

**Output:** WorkItemID, parsed requirements, ambiguities (if any)
**Error Cases:** TENANT_SUSPENDED (SDX-3001), QUOTA_EXHAUSTED (SDX-3002), INVALID_INPUT (SDX-2001), UNSUPPORTED_LANGUAGE (SDX-2002)
**gRPC:** `IngestionService.SubmitWorkItem`
**REST:** `POST /api/v1/work-items`

---

## UC-02: Resolve Ambiguity

**Actor:** Human (Product Lead) via Dashboard or API
**Preconditions:** WorkItem exists and has unresolved ambiguities.
**Input:** work_item_id, ambiguity_id, resolution text
**Flow:**
1. Find WorkItem by ID, verify tenant ownership
2. Find Ambiguity by ID within WorkItem
3. Set resolution and resolved_by
4. If all ambiguities resolved, auto-advance WorkItem status
5. Emit `sdx.workitem.ambiguity.resolved`

**Output:** Updated ambiguity with resolution
**Error Cases:** WORKITEM_NOT_FOUND (SDX-4001), AMBIGUITY_NOT_FOUND (SDX-4002), ALREADY_RESOLVED (SDX-3003)
**gRPC:** `IngestionService.ResolveAmbiguity`
**REST:** `PATCH /api/v1/work-items/{id}/ambiguities/{ambiguity_id}`

---

## UC-03: Initiate Build

**Actor:** System (triggered by `sdx.workitem.manifest.approved`)
**Preconditions:** WorkItem is APPROVED. Tenant quota allows new pipeline.
**Input:** work_item_id
**Flow:**
1. sdx-workflow-engine creates Build aggregate (status=INITIATED)
2. Consume tenant quota (pipeline count)
3. Start Temporal workflow `BuildPipelineWorkflow`
4. Workflow dispatches Stage 1 (Ingestion) activity
5. Emit `sdx.build.initiated`

**Output:** BuildID
**Error Cases:** WORKITEM_NOT_APPROVED (SDX-3004), QUOTA_EXHAUSTED (SDX-3002)
**gRPC:** Internal only (Temporal workflow start)

---

## UC-04: Execute Pipeline Stage

**Actor:** System (Temporal activity)
**Preconditions:** Build is in correct stage. Previous stage completed or recovered.
**Input:** build_id, stage_number
**Flow:**
1. sdx-agent-orchestrator selects available agent of required type
2. Agent assigned to build; status → ASSIGNED → PROCESSING
3. Agent executes stage-specific work (e.g., codegen, testing, scanning)
4. Agent produces output artifact, stores in RustFS
5. PipelineStageRecord updated with output_ref
6. If stage has associated gates, invoke QualityGateService
7. If all gates pass → advance to next stage
8. If gate fails → trigger RemediationLoop (see UC-07)
9. Checkpoint build state every 30s via heartbeat
10. Emit `sdx.build.stage.completed` or `sdx.build.stage.failed`

**Output:** Stage output reference, gate results
**Error Cases:** NO_AVAILABLE_AGENT (SDX-5001), AGENT_TIMEOUT (SDX-5002), LLM_UNAVAILABLE (SDX-6001)
**gRPC:** Internal (Temporal activities)

---

## UC-05: Evaluate Quality Gate

**Actor:** System (sdx-quality-svc)
**Preconditions:** Code or artifact for evaluation exists.
**Input:** build_id, gate_id, artifact_ref
**Flow:**
1. Retrieve gate definition (threshold, metric, operator)
2. Execute gate-specific evaluation (e.g., run gosec for G-07, compute CodeBLEU for G-01)
3. Record GateResult on Build aggregate
4. If passed → emit `sdx.gate.passed`
5. If failed → emit `sdx.gate.failed` with remediation_action
6. For CRITICAL gates (G-07 Security, G-14 Secrets), failure is non-remediable; build fails

**Output:** GateResult (gate_id, value, threshold, passed)
**Error Cases:** GATE_EVALUATION_ERROR (SDX-5003)
**gRPC:** `QualityService.EvaluateGate`

---

## UC-06: Sign and Publish Artifact

**Actor:** System (sdx-artifact-svc, triggered after security + compliance pass)
**Preconditions:** Security cleared. Compliance certified. SBOM generated.
**Input:** build_id, artifact_ref
**Flow:**
1. Compute SHA-256 content hash
2. Sign with Cosign (keyless via Fulcio)
3. Generate SLSA Level 3 provenance attestation
4. Publish signature to Rekor transparency log
5. Associate SBOM ref
6. Artifact status → SIGNED
7. Store in RustFS artifact registry
8. Artifact status → PUBLISHED
9. Emit `sdx.artifact.signed`, `sdx.artifact.published`

**Output:** Signed artifact with provenance
**Error Cases:** SIGNING_FAILED (SDX-5004), SBOM_MISSING (SDX-3005), PROVENANCE_INVALID (SDX-3006)
**gRPC:** `ArtifactService.SignArtifact`, `ArtifactService.PublishArtifact`

---

## UC-07: Execute Remediation Loop

**Actor:** System (Temporal sub-workflow)
**Preconditions:** Gate failed. Remediation attempts < 3.
**Input:** build_id, gate_id, failing_value
**Flow:**
1. Determine responsible agent (e.g., BuilderAgent for G-01 CodeBLEU, SecurityAgent for G-07)
2. Assign agent with gate failure context
3. Agent re-generates or fixes the artifact
4. Re-evaluate the gate
5. If passes → emit `sdx.remediation.succeeded`, resume pipeline
6. If fails and attempt < 3 → increment attempt, repeat from step 1
7. If fails and attempt == 3 → emit `sdx.remediation.exhausted`, escalate (see UC-08)

**Output:** Fixed artifact or escalation
**Error Cases:** REMEDIATION_AGENT_UNAVAILABLE (SDX-5001), MAX_RETRIES_EXCEEDED (SDX-3007)
**gRPC:** Internal (Temporal sub-workflow)

---

## UC-08: Trigger Human Escalation

**Actor:** System (after 5 recovery attempts or 3 remediation failures)
**Preconditions:** Automated recovery exhausted.
**Input:** build_id, reason, diagnostic_context
**Flow:**
1. Collect full diagnostic context: build state, gate results, agent logs, last checkpoint
2. Create PagerDuty incident with context URL
3. Build status → FAILED
4. Emit `sdx.build.escalation.triggered`
5. Notify tenant via webhook (if configured) and dashboard

**Output:** PagerDuty incident ID, context URL
**Error Cases:** PAGERDUTY_UNAVAILABLE (SDX-6002)
**gRPC:** Internal

---

## UC-09: Recover Build from Checkpoint

**Actor:** System (Temporal workflow recovery)
**Preconditions:** Build has a valid checkpoint. Recovery attempts < 5.
**Input:** build_id
**Flow:**
1. Load last valid checkpoint from NATS JetStream
2. Verify checkpoint hash integrity
3. If hash invalid → try previous checkpoint
4. Restore Build state to checkpoint stage
5. Build status → RECOVERING
6. Resume Temporal workflow from checkpoint stage
7. Increment recovery_attempts
8. Emit `sdx.build.recovery.started`
9. If resumes successfully → emit `sdx.build.recovery.succeeded`

**Output:** Recovered build at checkpoint stage
**Error Cases:** CHECKPOINT_CORRUPTED (SDX-5005), MAX_RECOVERY_EXCEEDED (SDX-3008)
**gRPC:** Internal (Temporal workflow signal)

---

## UC-10: Create Tenant

**Actor:** Human (Admin) or API Client
**Preconditions:** Valid Authentik session with admin role.
**Input:** name, plan_tier, nexo_region, compliance_frameworks[], billing_account_ref
**Flow:**
1. Validate plan_tier and nexo_region
2. Create Tenant aggregate
3. Provision K8s namespace `sdx-{tenant_id_short}`
4. Create Vault mount at `secret/sdx/{tenant_id}/`
5. Create DragonflyDB partition
6. Set quotas based on plan_tier
7. Create default Workspace
8. Emit `sdx.tenant.created`

**Output:** TenantID, workspace_id, API keys
**Error Cases:** INVALID_PLAN (SDX-2003), INVALID_REGION (SDX-2004), PROVISIONING_FAILED (SDX-5006)
**gRPC:** `TenantService.CreateTenant`
**REST:** `POST /api/v1/tenants`

---

## UC-11: Query Build Status

**Actor:** Human or API Client
**Preconditions:** Tenant owns the build.
**Input:** build_id
**Flow:**
1. Validate tenant ownership
2. Load Build aggregate (or read-model from DragonflyDB cache)
3. Return current status, stage, gate results, artifact refs

**Output:** Build summary with stage details and gate results
**Error Cases:** BUILD_NOT_FOUND (SDX-4003), TENANT_MISMATCH (SDX-1003)
**gRPC:** `SDXGatewayService.GetBuildStatus`
**REST:** `GET /api/v1/builds/{id}`

---

## UC-12: List Builds

**Actor:** Human or API Client
**Preconditions:** Valid tenant session.
**Input:** tenant_id, filters (status, date_range, work_item_id), pagination
**Flow:**
1. Query Build read-model with filters
2. Apply cursor-based pagination
3. Return list with pagination token

**Output:** Paginated list of build summaries
**Error Cases:** INVALID_FILTER (SDX-2005)
**gRPC:** `SDXGatewayService.ListBuilds`
**REST:** `GET /api/v1/builds`

---

## UC-13: Download Artifact

**Actor:** Human or API Client
**Preconditions:** Tenant owns the artifact. Artifact is PUBLISHED.
**Input:** artifact_id
**Flow:**
1. Validate tenant ownership
2. Verify artifact status is PUBLISHED (not REVOKED)
3. Generate pre-signed S3 URL (TTL: 1 hour)
4. Return download URL

**Output:** Pre-signed download URL, artifact metadata
**Error Cases:** ARTIFACT_NOT_FOUND (SDX-4004), ARTIFACT_REVOKED (SDX-3009), TENANT_MISMATCH (SDX-1003)
**gRPC:** `SDXGatewayService.GetArtifact`
**REST:** `GET /api/v1/artifacts/{id}`

---

## UC-14: Configure Webhook

**Actor:** Human or API Client
**Preconditions:** Tenant is ACTIVE.
**Input:** target_url, events[] (e.g., build.completed, gate.failed), secret (for HMAC)
**Flow:**
1. Validate target_url (must be HTTPS)
2. Validate events are known webhook event types
3. Store webhook config for tenant
4. Send test ping to verify connectivity
5. Return webhook_id

**Output:** webhook_id, test ping result
**Error Cases:** INVALID_URL (SDX-2006), UNKNOWN_EVENT_TYPE (SDX-2007), PING_FAILED (SDX-6003)
**gRPC:** `SDXGatewayService.CreateWebhook`
**REST:** `POST /api/v1/webhooks`

---

## UC-15: Route LLM Request

**Actor:** System (any agent via sdx-llm-gateway)
**Preconditions:** Agent has valid SPIFFE identity. Model routing config exists.
**Input:** task_class, prompt, max_tokens, temperature, agent_id
**Flow:**
1. Look up agent's ModelRoutingConfig
2. Check primary model circuit breaker state
3. If CLOSED → route to primary model
4. If OPEN → route to fallback model
5. If HALF_OPEN → send probe request to primary
6. Execute LLM completion
7. Record token usage and latency
8. If primary fails → open circuit breaker, route to fallback
9. Emit `sdx.llm.request.completed`

**Output:** Completion response, token usage
**Error Cases:** ALL_MODELS_UNAVAILABLE (SDX-6001), TOKEN_LIMIT_EXCEEDED (SDX-3010), TIMEOUT (SDX-6004)
**gRPC:** `LLMService.Complete`

---

## UC-16: Generate Compliance Evidence

**Actor:** System (sdx-compliance-svc, triggered by `sdx.quality.passed`)
**Preconditions:** Build has passed quality gates. Tenant has compliance profile.
**Input:** build_id, compliance_frameworks[]
**Flow:**
1. For each framework, load control catalog
2. Map build artifacts to controls (test results → testing controls, SBOM → asset inventory, audit log → change management)
3. Generate evidence documents (JSON + PDF)
4. Calculate control coverage percentage
5. If 100% → generate compliance badge (PASS)
6. If < 100% → flag gaps, badge = FAIL
7. Store evidence package in RustFS
8. Emit `sdx.compliance.certified` or `sdx.compliance.gap.detected`

**Output:** CompliancePackage artifact, badge
**Error Cases:** UNKNOWN_FRAMEWORK (SDX-2008), EVIDENCE_GENERATION_FAILED (SDX-5007)
**gRPC:** `ComplianceService.GenerateEvidence`

---

## UC-17: Deploy to Target Environment

**Actor:** System (sdx-deploy-svc, triggered by `sdx.artifact.signed`)
**Preconditions:** Artifact is SIGNED. Target environment exists.
**Input:** build_id, artifact_ids[], target_env, target_region
**Flow:**
1. Create Deployment aggregate
2. Generate Fleet GitOps bundle from IaC artifacts
3. Push bundle to Fleet (GitOps reconciliation)
4. Wait for Fleet to apply (poll or watch)
5. Run smoke tests against deployed service
6. If smoke tests pass → validate SLOs for 15 minutes
7. If SLOs valid → Deployment status → DEPLOYED
8. If smoke tests fail or SLO violation → trigger auto-rollback
9. Emit `sdx.deployment.completed` or `sdx.deployment.rollback.triggered`

**Output:** DeploymentID, deployment status
**Error Cases:** FLEET_UNAVAILABLE (SDX-6005), SMOKE_TEST_FAILED (SDX-3011), SLO_VIOLATION (SDX-3012)
**gRPC:** `DeploymentService.Deploy`

---

## UC-18: Query Audit Log

**Actor:** Human (Compliance Officer) or API Client
**Preconditions:** Tenant owns the data. User has AUDIT_READER role.
**Input:** tenant_id, date_range, event_types[], actor_id, target_id, pagination
**Flow:**
1. Validate AUDIT_READER permission
2. Query ClickHouse audit log with filters
3. Apply cursor-based pagination
4. Return entries with chain verification status

**Output:** Paginated audit entries
**Error Cases:** INSUFFICIENT_PERMISSION (SDX-1002), INVALID_DATE_RANGE (SDX-2009)
**gRPC:** `AuditService.QueryAuditLog`
**REST:** `GET /api/v1/audit-log`

---

## UC-19: Verify Audit Chain Integrity

**Actor:** System (nightly job) or Human (on-demand)
**Preconditions:** Audit entries exist for the given range.
**Input:** start_entry_id, end_entry_id (or date_range)
**Flow:**
1. Load entries in order
2. For each entry, verify prev_hash matches SHA-256 of previous entry
3. If all match → chain_valid = true
4. If mismatch found → record first_broken_entry_id, chain_valid = false, alert

**Output:** AuditChainVerification result
**Error Cases:** ENTRIES_NOT_FOUND (SDX-4005), CHAIN_BROKEN (SDX-3013 — critical alert)
**gRPC:** `AuditService.VerifyIntegrity`
**REST:** `POST /api/v1/audit-log/verify`

---

## UC-20: Update Tenant Plan

**Actor:** Human (Admin) or Billing System
**Preconditions:** Tenant exists and is ACTIVE.
**Input:** tenant_id, new_plan
**Flow:**
1. Validate new_plan is a valid PlanTier
2. If downgrade: verify current usage does not exceed new plan's quotas
3. Update Tenant plan and quotas
4. If upgrade to ENTERPRISE/SOVEREIGN: provision dedicated agent pool
5. Emit `sdx.tenant.plan.upgraded` or `sdx.tenant.plan.downgraded`

**Output:** Updated tenant with new quotas
**Error Cases:** USAGE_EXCEEDS_NEW_QUOTA (SDX-3014), INVALID_PLAN_TRANSITION (SDX-3015)
**gRPC:** `TenantService.UpdatePlan`
**REST:** `PATCH /api/v1/tenants/{id}/plan`
