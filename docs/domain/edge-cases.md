# NeuroForge SDX — Edge Cases and Failure Modes

> Every edge case must have an explicit handling strategy.
> Codex MUST implement these as tests BEFORE implementation (TDD).

---

## Build Pipeline Edge Cases

### EC-01: LLM Returns Empty Response
**Scenario:** LLM returns 200 OK but with empty content or content below minimum quality threshold.
**Detection:** Response body is empty or content length < 10 chars.
**Handling:** Treat as transient failure. Retry with same prompt up to 3 times. If persistent, trigger model fallback. Log anomaly for LearningAgent.
**Test:** `TestLLMEmptyResponse_TriggersRetryAndFallback`

### EC-02: LLM Returns Malformed Code
**Scenario:** LLM generates code that doesn't parse (syntax errors).
**Detection:** Language-specific parser fails (go build, tsc --noEmit, python -m py_compile).
**Handling:** Automatic retry with enhanced prompt including the parse error. Max 3 retries. If unresolved, treat as G-01 (CodeBLEU) failure and enter remediation loop.
**Test:** `TestMalformedCode_TriggersParseRetry`

### EC-03: Circular Remediation Loop
**Scenario:** Gate fails → remediation "fixes" it → different gate fails → remediation breaks original gate → infinite loop.
**Detection:** Same gate fails with same value twice in a row after remediation.
**Handling:** After detecting cycle (same gate_id, same or worse value, 2nd attempt), stop remediation for that gate. Escalate entire build. Log cycle for LearningAgent analysis.
**Test:** `TestCircularRemediation_DetectedAndEscalated`

### EC-04: Checkpoint Corruption
**Scenario:** Checkpoint stored in NATS JetStream has corrupted data (hash mismatch).
**Detection:** SHA-256 of restored state != stored checkpoint_hash.
**Handling:** Attempt previous checkpoint (up to 5 checkpoints back). If all corrupted, fail build with CHECKPOINT_CORRUPTED. Trigger infrastructure alert (possible storage issue).
**Test:** `TestCheckpointCorruption_FallbackToPrevious`

### EC-05: Agent Dies Mid-Stage
**Scenario:** Agent pod is OOM-killed or node failure during active processing.
**Detection:** Temporal activity heartbeat timeout (10s). Agent health monitor detects missed heartbeats.
**Handling:** Temporal automatically retries the activity on a different agent. Build state restored from last heartbeat data. If no other agent available, queue for up to 5 min.
**Test:** `TestAgentCrash_TemporalRetryOnDifferentAgent`

### EC-06: Concurrent Builds Modify Same Artifact
**Scenario:** Two builds for the same tenant produce artifacts with the same content hash (duplicate).
**Detection:** Content hash collision on save to RustFS.
**Handling:** Content-addressed storage is idempotent — second write is a no-op. Both builds reference the same artifact. No conflict.
**Test:** `TestDuplicateArtifact_ContentAddressedDedup`

### EC-07: Build Takes Longer Than 24 Hours
**Scenario:** Very large codebase or many remediation cycles push build past 24h max.
**Detection:** Temporal workflow timeout fires.
**Handling:** Fail build with TIMEOUT. Save all partial artifacts. Human escalation with full context. All consumed resources are billed.
**Test:** `TestBuildTimeout_24Hours_GracefulFail`

### EC-08: NATS JetStream Becomes Unavailable
**Scenario:** All 3 NATS nodes in a region go down simultaneously.
**Detection:** NATS client connection error; publish timeout.
**Handling:** Temporal workflows continue (state in Temporal server). Events are buffered in a local WAL (write-ahead log) per service. On NATS recovery, WAL is drained in order. No events lost.
**Test:** `TestNATSOutage_WALBufferingAndDrain`

---

## Tenant Edge Cases

### EC-09: Quota Exhausted Mid-Build
**Scenario:** Tenant's pipeline quota is consumed after build starts (another build claimed the last slot).
**Detection:** This cannot happen — quota is consumed atomically at build initiation. The build that already started continues.
**Handling:** Only new build initiation is blocked. Running builds are never cancelled due to quota. Next billing period resets the meter.
**Test:** `TestQuotaMidBuild_RunningBuildsNotAffected`

### EC-10: Tenant Suspended While Build In Progress
**Scenario:** Admin suspends tenant (payment failure) while a build is actively running.
**Detection:** Tenant status change event received by sdx-workflow-engine.
**Handling:** Running builds are allowed to complete (grace period: current build only). No new builds accepted. Artifacts from completed build are accessible for 30 days.
**Test:** `TestTenantSuspension_GracePeriodForRunningBuild`

### EC-11: Plan Downgrade Below Current Usage
**Scenario:** Tenant on ENTERPRISE (unlimited) requests downgrade to STARTER (5 pipelines/month) but has 20 active workspaces.
**Detection:** Validation on `ChangePlan()`.
**Handling:** Reject downgrade with SDX-3014 USAGE_EXCEEDS_NEW_QUOTA. Return specific usage vs new limit breakdown. Tenant must reduce usage first.
**Test:** `TestPlanDowngrade_RejectedIfUsageExceeds`

### EC-12: Tenant Region Mismatch
**Scenario:** Request arrives at London gateway but tenant is registered in Lagos.
**Detection:** Tenant's nexo_region != request gateway region.
**Handling:** Return redirect response with correct regional gateway URL. Do not process the request. Log for monitoring (should not happen with proper DNS routing).
**Test:** `TestRegionMismatch_RedirectToCorrectGateway`

---

## Security Edge Cases

### EC-13: SPIFFE SVID Expires During Processing
**Scenario:** Agent's SVID rotates while it's mid-task, and the new SVID fails issuance.
**Detection:** SPIRE agent reports rotation failure; mTLS handshake fails on next inter-service call.
**Handling:** Agent transitions to TERMINATED. Temporal detects heartbeat failure. Activity retried on different agent. Failed agent requires re-registration after SPIRE issue is resolved.
**Test:** `TestSVIDExpiry_AgentTerminatedAndTaskRetried`

### EC-14: Cosign Signing Service Unavailable
**Scenario:** Fulcio CA or Rekor transparency log is down during artifact signing.
**Detection:** Cosign client returns connection error.
**Handling:** Retry with exponential backoff (5 attempts over 10 min). If still unavailable, queue the signing operation. Build remains in SIGNING state. Alert SRE team.
**Test:** `TestCosignUnavailable_RetryAndQueue`

### EC-15: Secret Found in Generated Code After Signing
**Scenario:** Post-deployment security scan finds a secret that G-14 missed.
**Detection:** External security tool or manual review.
**Handling:** Immediately revoke artifact. Rotate the exposed secret in Vault. Trigger rebuild with enhanced secret patterns. File incident. Update Gitleaks rules.
**Test:** `TestPostDeploySecretDiscovery_ArtifactRevoked`

---

## Data Consistency Edge Cases

### EC-16: YugabyteDB and ClickHouse Projection Diverge
**Scenario:** Event processed in YugabyteDB but ClickHouse consumer lags, showing stale data on dashboard.
**Detection:** Monitoring alert: projection lag > 5 seconds.
**Handling:** ClickHouse consumer catches up automatically (NATS replay). Dashboard shows "data may be delayed" indicator if lag > 2s. No data loss — NATS is the source of truth.
**Test:** `TestProjectionLag_DashboardIndicator`

### EC-17: Duplicate Event Delivery
**Scenario:** NATS delivers same event twice (at-least-once semantics).
**Detection:** Event handler checks event_id against processed set (DragonflyDB, 2-min dedup window).
**Handling:** Second delivery is silently dropped. All event handlers MUST be idempotent. Repository operations use upsert semantics where appropriate.
**Test:** `TestDuplicateEvent_IdempotentHandling`

### EC-18: Concurrent Writes to Same Build Aggregate
**Scenario:** Two Temporal activities try to update the same Build simultaneously.
**Detection:** Optimistic concurrency control — version field mismatch on save.
**Handling:** YugabyteDB rejects the second write (version conflict). Temporal retries the activity, which re-reads the aggregate with the latest version.
**Test:** `TestConcurrentBuildUpdate_OptimisticLocking`

---

## Deployment Edge Cases

### EC-19: Fleet GitOps Reconciliation Stuck
**Scenario:** Fleet applies the bundle but reconciliation hangs (e.g., pending PVC, image pull error).
**Detection:** Deployment in APPLYING state for > 10 min without progress.
**Handling:** sdx-deploy-svc polls Fleet status. After 10 min timeout, mark deployment as FAILED. Trigger rollback to previous known-good deployment. Alert SRE.
**Test:** `TestFleetReconciliationStuck_TimeoutAndRollback`

### EC-20: SLO Violation During Validation Window
**Scenario:** Deployment passes smoke tests but SLO validation detects p99 latency spike at minute 8 of 15-min window.
**Detection:** SLO validation monitors metrics continuously during the 15-min window.
**Handling:** Immediately trigger rollback. Record SLO violation details. Feed data to PerformanceAgent for analysis. Build status remains COMPLETED but deployment is ROLLED_BACK.
**Test:** `TestSLOViolation_AutoRollbackDuringValidation`

### EC-21: Rollback Target Also Broken
**Scenario:** Auto-rollback to previous deployment, but that deployment has since been revoked or also has issues.
**Detection:** Previous deployment artifact is in REVOKED state.
**Handling:** Walk back through deployment history until a non-revoked, healthy deployment is found. If none exists (first deployment), fail deployment and alert. Service remains in degraded state.
**Test:** `TestRollbackChain_FindLastHealthyDeployment`

---

## LLM Gateway Edge Cases

### EC-22: All Models Unavailable Simultaneously
**Scenario:** Both cloud providers (Anthropic, OpenAI) and on-prem vLLM are down.
**Detection:** All circuit breakers are OPEN.
**Handling:** Queue requests for up to 5 min with backpressure. If no model recovers within 5 min, fail all queued requests with SDX-6001. Builds in progress pause at current stage (Temporal timer). Alert P1 incident.
**Test:** `TestAllModelsDown_QueueAndTimeout`

### EC-23: Model Returns Toxic/Harmful Content
**Scenario:** LLM generates code with offensive variable names, comments, or content.
**Detection:** Content filter in LLM gateway scans response for offensive patterns.
**Handling:** Reject response. Retry with appended safety prompt. If persistent, switch to different model. Log incident for review.
**Test:** `TestToxicContent_FilterAndRetry`

### EC-24: Token Budget Exceeded Mid-Generation
**Scenario:** LLM runs out of output tokens before completing the code generation.
**Detection:** Response has `finish_reason: length` or equivalent.
**Handling:** Detect truncation. Split the generation into smaller chunks. Retry with continuation prompt that includes the partial output. Stitch results together.
**Test:** `TestTokenBudgetExceeded_ChunkAndContinue`

---

## Audit Edge Cases

### EC-25: Audit Hash Chain Break
**Scenario:** Nightly integrity check finds a hash chain break.
**Detection:** `AuditService.VerifyIntegrity` returns `chain_valid: false`.
**Handling:** P0 critical alert. Freeze audit writes to prevent further contamination. Identify break point. Investigate root cause (bug, corruption, or tampering). Reconstruct from NATS event replay if possible. File compliance incident.
**Test:** `TestAuditChainBreak_CriticalAlert`

### EC-26: GDPR Deletion Request for Audited Tenant
**Scenario:** EU tenant requests data erasure under GDPR Article 17.
**Detection:** DSAR (Data Subject Access Request) received.
**Handling:** Audit logs are pseudonymised, not deleted (legal basis: legitimate interest for compliance record-keeping). PII in WorkItems and Builds is fully erased. Tenant is DEPROVISIONED. Pseudonymised audit entries retained for 7 years.
**Test:** `TestGDPRDeletion_PseudonymisedAuditRetained`

---

## Webhook Edge Cases

### EC-27: Webhook Target Returns 301 Redirect
**Scenario:** Tenant's webhook URL redirects to a different URL.
**Detection:** HTTP 301/302 response.
**Handling:** DO NOT follow redirects (security risk: SSRF). Treat as delivery failure. Notify tenant to update webhook URL. Log the redirect target for the tenant.
**Test:** `TestWebhookRedirect_NotFollowed`

### EC-28: Webhook Target Returns 200 But Body Indicates Failure
**Scenario:** Target returns 200 OK but response body says "error: invalid signature".
**Detection:** SDX does not inspect response body content. 2xx = success.
**Handling:** From SDX's perspective, delivery succeeded (2xx). Tenant's responsibility to validate signature. SDX provides signature verification guide in docs.
**Test:** `TestWebhook200Response_TreatedAsSuccess`
