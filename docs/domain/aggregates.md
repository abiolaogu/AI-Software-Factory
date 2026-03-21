# NeuroForge SDX — Aggregates with Invariants

> Each aggregate enforces its own consistency boundary.
> Invariants are checked on every state mutation. Violations throw domain errors, not infrastructure errors.

---

## 1. WorkItem Aggregate

**Root Entity:** WorkItem
**Identity:** WorkItemID (ULID)
**Bounded Context:** BC-01 WorkItem Ingestion

### Fields

| Field | Type | Required | Mutable |
|-------|------|----------|---------|
| id | WorkItemID | yes | no |
| tenant_id | TenantID | yes | no |
| status | WorkItemStatus | yes | yes |
| source | InputSource | yes | no |
| manifest | WorkItemManifest | no (set after parsing) | yes (until APPROVED) |
| requirements | []Requirement | yes (≥1 after PARSED) | yes (until MODELLING) |
| ambiguities | []Ambiguity | no | yes (until all resolved) |
| assigned_agents | []AgentAssignment | no | yes |
| created_at | Timestamp | yes | no |
| updated_at | Timestamp | yes | yes |

### Enums

```
WorkItemStatus: DRAFT → PARSED → MANIFEST_REVIEW → APPROVED → MODELLING → COMPLETED → FAILED
InputSourceType: PRD | USER_STORY | API_SPEC | CODEBASE | BUG_REPORT
Priority: P0 | P1 | P2
```

### Invariants

| # | Invariant | Enforcement Point |
|---|-----------|-------------------|
| WI-INV-01 | WorkItem MUST have ≥ 1 Requirement before transitioning to PARSED | `MarkAsParsed()` |
| WI-INV-02 | All Ambiguities MUST be resolved (resolution != nil) before transitioning to APPROVED | `ApproveManifest()` |
| WI-INV-03 | Manifest MUST specify ≥ 1 language and ≥ 1 compliance framework before APPROVED | `ApproveManifest()` |
| WI-INV-04 | Status transitions must follow the state machine; no skipping | Every status mutation |
| WI-INV-05 | source is immutable after creation | Constructor only |
| WI-INV-06 | tenant_id is immutable after creation | Constructor only |
| WI-INV-07 | Manifest languages MUST be within tenant's allowed_languages | `SetManifest()` |
| WI-INV-08 | A FAILED WorkItem cannot transition to any state except DRAFT (restart) | Status mutation |

---

## 2. Build Aggregate

**Root Entity:** Build
**Identity:** BuildID (ULID)
**Bounded Context:** Build Orchestration (cross-cutting; orchestrated by sdx-workflow-engine)

### Fields

| Field | Type | Required | Mutable |
|-------|------|----------|---------|
| id | BuildID | yes | no |
| work_item_id | WorkItemID | yes | no |
| tenant_id | TenantID | yes | no |
| status | BuildStatus | yes | yes |
| current_stage | StageNumber | yes | yes |
| stages | []PipelineStageRecord | yes | yes (append-only) |
| gate_results | []GateResult | no | yes (append-only) |
| remediation_loops | []RemediationRecord | no | yes (append-only) |
| checkpoint_hash | ContentHash | no | yes |
| checkpointed_at | Timestamp | no | yes |
| recovery_attempts | uint8 | yes (default 0) | yes |
| domain_model_ref | string | no | yes (set at stage 2) |
| blueprint_ref | string | no | yes (set at stage 3) |
| contract_refs | []string | no | yes (append) |
| artifact_refs | []ArtifactRef | no | yes (append) |
| created_at | Timestamp | yes | no |
| completed_at | Timestamp | no | yes |

### Enums

```
BuildStatus: INITIATED → DOMAIN_MODELLING → ARCHITECTING → CONTRACTING → TESTING_RED →
             CODING_GREEN → VERIFYING → CERTIFYING → SIGNING → DEPLOYING → COMPLETED | FAILED | RECOVERING
```

### Invariants

| # | Invariant | Enforcement Point |
|---|-----------|-------------------|
| B-INV-01 | Build MUST NOT advance past a stage with a failing BLOCKING gate | `AdvanceStage()` |
| B-INV-02 | checkpoint_hash MUST be refreshed within 30s during active processing | `Checkpoint()` — enforced by Temporal heartbeat |
| B-INV-03 | recovery_attempts MUST NOT exceed 5; at 5, Human Escalation is triggered | `RecoverFromCheckpoint()` |
| B-INV-04 | Every Build MUST produce an SBOM artifact before entering SIGNING stage (stage 14) | `AdvanceStage(14)` |
| B-INV-05 | Gate G-07 (Security) with CRITICAL finding blocks all forward progress; no override | `RecordGateResult(G-07)` |
| B-INV-06 | Gate G-14 (Secrets) with any detection blocks all forward progress; no override | `RecordGateResult(G-14)` |
| B-INV-07 | Build in COMPLETED state is immutable; no further stage or gate changes | All mutations |
| B-INV-08 | Build in FAILED state can only transition to RECOVERING | Status mutation |
| B-INV-09 | stages is append-only; past stage records MUST NOT be modified | `CompleteStage()` |
| B-INV-10 | current_stage MUST equal stages.length (no gaps) | `AdvanceStage()` |
| B-INV-11 | work_item_id and tenant_id are immutable after creation | Constructor only |
| B-INV-12 | Remediation loop for a gate MUST NOT exceed 3 attempts | `StartRemediation()` |

### Entities within Build

#### PipelineStageRecord
```
Fields: stage_number, name, status (PENDING|RUNNING|COMPLETED|FAILED|SKIPPED),
        started_at, completed_at, agent_id, output_ref, error_message
```

#### RemediationRecord
```
Fields: gate_id, attempt (1..3), agent_id, action_taken, result (FIXED|STILL_FAILING),
        started_at, completed_at
```

---

## 3. Artifact Aggregate

**Root Entity:** Artifact
**Identity:** ArtifactID (ULID)
**Bounded Context:** BC-13 Artifact Registry

### Fields

| Field | Type | Required | Mutable |
|-------|------|----------|---------|
| id | ArtifactID | yes | no |
| build_id | BuildID | yes | no |
| tenant_id | TenantID | yes | no |
| type | ArtifactType | yes | no |
| content_hash | ContentHash | yes | no |
| storage_uri | S3URI | yes | no |
| signature_ref | SignatureRef | no | yes (set once) |
| provenance_ref | ProvenanceRef | no | yes (set once) |
| sbom_ref | SBOMRef | no | yes (set once) |
| status | ArtifactStatus | yes | yes |
| size_bytes | uint64 | yes | no |
| created_at | Timestamp | yes | no |
| signed_at | Timestamp | no | yes |
| published_at | Timestamp | no | yes |

### Enums

```
ArtifactType: SOURCE_CODE | BINARY | CONTAINER_IMAGE | HELM_CHART | TERRAFORM_MODULE |
              SBOM | COMPLIANCE_PACKAGE | DOCS | TEST_REPORT | SECURITY_REPORT | PERF_REPORT
ArtifactStatus: CREATED → SIGNED → PUBLISHED → DEPLOYED | REVOKED
```

### Invariants

| # | Invariant | Enforcement Point |
|---|-----------|-------------------|
| A-INV-01 | content_hash MUST be computed at creation and NEVER change | Constructor |
| A-INV-02 | Artifact MUST NOT be SIGNED without a valid SignatureRef | `Sign()` |
| A-INV-03 | Artifact MUST NOT be PUBLISHED without status == SIGNED | `Publish()` |
| A-INV-04 | Artifact MUST NOT be PUBLISHED without an associated SBOM (sbom_ref != nil) | `Publish()` |
| A-INV-05 | ProvenanceRef MUST satisfy SLSA Level 3 (slsa_level == 3) | `AttachProvenance()` |
| A-INV-06 | REVOKED artifacts MUST NOT transition to any state except REVOKED | Status mutation |
| A-INV-07 | storage_uri MUST include tenant_id in the path | Constructor |
| A-INV-08 | signature_ref is write-once; cannot be overwritten | `Sign()` |

---

## 4. Tenant Aggregate

**Root Entity:** Tenant
**Identity:** TenantID (ULID)
**Bounded Context:** BC-16 Tenant Management

### Fields

| Field | Type | Required | Mutable |
|-------|------|----------|---------|
| id | TenantID | yes | no |
| name | string | yes | yes |
| plan | PlanTier | yes | yes |
| status | TenantStatus | yes | yes |
| nexo_region | NexoRegion | yes | no (migration = new tenant) |
| compliance_profiles | []ComplianceProfile | no | yes |
| quotas | TenantQuotas | yes | yes (derived from plan) |
| workspaces | []Workspace | no | yes |
| billing_meters | []BillingMeter | yes | yes |
| secret_vault_ref | VaultMountPath | yes | no |
| namespace_ref | K8sNamespace | yes | no |
| billing_account_ref | string | yes | yes |
| created_at | Timestamp | yes | no |
| updated_at | Timestamp | yes | yes |

### Enums

```
TenantStatus: PROVISIONING → ACTIVE → SUSPENDED → DEPROVISIONED
```

### Invariants

| # | Invariant | Enforcement Point |
|---|-----------|-------------------|
| T-INV-01 | Tenant MUST NOT exceed plan quotas; new builds queued when exhausted | `ConsumeQuota()` |
| T-INV-02 | STARTER tenants limited to GO, PYTHON, TYPESCRIPT languages | `ValidateLanguageAccess()` |
| T-INV-03 | SOVEREIGN tenants MUST have on_prem deployment; cloud LLM routing disabled | `ProvisionTenant()` |
| T-INV-04 | ComplianceProfile frameworks MUST match plan tier allowances | `SetComplianceProfile()` |
| T-INV-05 | SUSPENDED tenant cannot create new WorkItems or Builds | `ValidateTenantActive()` |
| T-INV-06 | DEPROVISIONED tenant data retained for 30 days, then purged | Background job |
| T-INV-07 | Workspace members MUST be authenticated via Authentik; max members per plan | `AddWorkspaceMember()` |
| T-INV-08 | nexo_region immutable — region migration requires new tenant provisioning | Constructor |
| T-INV-09 | BillingMeters reset at period_end; current_value resets to 0 | `ResetBillingPeriod()` |
| T-INV-10 | Plan downgrade cannot reduce quotas below current usage | `ChangePlan()` |

### Entities within Tenant

#### Workspace
```
Fields: id (WorkspaceID), name, members ([]WorkspaceMember), settings (WorkspaceSettings), created_at
```

#### WorkspaceMember
```
Fields: user_id (from Authentik), role (OWNER|ADMIN|MEMBER|VIEWER), joined_at
```

#### ComplianceProfile
```
Fields: id, frameworks ([]ComplianceFramework enum), controls ([]string), auto_evidence (bool)
ComplianceFramework: SOC2_TYPE_II | HIPAA | GDPR | ISO_27001 | PCI_DSS_V4 | NIST_800_53 | OWASP_TOP_10 | SLSA_LEVEL_3
```

---

## 5. Agent Aggregate

**Root Entity:** Agent
**Identity:** AgentID (ULID)
**Bounded Context:** BC-17 Agent Orchestration

### Fields

| Field | Type | Required | Mutable |
|-------|------|----------|---------|
| id | AgentID | yes | no |
| type | AgentType | yes | no |
| status | AgentStatus | yes | yes |
| spiffe_id | SPIFFEID | yes | no |
| model_config | ModelRoutingConfig | yes | yes |
| health | HealthStatus | yes | yes |
| current_build_id | BuildID | no | yes |
| current_stage | StageNumber | no | yes |
| heartbeat_at | Timestamp | yes | yes |
| pod_name | string | yes | no |
| nexo_region | NexoRegion | yes | no |
| created_at | Timestamp | yes | no |

### Enums

```
AgentType: ARCHITECT | BUILDER | VERIFIER | SPEC | DOMAIN | TEST | SECURITY | COMPLIANCE |
           DEVOPS | DOC | REFACTOR | PERFORMANCE | LOCALISATION | MIGRATION | OBSERVABILITY | LEARNING
AgentStatus: REGISTERED → IDLE → ASSIGNED → PROCESSING → RECOVERING → UNHEALTHY → TERMINATED
```

### Invariants

| # | Invariant | Enforcement Point |
|---|-----------|-------------------|
| AG-INV-01 | Agent MUST heartbeat every 10s; 3 missed → UNHEALTHY | `RecordHeartbeat()` / background monitor |
| AG-INV-02 | Agent MUST hold valid SPIFFE SVID; expired → TERMINATED | SPIRE rotation callback |
| AG-INV-03 | Agent in PROCESSING MUST have non-null current_build_id | `AssignToBuild()` |
| AG-INV-04 | Agent in IDLE MUST have null current_build_id | `ReleaseFromBuild()` |
| AG-INV-05 | Model fallback MUST trigger within 3s of primary model failure | `RouteModelRequest()` |
| AG-INV-06 | TERMINATED agent cannot be reassigned; must re-register | Status mutation |
| AG-INV-07 | Only one Build assignment at a time per Agent | `AssignToBuild()` |
| AG-INV-08 | Agent type is immutable after registration | Constructor |

---

## 6. DomainModel Aggregate

**Root Entity:** DomainModel
**Identity:** DomainModelID (ULID)
**Bounded Context:** BC-02 Domain Modelling

### Fields

| Field | Type | Required | Mutable |
|-------|------|----------|---------|
| id | DomainModelID | yes | no |
| work_item_id | WorkItemID | yes | no |
| tenant_id | TenantID | yes | no |
| bounded_contexts | []BoundedContextDef | yes (≥1) | yes |
| aggregates | []AggregateDef | yes (≥1) | yes |
| entities | []EntityDef | no | yes |
| value_objects | []ValueObjectDef | no | yes |
| domain_events | []DomainEventDef | no | yes |
| ubiquitous_language | map[string]string | yes (≥10 terms) | yes |
| completeness_score | float64 (0..100) | yes | yes |
| status | DomainModelStatus | yes | yes |
| version | uint32 | yes | yes |
| created_at | Timestamp | yes | no |

### Invariants

| # | Invariant |
|---|-----------|
| DM-INV-01 | DomainModel MUST have ≥ 1 BoundedContextDef before READY |
| DM-INV-02 | DomainModel MUST have ≥ 1 AggregateDef before READY |
| DM-INV-03 | ubiquitous_language MUST have ≥ 10 terms before READY |
| DM-INV-04 | completeness_score ≥ 95.0 required for APPROVED status |
| DM-INV-05 | version increments monotonically on every update |

---

## 7. Blueprint Aggregate

**Root Entity:** Blueprint
**Identity:** BlueprintID (ULID)
**Bounded Context:** BC-03 Architecture Design

### Fields

| Field | Type | Required | Mutable |
|-------|------|----------|---------|
| id | BlueprintID | yes | no |
| work_item_id | WorkItemID | yes | no |
| domain_model_id | DomainModelID | yes | no |
| tenant_id | TenantID | yes | no |
| service_definitions | []ServiceDefinition | yes (≥1) | yes |
| tech_stack | TechStackSelection | yes | yes |
| adrs | []ADRRecord | no | yes |
| status | BlueprintStatus | yes | yes |
| version | uint32 | yes | yes |
| created_at | Timestamp | yes | no |

### Invariants

| # | Invariant |
|---|-----------|
| BP-INV-01 | Blueprint MUST have ≥ 1 ServiceDefinition before APPROVED |
| BP-INV-02 | tech_stack.languages MUST be within tenant's allowed_languages |
| BP-INV-03 | Blueprint MUST have ≥ 1 ADR before APPROVED |
| BP-INV-04 | All ServiceDefinitions must have unique names |
| BP-INV-05 | version increments monotonically on every update |

---

## 8. Contract Aggregate

**Root Entity:** Contract
**Identity:** ContractID (ULID)
**Bounded Context:** BC-04 Contract Generation

### Fields

| Field | Type | Required | Mutable |
|-------|------|----------|---------|
| id | ContractID | yes | no |
| blueprint_id | BlueprintID | yes | no |
| tenant_id | TenantID | yes | no |
| type | ContractType (OPENAPI | ASYNCAPI | GRPC) | yes | no |
| spec_content | []byte | yes | yes (versioned) |
| spec_hash | ContentHash | yes | yes |
| version | string (semver) | yes | yes |
| status | ContractStatus (DRAFT | VALIDATED | APPROVED) | yes | yes |
| consumer_validations | []ConsumerValidation | no | yes |
| created_at | Timestamp | yes | no |

### Invariants

| # | Invariant |
|---|-----------|
| C-INV-01 | spec_hash MUST match SHA-256 of spec_content |
| C-INV-02 | Contract MUST pass spectral lint before VALIDATED |
| C-INV-03 | Contract MUST NOT have breaking changes vs previous version (oasdiff) |
| C-INV-04 | APPROVED contract is immutable; new changes require new version |

---

## 9. Deployment Aggregate

**Root Entity:** Deployment
**Identity:** DeploymentID (ULID)
**Bounded Context:** BC-14 Deployment Orchestration

### Fields

| Field | Type | Required | Mutable |
|-------|------|----------|---------|
| id | DeploymentID | yes | no |
| build_id | BuildID | yes | no |
| tenant_id | TenantID | yes | no |
| artifact_ids | []ArtifactID | yes | no |
| target_env | TargetEnvironment (DEV | STAGING | PROD) | yes | no |
| target_region | NexoRegion | yes | no |
| status | DeploymentStatus | yes | yes |
| smoke_test_results | []SmokeTestResult | no | yes |
| slo_validation | SLOValidation | no | yes |
| rollback_ref | DeploymentID | no | yes |
| deployed_at | Timestamp | no | yes |
| created_at | Timestamp | yes | no |

### Enums

```
DeploymentStatus: PENDING → APPLYING → SMOKE_TESTING → SLO_VALIDATING → DEPLOYED | ROLLED_BACK | FAILED
```

### Invariants

| # | Invariant |
|---|-----------|
| D-INV-01 | All artifact_ids MUST reference SIGNED artifacts |
| D-INV-02 | Deployment to PROD requires all smoke tests PASSED |
| D-INV-03 | SLO violation within 15 min of deployment triggers auto-rollback |
| D-INV-04 | ROLLED_BACK deployment records the rollback_ref (previous good deployment) |
| D-INV-05 | Only one active deployment per (tenant_id, service_name, target_env) |
