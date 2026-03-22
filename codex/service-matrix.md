# NeuroForge SDX — Service Matrix

> Definitive reference for Codex: what each service does, owns, and depends on.

---

## Implementation Order (Critical Path)

```
Phase 1 (M1): domain/shared → domain/workitem → domain/build → domain/artifact →
              domain/tenant → domain/agent → domain/deployment → domain/quality →
              domain/compliance → proto compilation

Phase 2 (M2): YugabyteDB adapters → ClickHouse adapters → DragonflyDB adapters →
              RustFS adapter → NATS adapter → sdx-gateway → sdx-ingestion-svc →
              sdx-tenant-svc → sdx-artifact-svc → sdx-quality-svc →
              sdx-llm-gateway → sdx-audit-svc → sdx-agent-orchestrator →
              sdx-deploy-svc → contract tests

Phase 3 (M3): BuildPipelineWorkflow → RemediationSubWorkflow → RecoveryWorkflow →
              Stage activities → Webhook delivery → E2E test
```

---

## Service Detail Matrix

| Service | BC | Port | Proto | Aggregate | Storage | NATS Streams | Dependencies |
|---------|-----|------|-------|-----------|---------|-------------|-------------|
| sdx-gateway | BC-20 | 8080 | gateway.proto | — | DragonflyDB | — | authentik, all domain svcs |
| sdx-ingestion-svc | BC-01 | 8081 | workitem.proto | WorkItem | YugabyteDB | SDX_WORKITEMS | sdx-llm-gateway |
| sdx-tenant-svc | BC-16 | 8082 | tenant.proto | Tenant | YugabyteDB | SDX_TENANTS | vault, k8s API |
| sdx-artifact-svc | BC-13 | 8083 | artifact.proto | Artifact | YugabyteDB + RustFS | SDX_ARTIFACTS | cosign, rekor |
| sdx-quality-svc | BC-07 | 8084 | quality.proto | QualityReport | ClickHouse | SDX_BUILDS (consume) | — |
| sdx-compliance-svc | BC-09 | 8085 | compliance.proto | CompliancePackage | YugabyteDB + RustFS | SDX_COMPLIANCE | — |
| sdx-agent-orchestrator | BC-17 | 8086 | agent.proto | Agent | YugabyteDB + DragonflyDB | SDX_AGENTS | spire |
| sdx-deploy-svc | BC-14 | 8087 | deploy.proto | Deployment | YugabyteDB | SDX_DEPLOYMENTS | fleet, k8s API |
| sdx-audit-svc | BC-audit | 8088 | audit.proto | AuditEntry | ClickHouse | SDX_BUILDS (consume all) | — |
| sdx-llm-gateway | BC-llm | 8089 | llm.proto | — | DragonflyDB | SDX_LLM | anthropic, openai, google, vllm |
| sdx-workflow-engine | Orch | 8090 | — | Build | YugabyteDB (via Temporal) | SDX_BUILDS (pub) | temporal, all activities |
| sdx-dashboard | BC-19 | 3000 | — | — | DragonflyDB (read) | — | sdx-gateway (read) |

---

## Database Schema Ownership

| Database | Schema | Owner Service | Tables |
|----------|--------|---------------|--------|
| YugabyteDB | sdx_ingestion | sdx-ingestion-svc | work_items, requirements, ambiguities |
| YugabyteDB | sdx_domain | sdx-domain-svc (activity) | domain_models, bounded_contexts, aggregates |
| YugabyteDB | sdx_architecture | sdx-architect-svc (activity) | blueprints, adrs, service_definitions |
| YugabyteDB | sdx_contracts | sdx-contract-svc (activity) | api_contracts, contract_versions |
| YugabyteDB | sdx_artifacts | sdx-artifact-svc | artifacts, signatures, provenance_attestations |
| YugabyteDB | sdx_compliance | sdx-compliance-svc | compliance_frameworks, controls, evidence_packages |
| YugabyteDB | sdx_tenants | sdx-tenant-svc | tenants, workspaces, members, quotas, billing_meters |
| YugabyteDB | sdx_agents | sdx-agent-orchestrator | agents, assignments, health_checks, routing_configs |
| YugabyteDB | sdx_deployments | sdx-deploy-svc | deployments, smoke_results, slo_validations |
| ClickHouse | sdx_quality | sdx-quality-svc | quality_reports, gate_results, remediation_logs |
| ClickHouse | sdx_performance | sdx-performance-svc (activity) | perf_reports, load_test_results |
| ClickHouse | sdx_audit | sdx-audit-svc | audit_entries |
| ClickHouse | sdx_llm | sdx-llm-gateway | llm_requests |
| ClickHouse | sdx_learning | sdx-learning-svc (activity) | build_metrics, prompt_versions |
| DragonflyDB | — | sdx-gateway | rate_limit:*, idempotency:*, agent_cache:* |
| RustFS | — | sdx-artifact-svc | /{tenant_id}/builds/{build_id}/* |

---

## Temporal Workflow Configuration

| Workflow | Task Queue | Retry Policy | Timeout | Heartbeat |
|----------|-----------|-------------|---------|-----------|
| BuildPipelineWorkflow | sdx-build-pipeline | MaxAttempts=1 (workflow-level recovery handled) | 24h | — |
| RemediationSubWorkflow | sdx-remediation | MaxAttempts=3 per gate | 30min | — |
| RecoveryWorkflow | sdx-recovery | MaxAttempts=5 | 1h | — |
| Stage Activities | sdx-stage-{n} | MaxAttempts=3, backoff=5s | 2h per stage | 10s |

---

## Inter-Service Communication Map

```
External Client
    │
    ▼
sdx-gateway (REST/gRPC) ──► sdx-ingestion-svc
    │                        sdx-tenant-svc
    │                        sdx-artifact-svc
    │                        sdx-quality-svc
    │                        sdx-compliance-svc
    │                        sdx-agent-orchestrator
    │                        sdx-deploy-svc
    │                        sdx-audit-svc
    │
    ▼
sdx-workflow-engine (Temporal) ──► [Stage Activities via Temporal]
    │                               ├── sdx-domain-svc (activity)
    │                               ├── sdx-architect-svc (activity)
    │                               ├── sdx-contract-svc (activity)
    │                               ├── sdx-test-spec-svc (activity)
    │                               ├── sdx-codegen-svc (activity)
    │                               ├── sdx-security-svc (activity)
    │                               ├── sdx-performance-svc (activity)
    │                               ├── sdx-docs-svc (activity)
    │                               ├── sdx-iac-svc (activity)
    │                               └── sdx-learning-svc (activity)
    │
    ▼
sdx-llm-gateway ──► Anthropic API
                     OpenAI API
                     Google AI API
                     vLLM (on-prem)
```
