# NeuroForge SDX — Bounded Context Definitions

## BC-01: WorkItem Ingestion

**Owner Service:** `sdx-ingestion-svc`
**Aggregate Root:** WorkItem
**Responsibility:** Parse natural-language PRDs, user stories, API specs, and existing codebases. Disambiguate requirements. Produce canonical WorkItem manifests.
**Upstream:** SDX-as-a-Service API (ACL), Build Dashboard (read)
**Downstream:** Domain Modelling (event: `workitem.created`)
**Data Store:** YugabyteDB — `sdx_ingestion` schema
**Key Tables:** `work_items`, `requirements`, `ambiguities`
**Team Ownership:** Ingestion Squad

---

## BC-02: Domain Modelling

**Owner Service:** `sdx-domain-svc`
**Aggregate Root:** DomainModel
**Responsibility:** Extract bounded contexts, aggregates, entities, value objects, domain events from parsed WorkItems. Build ubiquitous language glossary.
**Upstream:** WorkItem Ingestion (event: `workitem.created`)
**Downstream:** Architecture Design (event: `domain.model.ready`)
**Data Store:** YugabyteDB — `sdx_domain` schema
**Key Tables:** `domain_models`, `bounded_contexts`, `aggregates`, `entities`, `value_objects`, `domain_events_catalog`, `ubiquitous_language`
**Team Ownership:** Domain Squad

---

## BC-03: Architecture Design

**Owner Service:** `sdx-architect-svc`
**Aggregate Root:** Blueprint
**Responsibility:** Select tech stack per language routing policy. Design system topology: services, databases, message queues, APIs. Generate ADRs.
**Upstream:** Domain Modelling (event: `domain.model.ready`)
**Downstream:** Contract Generation (event: `blueprint.approved`), Infrastructure Provisioning (event: `blueprint.approved`)
**Data Store:** YugabyteDB — `sdx_architecture` schema
**Key Tables:** `blueprints`, `adrs`, `service_definitions`, `tech_stack_selections`
**Team Ownership:** Architecture Squad

---

## BC-04: Contract Generation

**Owner Service:** `sdx-contract-svc`
**Aggregate Root:** Contract
**Responsibility:** Generate OpenAPI 3.1, AsyncAPI 3.0, and gRPC Protobuf contracts from Blueprints. Validate against existing consumers.
**Upstream:** Architecture Design (event: `blueprint.approved`)
**Downstream:** Test Specification (event: `contracts.generated`)
**Data Store:** YugabyteDB — `sdx_contracts` schema
**Key Tables:** `api_contracts`, `contract_versions`, `consumer_validations`
**Team Ownership:** Architecture Squad

---

## BC-05: Test Specification

**Owner Service:** `sdx-test-spec-svc`
**Aggregate Root:** TestSpec
**Responsibility:** Write BDD scenarios (Gherkin). Generate failing unit, integration, contract, and load test stubs. All tests in RED state.
**Upstream:** Contract Generation (event: `contracts.generated`)
**Downstream:** Code Generation (event: `tests.red.ready`)
**Data Store:** YugabyteDB + RustFS — `sdx_testspec` schema
**Key Tables:** `test_specs`, `bdd_scenarios`, `test_stubs`
**Team Ownership:** Quality Squad

---

## BC-06: Code Generation

**Owner Service:** `sdx-codegen-svc`
**Aggregate Root:** CodeArtifact
**Responsibility:** Implement production code per language routing policy to make all tests GREEN. Apply language-specific style, idioms, and DDD patterns.
**Upstream:** Test Specification (event: `tests.red.ready`)
**Downstream:** Quality Verification (event: `code.generated`), Documentation Generation (event: `code.generated`)
**Data Store:** RustFS — `/{tenant_id}/builds/{build_id}/code/`
**Team Ownership:** CodeGen Squad

---

## BC-07: Quality Verification

**Owner Service:** `sdx-quality-svc`
**Aggregate Root:** QualityReport
**Responsibility:** Execute SAST, CodeBLEU scoring, cyclomatic complexity analysis, duplication detection, coverage enforcement, and mutation testing.
**Upstream:** Code Generation (event: `code.generated`)
**Downstream:** Security Scanning (event: `quality.passed`), Compliance Certification (event: `quality.passed`), Performance Testing (event: `quality.passed`)
**Data Store:** ClickHouse — `sdx_quality` database
**Key Tables:** `quality_reports`, `gate_results`, `remediation_logs`
**Team Ownership:** Quality Squad

---

## BC-08: Security Scanning

**Owner Service:** `sdx-security-svc`
**Aggregate Root:** SecurityReport
**Responsibility:** DAST (OWASP ZAP), SCA (Grype), secrets scan (Gitleaks), IaC scan (Checkov), SBOM generation (Syft).
**Upstream:** Quality Verification (event: `quality.passed`)
**Downstream:** Artifact Registry (event: `security.cleared`)
**Data Store:** RustFS + ClickHouse
**Key Tables:** `security_reports`, `vulnerability_findings`, `sboms`
**Team Ownership:** Security Squad

---

## BC-09: Compliance Certification

**Owner Service:** `sdx-compliance-svc`
**Aggregate Root:** CompliancePackage
**Responsibility:** Map build outputs to compliance framework controls. Generate evidence packages. Produce compliance badges.
**Upstream:** Quality Verification (event: `quality.passed`)
**Downstream:** Artifact Registry (event: `compliance.certified`)
**Data Store:** YugabyteDB + RustFS — `sdx_compliance` schema
**Key Tables:** `compliance_frameworks`, `compliance_controls`, `control_mappings`, `evidence_packages`, `compliance_badges`
**Team Ownership:** Compliance Squad

---

## BC-10: Performance Testing

**Owner Service:** `sdx-performance-svc`
**Aggregate Root:** PerformanceReport
**Responsibility:** Run k6 load tests, profile memory/CPU, enforce p99 latency and memory budgets, generate flame graphs.
**Upstream:** Quality Verification (event: `quality.passed`)
**Downstream:** Build Orchestration (event: `performance.passed` or `performance.failed`)
**Data Store:** ClickHouse — `sdx_performance` database
**Key Tables:** `perf_reports`, `load_test_results`, `flame_graphs`
**Team Ownership:** Performance Squad

---

## BC-11: Documentation Generation

**Owner Service:** `sdx-docs-svc`
**Aggregate Root:** DocsArtifact
**Responsibility:** Generate README, API docs, architecture diagrams (Mermaid), runbooks, changelogs. Always in sync with build.
**Upstream:** Code Generation (event: `code.generated`)
**Downstream:** Artifact Registry (artifact upload)
**Data Store:** RustFS — `/{tenant_id}/builds/{build_id}/docs/`
**Team Ownership:** Docs Squad

---

## BC-12: Infrastructure Provisioning

**Owner Service:** `sdx-iac-svc`
**Aggregate Root:** IaCBundle
**Responsibility:** Generate Dockerfiles, Helm charts, Terraform modules, GitHub Actions workflows, Fleet GitOps configurations.
**Upstream:** Architecture Design (event: `blueprint.approved`)
**Downstream:** Artifact Registry (artifact upload)
**Data Store:** RustFS — `/{tenant_id}/builds/{build_id}/iac/`
**Team Ownership:** DevOps Squad

---

## BC-13: Artifact Registry

**Owner Service:** `sdx-artifact-svc`
**Aggregate Root:** Artifact
**Responsibility:** Content-addressing (SHA-256), Sigstore/Cosign signing, SLSA Level 3 provenance, SBOM association, artifact publication.
**Upstream:** Security Scanning (event: `security.cleared`), Compliance Certification (event: `compliance.certified`)
**Downstream:** Deployment Orchestration (event: `artifact.signed`)
**Data Store:** RustFS (artifacts) + YugabyteDB (metadata) — `sdx_artifacts` schema
**Key Tables:** `artifacts`, `signatures`, `provenance_attestations`
**Team Ownership:** Platform Squad

---

## BC-14: Deployment Orchestration

**Owner Service:** `sdx-deploy-svc`
**Aggregate Root:** Deployment
**Responsibility:** Apply artifacts to target environments via Fleet GitOps. Run smoke tests. Validate SLOs. Auto-rollback on failure.
**Upstream:** Artifact Registry (event: `artifact.signed`)
**Downstream:** Build Orchestration (event: `deployment.completed` or `deployment.failed`)
**Data Store:** YugabyteDB — `sdx_deployments` schema
**Key Tables:** `deployments`, `smoke_test_results`, `slo_validations`, `rollback_history`
**Team Ownership:** DevOps Squad

---

## BC-15: Learning & Optimisation

**Owner Service:** `sdx-learning-svc`
**Aggregate Root:** LearningCycle
**Responsibility:** Aggregate build metrics (CodeBLEU, bug escape rate, cycle time). Fine-tune agent prompt templates. Update model routing policies.
**Upstream:** All BCs (via Redpanda `sdx.build.metrics` topic)
**Downstream:** Agent Orchestration (updated routing policies)
**Data Store:** ClickHouse + Qdrant — `sdx_learning` database
**Key Tables:** `build_metrics`, `prompt_versions`, `routing_policies`, `agent_performance`
**Team Ownership:** ML Squad

---

## BC-16: Tenant Management

**Owner Service:** `sdx-tenant-svc`
**Aggregate Root:** Tenant
**Responsibility:** Tenant CRUD, workspace management, quota enforcement, billing meters, compliance profile configuration.
**Upstream:** SDX-as-a-Service API (ACL)
**Downstream:** All BCs (shared kernel: tenant_id propagation)
**Data Store:** YugabyteDB — `sdx_tenants` schema
**Key Tables:** `tenants`, `workspaces`, `workspace_members`, `quotas`, `billing_meters`, `compliance_profiles`
**Team Ownership:** Platform Squad

---

## BC-17: Agent Orchestration

**Owner Service:** `sdx-agent-orchestrator`
**Aggregate Root:** Agent
**Responsibility:** Agent lifecycle (register, assign, monitor, recover, terminate). Health monitoring. Model routing. Fallback policy execution.
**Upstream:** Build Orchestration (assignment requests)
**Downstream:** All Agent BCs (agent assignment), LLM Gateway (model selection)
**Data Store:** YugabyteDB + DragonflyDB — `sdx_agents` schema
**Key Tables:** `agents`, `agent_assignments`, `health_checks`, `model_routing_configs`
**Team Ownership:** Platform Squad

---

## BC-18: Observability Platform

**Owner Service:** External (Grafana Stack)
**Responsibility:** Collect traces (Tempo), metrics (Mimir), and logs (Loki) from all services via OpenTelemetry. Provide dashboards and alerting.
**Upstream:** All services (OTel gRPC push)
**Downstream:** Build Dashboard (read), Learning & Optimisation (metrics export)
**Data Store:** Tempo, Mimir, Loki (managed)
**Team Ownership:** SRE Squad

---

## BC-19: Build Dashboard

**Owner Service:** `sdx-dashboard`
**Responsibility:** Real-time web UI showing all active workflows, agent status, quality gate results, compliance badges, and artifact lineage graphs.
**Upstream:** All BCs (read-only via sdx-gateway)
**Data Store:** Read-model (DragonflyDB cache + ClickHouse queries)
**Team Ownership:** Frontend Squad

---

## BC-20: SDX-as-a-Service API

**Owner Service:** `sdx-gateway`
**Responsibility:** External REST/gRPC/GraphQL API for build job submission, status queries, artifact retrieval. Rate limiting. Auth. Tenant routing.
**Upstream:** External clients
**Downstream:** WorkItem Ingestion (ACL), all query services
**Data Store:** DragonflyDB (rate limit counters, idempotency cache)
**Team Ownership:** Platform Squad

---

## Context Relationship Types

| Relationship | Description |
|-------------|-------------|
| **Customer-Supplier** | Downstream depends on upstream's output; upstream accommodates downstream needs |
| **Conformist** | Downstream accepts upstream's model as-is; no translation |
| **Published Language** | Upstream publishes a well-defined, versioned contract (proto/OpenAPI) |
| **Shared Kernel** | Both contexts share a small, co-owned model (tenant context, event envelope) |
| **Anti-Corruption Layer (ACL)** | Downstream translates upstream's foreign model into its own domain language |
| **Open Host Service** | Upstream provides a standardised protocol for any consumer (OTel) |
