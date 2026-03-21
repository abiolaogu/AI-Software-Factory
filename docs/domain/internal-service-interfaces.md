# NeuroForge SDX — Internal Service Interfaces (Temporal Activity Contracts)

> Per ADR-013: Internal services communicate via Temporal activity interfaces in M1.
> gRPC proto definitions will be added in M2 for services requiring independent scaling.
> These Go function signatures are the contract Codex implements for each Temporal activity.

---

## Naming Convention

```
Package:    internal/workflow/activities/{service_name}
Interface:  {ServiceName}Activities
Methods:    Activity methods called by BuildPipelineWorkflow
Context:    context.Context carries tenant_id, build_id, trace_id
```

---

## 1. DomainModellingActivities (BC-02)

```go
package domainmodelling

import "context"

type Activities interface {
    // ExtractDomainModel analyses parsed WorkItem requirements and produces
    // a DomainModel with bounded contexts, aggregates, entities, VOs, events.
    ExtractDomainModel(ctx context.Context, input ExtractDomainModelInput) (*ExtractDomainModelOutput, error)

    // RefineDomainModel improves an existing domain model based on review feedback.
    RefineDomainModel(ctx context.Context, input RefineDomainModelInput) (*RefineDomainModelOutput, error)
}

type ExtractDomainModelInput struct {
    BuildID       string
    WorkItemID    string
    TenantID      string
    Requirements  []Requirement
    SourceContent string
}

type ExtractDomainModelOutput struct {
    DomainModelID       string
    BoundedContextCount int
    AggregateCount      int
    EntityCount         int
    ValueObjectCount    int
    DomainEventCount    int
    LanguageTermCount   int
    CompletenessScore   float64
    StorageRef          string // RustFS path
}

type RefineDomainModelInput struct {
    BuildID       string
    DomainModelID string
    Feedback      string
}

type RefineDomainModelOutput struct {
    DomainModelID     string
    CompletenessScore float64
    Version           uint32
}
```

---

## 2. ArchitectureActivities (BC-03)

```go
package architecture

import "context"

type Activities interface {
    // DesignBlueprint generates a system architecture from the domain model.
    DesignBlueprint(ctx context.Context, input DesignBlueprintInput) (*DesignBlueprintOutput, error)

    // GenerateADRs creates Architecture Decision Records for the blueprint.
    GenerateADRs(ctx context.Context, input GenerateADRsInput) (*GenerateADRsOutput, error)
}

type DesignBlueprintInput struct {
    BuildID       string
    DomainModelID string
    TenantID      string
    Languages     []string
    Frameworks    []string
}

type DesignBlueprintOutput struct {
    BlueprintID         string
    ServiceCount        int
    TechStackSummary    map[string]string
    StorageRef          string
}

type GenerateADRsInput struct {
    BuildID     string
    BlueprintID string
}

type GenerateADRsOutput struct {
    ADRCount   int
    StorageRef string
}
```

---

## 3. ContractGenerationActivities (BC-04)

```go
package contractgen

import "context"

type Activities interface {
    // GenerateContracts produces OpenAPI, AsyncAPI, and gRPC proto from a blueprint.
    GenerateContracts(ctx context.Context, input GenerateContractsInput) (*GenerateContractsOutput, error)

    // ValidateContracts checks for breaking changes against previous version.
    ValidateContracts(ctx context.Context, input ValidateContractsInput) (*ValidateContractsOutput, error)
}

type GenerateContractsInput struct {
    BuildID     string
    BlueprintID string
    TenantID    string
}

type GenerateContractsOutput struct {
    ContractIDs []string // One per contract type generated
    StorageRef  string
}

type ValidateContractsInput struct {
    BuildID        string
    ContractIDs    []string
    PreviousSpecID string
}

type ValidateContractsOutput struct {
    BreakingChanges []string
    Valid           bool
}
```

---

## 4. TestSpecActivities (BC-05)

```go
package testspec

import "context"

type Activities interface {
    // GenerateTestSpecs writes BDD scenarios and failing test stubs (RED phase).
    GenerateTestSpecs(ctx context.Context, input GenerateTestSpecsInput) (*GenerateTestSpecsOutput, error)
}

type GenerateTestSpecsInput struct {
    BuildID     string
    ContractIDs []string
    BlueprintID string
    TenantID    string
    Languages   []string
}

type GenerateTestSpecsOutput struct {
    TestSpecID     string
    ScenarioCount  int
    TestStubCount  int
    StorageRef     string
}
```

---

## 5. CodeGenActivities (BC-06)

```go
package codegen

import "context"

type Activities interface {
    // GenerateCode implements production code to make all test stubs GREEN.
    GenerateCode(ctx context.Context, input GenerateCodeInput) (*GenerateCodeOutput, error)

    // RefactorCode improves code quality without changing behaviour (remediation).
    RefactorCode(ctx context.Context, input RefactorCodeInput) (*RefactorCodeOutput, error)
}

type GenerateCodeInput struct {
    BuildID    string
    TestSpecID string
    BlueprintID string
    TenantID   string
    Languages  []string
}

type GenerateCodeOutput struct {
    CodeArtifactRef string
    FileCount       int
    LineCount       int
    StorageRef      string
}

type RefactorCodeInput struct {
    BuildID         string
    CodeArtifactRef string
    GateID          string    // Which gate failed
    GateValue       float64   // Current score
    GateThreshold   float64   // Required threshold
}

type RefactorCodeOutput struct {
    CodeArtifactRef string
    ChangesSummary  string
}
```

---

## 6. SecurityScanActivities (BC-08)

```go
package securityscan

import "context"

type Activities interface {
    // RunSAST executes static analysis (gosec, semgrep, etc.).
    RunSAST(ctx context.Context, input ScanInput) (*SecurityScanOutput, error)

    // RunDAST executes dynamic analysis (OWASP ZAP).
    RunDAST(ctx context.Context, input ScanInput) (*SecurityScanOutput, error)

    // RunSCA executes software composition analysis (Grype).
    RunSCA(ctx context.Context, input ScanInput) (*SecurityScanOutput, error)

    // RunSecretsScan detects secrets (Gitleaks + custom patterns).
    RunSecretsScan(ctx context.Context, input ScanInput) (*SecurityScanOutput, error)

    // RunIaCScan validates IaC (Checkov).
    RunIaCScan(ctx context.Context, input ScanInput) (*SecurityScanOutput, error)

    // GenerateSBOM produces SBOM (Syft).
    GenerateSBOM(ctx context.Context, input ScanInput) (*SBOMOutput, error)
}

type ScanInput struct {
    BuildID         string
    TenantID        string
    CodeArtifactRef string
}

type SecurityScanOutput struct {
    ScanType      string // SAST, DAST, SCA, SECRETS, IAC
    FindingsCount int
    CriticalCount int
    HighCount     int
    MediumCount   int
    LowCount      int
    ReportRef     string
}

type SBOMOutput struct {
    ArtifactID     string
    Format         string // SPDX or CYCLONEDX
    ComponentCount int
    StorageRef     string
}
```

---

## 7. PerformanceTestActivities (BC-10)

```go
package perftest

import "context"

type Activities interface {
    // RunLoadTest executes k6 load tests.
    RunLoadTest(ctx context.Context, input LoadTestInput) (*PerformanceOutput, error)

    // ProfileMemory runs memory profiling.
    ProfileMemory(ctx context.Context, input ProfileInput) (*PerformanceOutput, error)
}

type LoadTestInput struct {
    BuildID         string
    TenantID        string
    CodeArtifactRef string
    TargetRPS       uint32
    P99LatencyMS    uint32
}

type ProfileInput struct {
    BuildID         string
    TenantID        string
    CodeArtifactRef string
    MemoryBudgetMB  uint32
}

type PerformanceOutput struct {
    P99LatencyMS     uint32
    AvgLatencyMS     uint32
    ErrorRatePct     float64
    MemoryPeakMB     uint32
    Passed           bool
    ReportRef        string
    FlameGraphRef    string
}
```

---

## 8. DocsGenerationActivities (BC-11)

```go
package docsgen

import "context"

type Activities interface {
    // GenerateDocs produces README, API docs, Mermaid diagrams, runbooks, changelogs.
    GenerateDocs(ctx context.Context, input GenerateDocsInput) (*GenerateDocsOutput, error)
}

type GenerateDocsInput struct {
    BuildID         string
    BlueprintID     string
    ContractIDs     []string
    CodeArtifactRef string
    TenantID        string
}

type GenerateDocsOutput struct {
    DocsArtifactRef string
    FileCount       int
    StorageRef      string
}
```

---

## 9. IaCActivities (BC-12)

```go
package iac

import "context"

type Activities interface {
    // GenerateDockerfiles creates Dockerfiles for all services.
    GenerateDockerfiles(ctx context.Context, input IaCInput) (*IaCOutput, error)

    // GenerateHelmCharts creates Helm charts.
    GenerateHelmCharts(ctx context.Context, input IaCInput) (*IaCOutput, error)

    // GenerateTerraform creates Terraform modules.
    GenerateTerraform(ctx context.Context, input IaCInput) (*IaCOutput, error)

    // GenerateGitOps creates Fleet GitOps configuration.
    GenerateGitOps(ctx context.Context, input IaCInput) (*IaCOutput, error)

    // GenerateCI creates GitHub Actions / GitLab CI workflows.
    GenerateCI(ctx context.Context, input IaCInput) (*IaCOutput, error)
}

type IaCInput struct {
    BuildID     string
    BlueprintID string
    TenantID    string
    TargetEnv   string
    NexoRegion  string
}

type IaCOutput struct {
    ArtifactType string // HELM_CHART, TERRAFORM_MODULE, etc.
    ArtifactRef  string
    FileCount    int
    StorageRef   string
}
```

---

## 10. LearningActivities (BC-15)

```go
package learning

import "context"

type Activities interface {
    // AggregateBuildMetrics collects and stores metrics from a completed build.
    AggregateBuildMetrics(ctx context.Context, input AggregateMetricsInput) (*AggregateMetricsOutput, error)

    // UpdatePromptTemplates refines agent prompt templates based on build outcomes.
    UpdatePromptTemplates(ctx context.Context, input UpdatePromptsInput) (*UpdatePromptsOutput, error)

    // UpdateRoutingPolicies adjusts model routing based on cost/quality metrics.
    UpdateRoutingPolicies(ctx context.Context, input UpdateRoutingInput) (*UpdateRoutingOutput, error)
}

type AggregateMetricsInput struct {
    BuildID  string
    TenantID string
}

type AggregateMetricsOutput struct {
    MetricsStored int
}

type UpdatePromptsInput struct {
    BuildID       string
    GateResults   map[string]float64
    TokenUsage    map[string]uint64
}

type UpdatePromptsOutput struct {
    TemplatesUpdated int
    Version          uint32
}

type UpdateRoutingInput struct {
    Period    string // e.g., "24h", "7d"
}

type UpdateRoutingOutput struct {
    PoliciesUpdated int
}
```

---

## Activity Registration

All activities are registered with the Temporal worker in `internal/workflow/worker.go`:

```go
func RegisterActivities(w worker.Worker) {
    w.RegisterActivity(&domainmodelling.ActivitiesImpl{})
    w.RegisterActivity(&architecture.ActivitiesImpl{})
    w.RegisterActivity(&contractgen.ActivitiesImpl{})
    w.RegisterActivity(&testspec.ActivitiesImpl{})
    w.RegisterActivity(&codegen.ActivitiesImpl{})
    w.RegisterActivity(&securityscan.ActivitiesImpl{})
    w.RegisterActivity(&perftest.ActivitiesImpl{})
    w.RegisterActivity(&docsgen.ActivitiesImpl{})
    w.RegisterActivity(&iac.ActivitiesImpl{})
    w.RegisterActivity(&learning.ActivitiesImpl{})
}
```
