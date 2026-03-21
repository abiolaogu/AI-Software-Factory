# NeuroForge SDX — Domain Event Catalog

> Every domain event uses the DomainEventEnvelope (see value-objects.md).
> Events are published to NATS JetStream subjects. Inter-region events replicated via Redpanda.
> All events are immutable, append-only, and form the event-sourced backbone of the system.

---

## Event Naming Convention

```
Pattern: sdx.{aggregate}.{action}[.{qualifier}]
Examples:
  sdx.workitem.created
  sdx.build.stage.completed
  sdx.gate.failed
  sdx.artifact.signed
```

---

## WorkItem Events

| Event Type | NATS Subject | Payload Fields | Emitted By | Consumed By |
|-----------|-------------|----------------|------------|-------------|
| `sdx.workitem.created` | `sdx.workitem.created` | work_item_id, tenant_id, source_type, source_hash | sdx-ingestion-svc | sdx-dashboard |
| `sdx.workitem.parsed` | `sdx.workitem.parsed` | work_item_id, requirement_count, ambiguity_count | sdx-ingestion-svc | sdx-dashboard |
| `sdx.workitem.ambiguity.detected` | `sdx.workitem.ambiguity.detected` | work_item_id, ambiguity_id, question | sdx-ingestion-svc | sdx-dashboard |
| `sdx.workitem.ambiguity.resolved` | `sdx.workitem.ambiguity.resolved` | work_item_id, ambiguity_id, resolution, resolved_by | sdx-ingestion-svc | sdx-ingestion-svc |
| `sdx.workitem.manifest.approved` | `sdx.workitem.manifest.approved` | work_item_id, manifest (languages, frameworks, target_env) | sdx-ingestion-svc | sdx-domain-svc, sdx-workflow-engine |
| `sdx.workitem.failed` | `sdx.workitem.failed` | work_item_id, reason | sdx-ingestion-svc | sdx-dashboard, sdx-audit-svc |

---

## Build Lifecycle Events

| Event Type | NATS Subject | Payload Fields | Emitted By | Consumed By |
|-----------|-------------|----------------|------------|-------------|
| `sdx.build.initiated` | `sdx.build.initiated` | build_id, work_item_id, tenant_id | sdx-workflow-engine | sdx-dashboard, sdx-audit-svc |
| `sdx.build.stage.started` | `sdx.build.stage.started` | build_id, stage_number, stage_name, agent_id | sdx-workflow-engine | sdx-dashboard |
| `sdx.build.stage.completed` | `sdx.build.stage.completed` | build_id, stage_number, stage_name, output_ref, duration_ms | sdx-workflow-engine | sdx-dashboard, sdx-learning-svc |
| `sdx.build.stage.failed` | `sdx.build.stage.failed` | build_id, stage_number, stage_name, error_message, recovery_action | sdx-workflow-engine | sdx-dashboard, sdx-audit-svc |
| `sdx.build.checkpointed` | `sdx.build.checkpointed` | build_id, checkpoint_hash, stage_number, checkpointed_at | sdx-workflow-engine | sdx-audit-svc |
| `sdx.build.recovery.started` | `sdx.build.recovery.started` | build_id, checkpoint_hash, last_valid_stage, recovery_attempt | sdx-workflow-engine | sdx-dashboard, sdx-audit-svc |
| `sdx.build.recovery.succeeded` | `sdx.build.recovery.succeeded` | build_id, resumed_at_stage | sdx-workflow-engine | sdx-dashboard |
| `sdx.build.completed` | `sdx.build.completed` | build_id, artifact_ids[], compliance_badge, total_duration_ms | sdx-workflow-engine | sdx-dashboard, sdx-learning-svc, sdx-audit-svc, webhooks |
| `sdx.build.failed` | `sdx.build.failed` | build_id, stage_number, reason, escalation_triggered | sdx-workflow-engine | sdx-dashboard, sdx-audit-svc, webhooks |
| `sdx.build.escalation.triggered` | `sdx.build.escalation.triggered` | build_id, reason, context_url, pagerduty_incident_id | sdx-workflow-engine | sdx-audit-svc |

---

## Quality Gate Events

| Event Type | NATS Subject | Payload Fields | Emitted By | Consumed By |
|-----------|-------------|----------------|------------|-------------|
| `sdx.gate.evaluated` | `sdx.gate.evaluated` | build_id, gate_id, metric_name, value, threshold, passed | sdx-quality-svc | sdx-dashboard, sdx-learning-svc |
| `sdx.gate.passed` | `sdx.gate.passed` | build_id, gate_id, metric_name, value | sdx-quality-svc | sdx-workflow-engine |
| `sdx.gate.failed` | `sdx.gate.failed` | build_id, gate_id, metric_name, value, threshold, remediation_action | sdx-quality-svc | sdx-workflow-engine, sdx-dashboard, webhooks |
| `sdx.remediation.started` | `sdx.remediation.started` | build_id, gate_id, attempt, agent_id, action | sdx-workflow-engine | sdx-dashboard |
| `sdx.remediation.succeeded` | `sdx.remediation.succeeded` | build_id, gate_id, attempt, new_value | sdx-workflow-engine | sdx-dashboard, sdx-learning-svc |
| `sdx.remediation.exhausted` | `sdx.remediation.exhausted` | build_id, gate_id, attempts, last_value | sdx-workflow-engine | sdx-dashboard, sdx-audit-svc |

---

## Artifact Events

| Event Type | NATS Subject | Payload Fields | Emitted By | Consumed By |
|-----------|-------------|----------------|------------|-------------|
| `sdx.artifact.created` | `sdx.artifact.created` | artifact_id, build_id, type, content_hash, size_bytes | sdx-artifact-svc | sdx-dashboard |
| `sdx.artifact.signed` | `sdx.artifact.signed` | artifact_id, signature_ref, provenance_ref | sdx-artifact-svc | sdx-deploy-svc, sdx-dashboard, webhooks |
| `sdx.artifact.published` | `sdx.artifact.published` | artifact_id, registry_uri | sdx-artifact-svc | sdx-dashboard |
| `sdx.artifact.deployed` | `sdx.artifact.deployed` | artifact_id, deployment_id, environment, region | sdx-deploy-svc | sdx-dashboard, sdx-audit-svc |
| `sdx.artifact.revoked` | `sdx.artifact.revoked` | artifact_id, reason, revoked_by | sdx-artifact-svc | sdx-deploy-svc, sdx-audit-svc |

---

## Tenant Events

| Event Type | NATS Subject | Payload Fields | Emitted By | Consumed By |
|-----------|-------------|----------------|------------|-------------|
| `sdx.tenant.created` | `sdx.tenant.created` | tenant_id, name, plan, nexo_region | sdx-tenant-svc | sdx-audit-svc |
| `sdx.tenant.plan.upgraded` | `sdx.tenant.plan.upgraded` | tenant_id, old_plan, new_plan | sdx-tenant-svc | sdx-audit-svc, sdx-agent-orchestrator |
| `sdx.tenant.plan.downgraded` | `sdx.tenant.plan.downgraded` | tenant_id, old_plan, new_plan | sdx-tenant-svc | sdx-audit-svc |
| `sdx.tenant.suspended` | `sdx.tenant.suspended` | tenant_id, reason | sdx-tenant-svc | sdx-gateway, sdx-audit-svc |
| `sdx.tenant.reactivated` | `sdx.tenant.reactivated` | tenant_id | sdx-tenant-svc | sdx-gateway, sdx-audit-svc |
| `sdx.tenant.quota.exhausted` | `sdx.tenant.quota.exhausted` | tenant_id, meter_type, current_value, limit | sdx-tenant-svc | sdx-gateway, sdx-dashboard |
| `sdx.tenant.quota.reset` | `sdx.tenant.quota.reset` | tenant_id, meter_type, period_start | sdx-tenant-svc | sdx-dashboard |
| `sdx.tenant.workspace.created` | `sdx.tenant.workspace.created` | tenant_id, workspace_id, name | sdx-tenant-svc | sdx-audit-svc |
| `sdx.tenant.workspace.member.added` | `sdx.tenant.workspace.member.added` | tenant_id, workspace_id, user_id, role | sdx-tenant-svc | sdx-audit-svc |

---

## Agent Events

| Event Type | NATS Subject | Payload Fields | Emitted By | Consumed By |
|-----------|-------------|----------------|------------|-------------|
| `sdx.agent.registered` | `sdx.agent.registered` | agent_id, type, spiffe_id, pod_name, nexo_region | sdx-agent-orchestrator | sdx-dashboard |
| `sdx.agent.assigned` | `sdx.agent.assigned` | agent_id, build_id, stage_number | sdx-agent-orchestrator | sdx-dashboard |
| `sdx.agent.released` | `sdx.agent.released` | agent_id, build_id | sdx-agent-orchestrator | sdx-dashboard |
| `sdx.agent.heartbeat` | `sdx.agent.heartbeat` | agent_id, health_status, latency_ms | sdx-agent-orchestrator | (internal, high-freq) |
| `sdx.agent.unhealthy` | `sdx.agent.unhealthy` | agent_id, reason, consecutive_failures | sdx-agent-orchestrator | sdx-workflow-engine, sdx-dashboard |
| `sdx.agent.recovered` | `sdx.agent.recovered` | agent_id | sdx-agent-orchestrator | sdx-dashboard |
| `sdx.agent.terminated` | `sdx.agent.terminated` | agent_id, reason | sdx-agent-orchestrator | sdx-dashboard, sdx-audit-svc |
| `sdx.agent.model.fallback` | `sdx.agent.model.fallback` | agent_id, primary_model, fallback_model, reason | sdx-llm-gateway | sdx-dashboard, sdx-learning-svc |

---

## Security Events

| Event Type | NATS Subject | Payload Fields | Emitted By | Consumed By |
|-----------|-------------|----------------|------------|-------------|
| `sdx.security.scan.completed` | `sdx.security.scan.completed` | build_id, scan_type (SAST|DAST|SCA|SECRETS|IAC), findings_count, critical_count | sdx-security-svc | sdx-dashboard |
| `sdx.security.vulnerability.found` | `sdx.security.vulnerability.found` | build_id, cve_id, severity, package, version | sdx-security-svc | sdx-audit-svc |
| `sdx.security.sbom.generated` | `sdx.security.sbom.generated` | build_id, artifact_id, sbom_format, component_count | sdx-security-svc | sdx-artifact-svc |
| `sdx.security.cleared` | `sdx.security.cleared` | build_id, scan_types_passed[] | sdx-security-svc | sdx-artifact-svc, sdx-compliance-svc |

---

## Compliance Events

| Event Type | NATS Subject | Payload Fields | Emitted By | Consumed By |
|-----------|-------------|----------------|------------|-------------|
| `sdx.compliance.mapping.completed` | `sdx.compliance.mapping.completed` | build_id, framework, controls_mapped, controls_total | sdx-compliance-svc | sdx-dashboard |
| `sdx.compliance.evidence.generated` | `sdx.compliance.evidence.generated` | build_id, framework, evidence_package_ref | sdx-compliance-svc | sdx-artifact-svc |
| `sdx.compliance.certified` | `sdx.compliance.certified` | build_id, frameworks_passed[], badge | sdx-compliance-svc | sdx-artifact-svc, webhooks |
| `sdx.compliance.gap.detected` | `sdx.compliance.gap.detected` | build_id, framework, control_id, gap_description | sdx-compliance-svc | sdx-dashboard, sdx-audit-svc |

---

## Deployment Events

| Event Type | NATS Subject | Payload Fields | Emitted By | Consumed By |
|-----------|-------------|----------------|------------|-------------|
| `sdx.deployment.started` | `sdx.deployment.started` | deployment_id, build_id, target_env, target_region | sdx-deploy-svc | sdx-dashboard |
| `sdx.deployment.smoke.passed` | `sdx.deployment.smoke.passed` | deployment_id, test_count, duration_ms | sdx-deploy-svc | sdx-dashboard |
| `sdx.deployment.smoke.failed` | `sdx.deployment.smoke.failed` | deployment_id, failed_tests[], error | sdx-deploy-svc | sdx-dashboard |
| `sdx.deployment.slo.validated` | `sdx.deployment.slo.validated` | deployment_id, slo_results[] | sdx-deploy-svc | sdx-dashboard |
| `sdx.deployment.completed` | `sdx.deployment.completed` | deployment_id, build_id, deployed_at | sdx-deploy-svc | sdx-workflow-engine, sdx-dashboard, sdx-audit-svc |
| `sdx.deployment.rollback.triggered` | `sdx.deployment.rollback.triggered` | deployment_id, reason, rollback_to_deployment_id | sdx-deploy-svc | sdx-dashboard, sdx-audit-svc |
| `sdx.deployment.rollback.completed` | `sdx.deployment.rollback.completed` | deployment_id, rolled_back_to | sdx-deploy-svc | sdx-dashboard |

---

## LLM Gateway Events

| Event Type | NATS Subject | Payload Fields | Emitted By | Consumed By |
|-----------|-------------|----------------|------------|-------------|
| `sdx.llm.request.completed` | `sdx.llm.request.completed` | request_id, model, input_tokens, output_tokens, latency_ms, cost_microdollars | sdx-llm-gateway | sdx-learning-svc |
| `sdx.llm.circuit.opened` | `sdx.llm.circuit.opened` | model, reason, cooldown_seconds | sdx-llm-gateway | sdx-dashboard |
| `sdx.llm.circuit.closed` | `sdx.llm.circuit.closed` | model | sdx-llm-gateway | sdx-dashboard |

---

## Webhook Delivery Events (internal)

| Event Type | NATS Subject | Payload Fields | Emitted By | Consumed By |
|-----------|-------------|----------------|------------|-------------|
| `sdx.webhook.dispatched` | `sdx.webhook.dispatched` | webhook_id, tenant_id, event_type, target_url, attempt | sdx-gateway | sdx-audit-svc |
| `sdx.webhook.delivered` | `sdx.webhook.delivered` | webhook_id, status_code, duration_ms | sdx-gateway | sdx-audit-svc |
| `sdx.webhook.failed` | `sdx.webhook.failed` | webhook_id, status_code, error, will_retry | sdx-gateway | sdx-audit-svc |
| `sdx.webhook.dead_letter` | `sdx.webhook.dead_letter` | webhook_id, tenant_id, event_type, total_attempts | sdx-gateway | sdx-dashboard, sdx-audit-svc |

---

## Redpanda Inter-Regional Topics

| Topic | Event Types Replicated | Retention |
|-------|----------------------|-----------|
| `sdx.build.metrics` | build.completed, gate.evaluated, llm.request.completed, stage.completed | 90 days |
| `sdx.audit.events` | ALL events (full firehose) | 7 years (tiered to RustFS) |
| `sdx.compliance.evidence` | compliance.certified, compliance.evidence.generated | Infinite (compacted) |

---

## NATS JetStream Configuration

```yaml
streams:
  - name: SDX_WORKITEMS
    subjects: ["sdx.workitem.>"]
    retention: limits
    max_age: 30d
    replicas: 3
    storage: file
    dedup_window: 2m

  - name: SDX_BUILDS
    subjects: ["sdx.build.>", "sdx.gate.>", "sdx.remediation.>"]
    retention: limits
    max_age: 90d
    replicas: 3
    storage: file
    dedup_window: 2m

  - name: SDX_ARTIFACTS
    subjects: ["sdx.artifact.>"]
    retention: limits
    max_age: 365d
    replicas: 3
    storage: file
    dedup_window: 2m

  - name: SDX_TENANTS
    subjects: ["sdx.tenant.>"]
    retention: limits
    max_age: 365d
    replicas: 3
    storage: file
    dedup_window: 2m

  - name: SDX_AGENTS
    subjects: ["sdx.agent.>"]
    retention: limits
    max_age: 7d
    replicas: 3
    storage: file
    dedup_window: 2m

  - name: SDX_SECURITY
    subjects: ["sdx.security.>"]
    retention: limits
    max_age: 365d
    replicas: 3
    storage: file
    dedup_window: 2m

  - name: SDX_COMPLIANCE
    subjects: ["sdx.compliance.>"]
    retention: limits
    max_age: 2555d  # 7 years
    replicas: 3
    storage: file
    dedup_window: 2m

  - name: SDX_DEPLOYMENTS
    subjects: ["sdx.deployment.>"]
    retention: limits
    max_age: 365d
    replicas: 3
    storage: file
    dedup_window: 2m

  - name: SDX_LLM
    subjects: ["sdx.llm.>"]
    retention: limits
    max_age: 30d
    replicas: 3
    storage: file
    dedup_window: 2m

  - name: SDX_WEBHOOKS
    subjects: ["sdx.webhook.>"]
    retention: limits
    max_age: 30d
    replicas: 3
    storage: file
    dedup_window: 2m
```
