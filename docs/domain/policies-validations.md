# NeuroForge SDX — Policies and Validation Rules

> Policies are domain-level decision rules triggered by domain events.
> Validation rules are checked synchronously on aggregate mutations.
> Codex implements policies as event handlers and validations as guard clauses in aggregate methods.

---

## 1. Remediation Policy

**Trigger:** `sdx.gate.failed` event
**Scope:** Build Aggregate
**Logic:**
```
WHEN gate_failed(build_id, gate_id, value, threshold):
  IF gate_id IN [G-07, G-14] AND severity == CRITICAL:
    → build.fail(reason: "Non-remediable critical finding")
    → EMIT sdx.build.failed
    → RETURN (no remediation for critical security/secrets)

  attempts = build.remediation_count(gate_id)
  IF attempts >= 3:
    → EMIT sdx.remediation.exhausted
    → trigger_escalation_policy(build_id, gate_id)
    → RETURN

  responsible_agent = gate_to_agent_mapping[gate_id]
  → assign_agent(responsible_agent, build_id, gate_id)
  → EMIT sdx.remediation.started(attempt = attempts + 1)
```

**Gate-to-Agent Mapping:**

| Gate | Responsible Agent | Remediation Action |
|------|-------------------|-------------------|
| G-01 CodeBLEU | BuilderAgent | Refactor code to improve quality score |
| G-02 Complexity | BuilderAgent | Decompose high-complexity functions |
| G-03 Duplication | RefactorAgent | Extract common abstractions |
| G-04 Bug Density | SecurityAgent + BuilderAgent | Fix detected bugs |
| G-05 Coverage | TestAgent + BuilderAgent | Generate missing tests, fix untestable code |
| G-06 Mutation | TestAgent | Strengthen weak assertions |
| G-07 Security | SecurityAgent | Remediate vulnerabilities (if HIGH; CRITICAL fails build) |
| G-08 SBOM | DevOpsAgent | Add missing component declarations |
| G-09 License | BuilderAgent | Swap non-compliant dependency for compliant alternative |
| G-10 Compliance | ComplianceAgent | Resolve control gaps |
| G-11 Performance | PerformanceAgent + BuilderAgent | Optimise hot paths |
| G-12 Docs | DocAgent | Fill documentation gaps |
| G-13 Type Safety | BuilderAgent | Fix type errors |
| G-14 Secrets | SecurityAgent | Remove secret, inject Vault reference (CRITICAL = fail) |

---

## 2. Model Fallback Policy

**Trigger:** LLM API response with status >= 500, timeout > 3s, or circuit breaker OPEN
**Scope:** Agent Aggregate via sdx-llm-gateway
**Logic:**
```
WHEN llm_request_failed(agent_id, model_id, error):
  circuit = get_circuit_breaker(model_id)
  circuit.record_failure()

  IF circuit.state == OPEN:
    fallback = agent.model_config.fallback_model
    IF fallback circuit is CLOSED:
      → route_to(fallback)
      → EMIT sdx.agent.model.fallback
    ELSE:
      → check on_prem_models
      IF on_prem available:
        → route_to(on_prem)
      ELSE:
        → queue request (max 5 min)
        → IF still unavailable after 5 min:
          → FAIL with SDX-6001 ALL_MODELS_UNAVAILABLE

  IF circuit.consecutive_failures >= 3:
    circuit.state = OPEN
    circuit.cooldown = 30s
    → EMIT sdx.llm.circuit.opened
```

**Circuit Breaker Parameters:**

| Parameter | Value |
|-----------|-------|
| Failure threshold | 3 consecutive failures |
| Cooldown period | 30 seconds |
| Half-open probe | 1 request |
| Success threshold to close | 2 consecutive successes |
| Max queue wait | 5 minutes |

---

## 3. Checkpoint Policy

**Trigger:** Timer (every 30 seconds during active build processing)
**Scope:** Build Aggregate
**Logic:**
```
EVERY 30s WHILE build.status IN [active_statuses]:
  state_snapshot = serialize(build)
  hash = sha256(state_snapshot)
  persist_to_nats_jetstream(build_id, state_snapshot, hash)
  build.checkpoint_hash = hash
  build.checkpointed_at = now()
  → EMIT sdx.build.checkpointed
```

**Implementation:** Temporal workflow heartbeat carries checkpoint data.

---

## 4. Quota Enforcement Policy

**Trigger:** Any build or resource creation request
**Scope:** Tenant Aggregate
**Logic:**
```
WHEN request_resource(tenant_id, resource_type):
  tenant = load_tenant(tenant_id)

  IF tenant.status != ACTIVE:
    → REJECT with SDX-3001 TENANT_SUSPENDED

  meter = tenant.billing_meters[resource_type]
  IF meter.limit > 0 AND meter.current_value >= meter.limit:
    → EMIT sdx.tenant.quota.exhausted
    → REJECT with SDX-3002 QUOTA_EXHAUSTED

  meter.current_value += 1
  → save_tenant(tenant)
```

**Resource Type Quotas:**

| Resource | STARTER | GROWTH | ENTERPRISE | SOVEREIGN |
|----------|---------|--------|------------|-----------|
| Pipelines/month | 5 | 30 | Unlimited | Unlimited |
| Languages | 3 (Go, Python, TS) | 12 | 12 | 12 |
| Storage GB | 10 | 100 | Unlimited | Unlimited |
| Concurrent agents | 2 | 10 | Unlimited | Unlimited |

---

## 5. Zero Software Tax Policy

**Trigger:** Dependency added during code generation (G-09 gate)
**Scope:** Code Generation + Quality Verification
**Logic:**
```
WHEN dependency_added(package_name, version):
  license = resolve_license(package_name, version)

  IF license NOT IN [APACHE_2_0, MIT, MPL_2_0, BSD_2_CLAUSE, BSD_3_CLAUSE, ISC, UNLICENSE]:
    → REJECT dependency
    → instruct BuilderAgent to find alternative with compliant license
    → EMIT sdx.gate.failed(G-09)
    → RETURN

  IF license == UNKNOWN or license == PROPRIETARY:
    → REJECT dependency
    → escalate for manual review
```

**Allowed Licenses (exhaustive):**
- Apache-2.0
- MIT
- MPL-2.0
- BSD-2-Clause
- BSD-3-Clause
- ISC
- Unlicense
- CC0-1.0 (for data/docs only)
- PostgreSQL (for database drivers)

**Explicitly Forbidden:**
- GPL-2.0, GPL-3.0
- AGPL-3.0
- LGPL-2.1, LGPL-3.0 (borderline; forbidden to avoid linking risk)
- SSPL
- BSL (except Vault, evaluated and excepted per ADR-0011)
- Any commercial/proprietary license

---

## 6. Secret Scan Policy

**Trigger:** Code artifact generated (G-14 gate)
**Scope:** Security Scanning
**Logic:**
```
WHEN code_generated(build_id, code_artifact_ref):
  findings = run_gitleaks(code_artifact_ref)
  findings += run_custom_patterns(code_artifact_ref)

  IF findings.count > 0:
    FOR each finding:
      classify(finding) → HIGH_ENTROPY_STRING | KNOWN_PATTERN | API_KEY | PASSWORD | CERTIFICATE

    → EMIT sdx.gate.failed(G-14, count=findings.count)

    IF any finding is CERTAIN (known pattern match):
      → build.fail() — non-remediable
    ELSE:
      → trigger remediation: SecurityAgent removes finding, injects Vault reference
```

**Custom Patterns (in addition to Gitleaks defaults):**

| Pattern | Type | Severity |
|---------|------|----------|
| `sk-[a-zA-Z0-9]{32,}` | OpenAI API key | CRITICAL |
| `sk-ant-[a-zA-Z0-9-]{32,}` | Anthropic API key | CRITICAL |
| `AIza[0-9A-Za-z_-]{35}` | Google API key | CRITICAL |
| `ghp_[a-zA-Z0-9]{36}` | GitHub personal access token | CRITICAL |
| `PRIVATE KEY-----` | Private key | CRITICAL |
| `password\s*[:=]\s*["'][^"']+["']` | Hardcoded password | HIGH |
| `[0-9a-f]{64}` (in non-hash context) | High-entropy hex string | MEDIUM |

---

## 7. Escalation Policy

**Trigger:** 5 consecutive recovery failures OR 3 remediation exhaustions
**Scope:** Build Aggregate
**Logic:**
```
WHEN recovery_failed(build_id, attempt=5) OR remediation_exhausted(build_id, gate_id, attempts=3):
  context = collect_diagnostic_context(build_id):
    - build state snapshot
    - all gate results
    - agent logs (last 100 lines per agent)
    - checkpoint history
    - error messages
    - trace URL (Grafana Tempo)

  incident = create_pagerduty_incident(
    severity: "high",
    title: "SDX Build {build_id} requires human intervention",
    body: context,
    service: "sdx-workflow-engine"
  )

  build.status = FAILED
  → EMIT sdx.build.escalation.triggered(incident_id)
  → EMIT sdx.build.failed(reason="human_escalation")
  → notify_webhook(tenant_id, "build.failed")
```

---

## 8. Data Residency Policy

**Trigger:** Any data storage or processing operation
**Scope:** All bounded contexts
**Logic:**
```
WHEN store_data(tenant_id, data, target_store):
  tenant = load_tenant(tenant_id)
  region = tenant.nexo_region

  IF tenant compliance_profiles contains GDPR:
    IF region NOT IN [LONDON]:
      → REJECT with "GDPR data must remain in EU region"

  ensure target_store is in region
  ensure no cross-region replication of PII (except pseudonymised audit logs)
```

---

## 9. Idempotency Policy

**Trigger:** Any mutating API request to sdx-gateway
**Scope:** API Gateway
**Logic:**
```
WHEN mutating_request(idempotency_key, tenant_id):
  IF idempotency_key is present:
    cached = dragonfly.get("idempotency:{tenant_id}:{idempotency_key}")
    IF cached exists:
      → return cached response (HTTP 200, not 409)

  execute_request()
  response = get_response()
  dragonfly.set("idempotency:{tenant_id}:{idempotency_key}", response, TTL=24h)
  → return response
```

---

## 10. Webhook Delivery Policy

**Trigger:** Webhook-eligible domain event
**Scope:** API Gateway
**Logic:**
```
WHEN webhook_event(tenant_id, event_type, payload):
  configs = webhook_repo.find_by_event(tenant_id, event_type)

  FOR each config:
    signature = hmac_sha256(config.secret, payload)
    attempt = 1

    WHILE attempt <= 5:
      response = http_post(config.target_url, payload, headers={
        "X-SDX-Signature": signature,
        "X-SDX-Event": event_type,
        "X-SDX-Delivery": ulid()
      })

      IF response.status IN [200..299]:
        → EMIT sdx.webhook.delivered
        → BREAK

      wait = exponential_backoff(attempt, base=5s, max=1h)
      → EMIT sdx.webhook.failed(will_retry=true)
      attempt += 1

    IF attempt > 5:
      → EMIT sdx.webhook.dead_letter
```

**Retry Schedule:**
| Attempt | Delay |
|---------|-------|
| 1 | Immediate |
| 2 | 5s |
| 3 | 25s |
| 4 | 2min |
| 5 | 15min |
| Dead letter | After 5 failures |

---

## Validation Rules (Synchronous)

### WorkItem Validations

| Rule | Field | Condition | Error |
|------|-------|-----------|-------|
| V-WI-01 | source.raw_content | Not empty, ≤ 5MB | SDX-2001 INVALID_INPUT |
| V-WI-02 | source.type | Must be valid InputSourceType | SDX-2001 INVALID_INPUT |
| V-WI-03 | manifest.languages | Each must be a valid LanguageCode | SDX-2002 UNSUPPORTED_LANGUAGE |
| V-WI-04 | manifest.languages | ≥ 1 language | SDX-2001 INVALID_INPUT |
| V-WI-05 | manifest.compliance_frameworks | Each must be valid ComplianceFramework | SDX-2008 UNKNOWN_FRAMEWORK |
| V-WI-06 | manifest.languages | Must be within tenant's plan allowance | SDX-3002 QUOTA_EXCEEDED |

### Tenant Validations

| Rule | Field | Condition | Error |
|------|-------|-----------|-------|
| V-T-01 | name | Not empty, 3-100 chars, unique | SDX-2010 INVALID_TENANT_NAME |
| V-T-02 | plan | Must be valid PlanTier | SDX-2003 INVALID_PLAN |
| V-T-03 | nexo_region | Must be valid NexoRegion | SDX-2004 INVALID_REGION |
| V-T-04 | billing_account_ref | Not empty | SDX-2011 MISSING_BILLING |

### Webhook Validations

| Rule | Field | Condition | Error |
|------|-------|-----------|-------|
| V-WH-01 | target_url | Must be HTTPS | SDX-2006 INVALID_URL |
| V-WH-02 | target_url | Must resolve to public IP (no internal) | SDX-2006 INVALID_URL |
| V-WH-03 | events | Each must be known webhook event type | SDX-2007 UNKNOWN_EVENT_TYPE |
| V-WH-04 | events | ≥ 1 event | SDX-2007 UNKNOWN_EVENT_TYPE |
| V-WH-05 | secret | ≥ 32 bytes for HMAC | SDX-2012 WEAK_SECRET |

### Agent Validations

| Rule | Field | Condition | Error |
|------|-------|-----------|-------|
| V-AG-01 | type | Must be valid AgentType | SDX-2013 INVALID_AGENT_TYPE |
| V-AG-02 | spiffe_id | Must be valid SPIFFE format | SDX-2014 INVALID_SPIFFE_ID |
| V-AG-03 | model_config.primary | Must be valid ModelID | SDX-2015 INVALID_MODEL |
| V-AG-04 | model_config.primary | Must differ from fallback | SDX-2016 SAME_MODEL_FALLBACK |
