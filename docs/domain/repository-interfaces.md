# NeuroForge SDX — Repository Interfaces

> All repositories follow the Repository pattern from DDD.
> Interfaces live in `domain/{aggregate}/repository.go`.
> Implementations live in `internal/adapter/{store}/{aggregate}_repo.go`.
> Every method takes `context.Context` as the first parameter (carries tenant_id, trace_id).
> Every mutating method is transactional within its aggregate boundary.

---

## Common Types

```go
package shared

import "time"

// Pagination — cursor-based, not offset-based
type Pagination struct {
    PageSize  int32  // Max 100, default 20
    PageToken string // Opaque cursor; empty for first page
}

type PaginatedResult[T any] struct {
    Items         []T
    NextPageToken string // Empty if last page
    TotalCount    int64
}

// SortOrder for list operations
type SortOrder string
const (
    SortAsc  SortOrder = "ASC"
    SortDesc SortOrder = "DESC"
)

// DateRange for temporal queries
type DateRange struct {
    Start time.Time
    End   time.Time
}
```

---

## WorkItemRepository

```go
package workitem

import (
    "context"
    "neuroforge-sdx/domain/shared"
)

type Repository interface {
    // Save persists a new or updated WorkItem. Upsert semantics.
    Save(ctx context.Context, item *WorkItem) error

    // FindByID loads a WorkItem by its ID. Returns ErrNotFound if absent.
    FindByID(ctx context.Context, id WorkItemID) (*WorkItem, error)

    // FindByTenant returns paginated WorkItems for a tenant, with optional filters.
    FindByTenant(ctx context.Context, tenantID TenantID, filter WorkItemFilter, page shared.Pagination) (*shared.PaginatedResult[*WorkItem], error)

    // FindByStatus returns WorkItems in a given status across all tenants (admin use).
    FindByStatus(ctx context.Context, status WorkItemStatus, page shared.Pagination) (*shared.PaginatedResult[*WorkItem], error)

    // CountByTenant returns the number of WorkItems for quota checking.
    CountByTenant(ctx context.Context, tenantID TenantID, filter WorkItemFilter) (int64, error)

    // Delete soft-deletes a WorkItem. Only allowed in DRAFT or FAILED status.
    Delete(ctx context.Context, id WorkItemID) error
}

type WorkItemFilter struct {
    Status     *WorkItemStatus // nil = all statuses
    SourceType *InputSourceType
    DateRange  *shared.DateRange
    Search     *string // Full-text search on requirements
}
```

---

## BuildRepository

```go
package build

import (
    "context"
    "time"
    "neuroforge-sdx/domain/shared"
)

type Repository interface {
    Save(ctx context.Context, build *Build) error
    FindByID(ctx context.Context, id BuildID) (*Build, error)
    FindByWorkItem(ctx context.Context, workItemID WorkItemID) ([]*Build, error)

    // FindActiveByTenant returns all non-terminal builds for a tenant.
    FindActiveByTenant(ctx context.Context, tenantID TenantID) ([]*Build, error)

    // FindByTenant returns paginated builds with filters.
    FindByTenant(ctx context.Context, tenantID TenantID, filter BuildFilter, page shared.Pagination) (*shared.PaginatedResult[*Build], error)

    // FindByStatus returns builds in a specific status (cross-tenant, admin only).
    FindByStatus(ctx context.Context, status BuildStatus, page shared.Pagination) (*shared.PaginatedResult[*Build], error)

    // FindStaleCheckpoints finds builds whose checkpoint is older than threshold.
    // Used by the recovery monitor.
    FindStaleCheckpoints(ctx context.Context, olderThan time.Duration) ([]*Build, error)

    // CountActiveByTenant for quota enforcement.
    CountActiveByTenant(ctx context.Context, tenantID TenantID) (int64, error)
}

type BuildFilter struct {
    Status    *BuildStatus
    Stage     *StageNumber
    DateRange *shared.DateRange
    SortBy    string // "created_at", "updated_at", "current_stage"
    SortOrder shared.SortOrder
}
```

---

## ArtifactRepository

```go
package artifact

import (
    "context"
    "neuroforge-sdx/domain/shared"
)

type Repository interface {
    Save(ctx context.Context, artifact *Artifact) error
    FindByID(ctx context.Context, id ArtifactID) (*Artifact, error)

    // FindByBuild returns all artifacts for a build.
    FindByBuild(ctx context.Context, buildID BuildID) ([]*Artifact, error)

    // FindByContentHash for deduplication — checks if identical artifact already exists.
    FindByContentHash(ctx context.Context, hash ContentHash) (*Artifact, error)

    // FindByTenant returns paginated artifacts for a tenant.
    FindByTenant(ctx context.Context, tenantID TenantID, filter ArtifactFilter, page shared.Pagination) (*shared.PaginatedResult[*Artifact], error)

    // FindPublishedByType for registry queries.
    FindPublishedByType(ctx context.Context, tenantID TenantID, artifactType ArtifactType, page shared.Pagination) (*shared.PaginatedResult[*Artifact], error)
}

type ArtifactFilter struct {
    Type      *ArtifactType
    Status    *ArtifactStatus
    BuildID   *BuildID
    DateRange *shared.DateRange
}
```

---

## TenantRepository

```go
package tenant

import (
    "context"
    "neuroforge-sdx/domain/shared"
)

type Repository interface {
    Save(ctx context.Context, tenant *Tenant) error
    FindByID(ctx context.Context, id TenantID) (*Tenant, error)

    // FindByPlan returns tenants on a specific plan tier.
    FindByPlan(ctx context.Context, plan PlanTier, page shared.Pagination) (*shared.PaginatedResult[*Tenant], error)

    // FindByStatus returns tenants in a specific status.
    FindByStatus(ctx context.Context, status TenantStatus, page shared.Pagination) (*shared.PaginatedResult[*Tenant], error)

    // FindByRegion returns tenants in a specific NexoRegion.
    FindByRegion(ctx context.Context, region NexoRegion, page shared.Pagination) (*shared.PaginatedResult[*Tenant], error)

    // ExistsByName checks for tenant name uniqueness.
    ExistsByName(ctx context.Context, name string) (bool, error)
}
```

---

## AgentRepository

```go
package agent

import (
    "context"
    "time"
    "neuroforge-sdx/domain/shared"
)

type Repository interface {
    Save(ctx context.Context, agent *Agent) error
    FindByID(ctx context.Context, id AgentID) (*Agent, error)

    // FindAvailableByType returns IDLE agents of a specific type in a region.
    FindAvailableByType(ctx context.Context, agentType AgentType, region NexoRegion) ([]*Agent, error)

    // FindByBuild returns agents currently assigned to a build.
    FindByBuild(ctx context.Context, buildID BuildID) ([]*Agent, error)

    // FindUnhealthy returns agents that haven't heartbeated within threshold.
    FindUnhealthy(ctx context.Context, threshold time.Duration) ([]*Agent, error)

    // FindByType returns all agents of a given type, with pagination.
    FindByType(ctx context.Context, agentType AgentType, page shared.Pagination) (*shared.PaginatedResult[*Agent], error)

    // CountByTypeAndStatus for dashboard metrics.
    CountByTypeAndStatus(ctx context.Context, agentType AgentType, status AgentStatus) (int64, error)
}
```

---

## QualityReportRepository

```go
package quality

import (
    "context"
    "neuroforge-sdx/domain/shared"
)

type Repository interface {
    // Save stores a quality report (ClickHouse — append-only).
    Save(ctx context.Context, report *QualityReport) error

    // FindByBuild returns all quality reports for a build.
    FindByBuild(ctx context.Context, buildID BuildID) ([]*QualityReport, error)

    // FindGateResults returns gate evaluation history for a build.
    FindGateResults(ctx context.Context, buildID BuildID) ([]*GateResult, error)

    // FindGateResultsByGate returns evaluations for a specific gate across builds.
    FindGateResultsByGate(ctx context.Context, gateID GateID, filter GateResultFilter, page shared.Pagination) (*shared.PaginatedResult[*GateResult], error)

    // AggregateGateMetrics returns aggregated gate pass/fail rates for analytics.
    AggregateGateMetrics(ctx context.Context, tenantID TenantID, dateRange shared.DateRange) (*GateMetricsAggregate, error)
}

type GateResultFilter struct {
    TenantID  *TenantID
    Passed    *bool
    DateRange *shared.DateRange
}

type GateMetricsAggregate struct {
    GateID    GateID
    PassRate  float64
    AvgValue  float64
    MinValue  float64
    MaxValue  float64
    EvalCount int64
}
```

---

## ComplianceEvidenceRepository

```go
package compliance

import (
    "context"
    "neuroforge-sdx/domain/shared"
)

type Repository interface {
    Save(ctx context.Context, evidence *ComplianceEvidence) error
    FindByBuild(ctx context.Context, buildID BuildID) ([]*ComplianceEvidence, error)

    // FindByFramework returns evidence packages for a framework (compliance auditor view).
    FindByFramework(ctx context.Context, tenantID TenantID, framework ComplianceFramework, page shared.Pagination) (*shared.PaginatedResult[*ComplianceEvidence], error)

    // FindByTenant returns all evidence for a tenant (cross-framework).
    FindByTenant(ctx context.Context, tenantID TenantID, dateRange shared.DateRange, page shared.Pagination) (*shared.PaginatedResult[*ComplianceEvidence], error)

    // GetControlCoverage returns coverage percentage per framework for a build.
    GetControlCoverage(ctx context.Context, buildID BuildID) (map[ComplianceFramework]float64, error)
}
```

---

## AuditLogRepository

```go
package audit

import (
    "context"
    "neuroforge-sdx/domain/shared"
)

type Repository interface {
    // Append writes an audit entry. Append-only. No updates. No deletes.
    Append(ctx context.Context, entry *AuditEntry) error

    // Query returns audit entries matching filters, ordered by timestamp.
    Query(ctx context.Context, filter AuditFilter, page shared.Pagination) (*shared.PaginatedResult[*AuditEntry], error)

    // GetChain returns entries between two entry IDs for chain verification.
    GetChain(ctx context.Context, startID EventID, endID EventID) ([]*AuditEntry, error)

    // GetLatestEntry returns the most recent entry (for prev_hash chaining).
    GetLatestEntry(ctx context.Context, tenantID TenantID) (*AuditEntry, error)
}

type AuditFilter struct {
    TenantID   *TenantID
    EventTypes []string
    ActorType  *ActorType
    ActorID    *string
    TargetType *string
    TargetID   *string
    DateRange  *shared.DateRange
}
```

---

## DeploymentRepository

```go
package deployment

import (
    "context"
    "neuroforge-sdx/domain/shared"
)

type Repository interface {
    Save(ctx context.Context, deployment *Deployment) error
    FindByID(ctx context.Context, id DeploymentID) (*Deployment, error)
    FindByBuild(ctx context.Context, buildID BuildID) ([]*Deployment, error)

    // FindActiveByService returns the current active deployment for a service in an environment.
    FindActiveByService(ctx context.Context, tenantID TenantID, serviceName string, targetEnv TargetEnvironment) (*Deployment, error)

    // FindByTenant returns deployment history.
    FindByTenant(ctx context.Context, tenantID TenantID, filter DeploymentFilter, page shared.Pagination) (*shared.PaginatedResult[*Deployment], error)
}

type DeploymentFilter struct {
    TargetEnv *TargetEnvironment
    Status    *DeploymentStatus
    DateRange *shared.DateRange
}
```

---

## DomainModelRepository

```go
package domainmodel

import (
    "context"
)

type Repository interface {
    Save(ctx context.Context, model *DomainModel) error
    FindByID(ctx context.Context, id DomainModelID) (*DomainModel, error)
    FindByWorkItem(ctx context.Context, workItemID WorkItemID) (*DomainModel, error)
}
```

---

## BlueprintRepository

```go
package blueprint

import (
    "context"
)

type Repository interface {
    Save(ctx context.Context, bp *Blueprint) error
    FindByID(ctx context.Context, id BlueprintID) (*Blueprint, error)
    FindByWorkItem(ctx context.Context, workItemID WorkItemID) (*Blueprint, error)
}
```

---

## WebhookConfigRepository

```go
package webhook

import (
    "context"
    "neuroforge-sdx/domain/shared"
)

type Repository interface {
    Save(ctx context.Context, config *WebhookConfig) error
    FindByID(ctx context.Context, id WebhookID) (*WebhookConfig, error)
    FindByTenant(ctx context.Context, tenantID TenantID) ([]*WebhookConfig, error)
    FindByEvent(ctx context.Context, tenantID TenantID, eventType string) ([]*WebhookConfig, error)
    Delete(ctx context.Context, id WebhookID) error
}
```

---

## ContractRepository

```go
package contract

import (
    "context"
    "neuroforge-sdx/domain/shared"
)

type Repository interface {
    // Save persists a new or updated Contract. Version-aware upsert.
    Save(ctx context.Context, contract *Contract) error

    // FindByID loads a Contract by its ID.
    FindByID(ctx context.Context, id ContractID) (*Contract, error)

    // FindByBlueprint returns all contracts generated from a blueprint.
    FindByBlueprint(ctx context.Context, blueprintID BlueprintID) ([]*Contract, error)

    // FindByVersion returns a specific version of a contract for a blueprint.
    FindByVersion(ctx context.Context, blueprintID BlueprintID, version string) (*Contract, error)

    // FindByTenant returns paginated contracts for a tenant.
    FindByTenant(ctx context.Context, tenantID TenantID, filter ContractFilter, page shared.Pagination) (*shared.PaginatedResult[*Contract], error)
}

type ContractFilter struct {
    BlueprintID *BlueprintID
    Type        *ContractType // OPENAPI, ASYNCAPI, GRPC
    Status      *ContractStatus
    DateRange   *shared.DateRange
}
```

---

## ModelRoutingConfigRepository

```go
package modelrouting

import "context"

type Repository interface {
    // Save persists a model routing config (DragonflyDB or YugabyteDB).
    Save(ctx context.Context, config *ModelRoutingConfig) error

    // FindByAgentType returns the routing config for a specific agent type.
    FindByAgentType(ctx context.Context, agentType AgentType) (*ModelRoutingConfig, error)

    // FindByAgentID returns the routing config for a specific agent instance.
    FindByAgentID(ctx context.Context, agentID AgentID) (*ModelRoutingConfig, error)

    // FindDefault returns the default routing config (fallback when no agent-specific config exists).
    FindDefault(ctx context.Context) (*ModelRoutingConfig, error)

    // FindAll returns all routing configs for admin dashboard.
    FindAll(ctx context.Context) ([]*ModelRoutingConfig, error)
}
```

---

## LLMRequestRepository

```go
package llmrequest

import (
    "context"
    "neuroforge-sdx/domain/shared"
)

// LLMRequestRepository stores LLM request/response records for cost tracking and learning.
// Backed by ClickHouse (append-only, time-series optimised).
type Repository interface {
    // Append writes a new LLM request record. Never updates.
    Append(ctx context.Context, record *LLMRequestRecord) error

    // GetCostByTenant returns aggregated cost for a tenant over a period.
    GetCostByTenant(ctx context.Context, tenantID TenantID, period shared.DateRange) (*CostReport, error)

    // GetCostByBuild returns aggregated cost for a specific build.
    GetCostByBuild(ctx context.Context, buildID BuildID) (*CostReport, error)

    // GetCostByModel returns aggregated cost per model over a period.
    GetCostByModel(ctx context.Context, model ModelID, period shared.DateRange) (*ModelCostBreakdown, error)

    // GetRequestsByBuild returns all LLM requests for a build (for learning analysis).
    GetRequestsByBuild(ctx context.Context, buildID BuildID) ([]*LLMRequestRecord, error)
}

type LLMRequestRecord struct {
    RequestID       string
    BuildID         string
    TenantID        string
    AgentID         string
    Model           string
    TaskClass       string
    InputTokens     uint64
    OutputTokens    uint64
    LatencyMS       uint64
    CostMicrodollar uint64
    FallbackUsed    bool
    Timestamp       shared.Timestamp
}

type CostReport struct {
    TotalCostMicrodollars uint64
    TotalInputTokens      uint64
    TotalOutputTokens     uint64
    RequestCount          uint64
    OnPremPct             float64
    ByModel               []ModelCostBreakdown
}

type ModelCostBreakdown struct {
    Model             string
    InputTokens       uint64
    OutputTokens      uint64
    CostMicrodollars  uint64
    RequestCount      uint64
    AvgLatencyMS      float64
}
```

---

## Error Conventions

```go
package shared

import "errors"

var (
    ErrNotFound          = errors.New("entity not found")
    ErrConflict          = errors.New("entity already exists or version conflict")
    ErrTenantMismatch    = errors.New("tenant ID does not match")
    ErrInvariantViolated = errors.New("aggregate invariant violated")
    ErrQuotaExhausted    = errors.New("tenant quota exhausted")
    ErrImmutable         = errors.New("entity is immutable in current state")
)
```
