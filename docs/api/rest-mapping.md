# NeuroForge SDX — REST Mapping for All gRPC RPCs

> REST endpoints are served by sdx-gateway via grpc-gateway.
> Base URL: `https://api.sdx.{region}.stratogroup.io/api/v1`
> All endpoints require `Authorization: Bearer {jwt}` header.
> All mutating endpoints accept optional `Idempotency-Key: {uuid}` header.
> All responses include `X-Request-ID`, `X-RateLimit-*` headers.

---

## WorkItem Endpoints

| Method | Path | gRPC RPC | Description |
|--------|------|----------|-------------|
| POST | `/work-items` | `IngestionService.SubmitWorkItem` | Submit a new WorkItem |
| GET | `/work-items/{id}` | `IngestionService.GetWorkItem` | Get a WorkItem by ID |
| GET | `/work-items` | `IngestionService.ListWorkItems` | List WorkItems with filters |
| PATCH | `/work-items/{id}/ambiguities/{ambiguity_id}` | `IngestionService.ResolveAmbiguity` | Resolve an ambiguity |
| POST | `/work-items/{id}/approve` | `IngestionService.ApproveManifest` | Approve manifest and start build |
| POST | `/work-items/{id}/restart` | `IngestionService.RestartWorkItem` | Restart a FAILED WorkItem |
| DELETE | `/work-items/{id}` | `IngestionService.DeleteWorkItem` | Soft-delete (DRAFT/FAILED only) |

## Build Endpoints

| Method | Path | gRPC RPC | Description |
|--------|------|----------|-------------|
| GET | `/builds/{id}` | `BuildService.GetBuild` | Get Build by ID |
| GET | `/builds` | `BuildService.ListBuilds` | List Builds with filters |
| GET | `/builds/{id}/gates` | `BuildService.GetBuildGateResults` | Get gate results for a build |
| GET | `/builds/{id}/stages` | `BuildService.GetBuildStages` | Get stage history |
| POST | `/builds/{id}/cancel` | `BuildService.CancelBuild` | Cancel a running build |

## Artifact Endpoints

| Method | Path | gRPC RPC | Description |
|--------|------|----------|-------------|
| GET | `/artifacts/{id}` | `ArtifactService.GetArtifact` | Get artifact metadata |
| GET | `/artifacts` | `ArtifactService.ListArtifacts` | List artifacts with filters |
| GET | `/artifacts/{id}/download` | `ArtifactService.GetArtifactDownloadURL` | Get pre-signed download URL |
| POST | `/artifacts/{id}/verify` | `ArtifactService.VerifyArtifact` | Verify integrity and provenance |
| POST | `/artifacts/{id}/revoke` | `ArtifactService.RevokeArtifact` | Revoke an artifact |

## Tenant Endpoints

| Method | Path | gRPC RPC | Description |
|--------|------|----------|-------------|
| POST | `/tenants` | `TenantService.CreateTenant` | Create a new tenant |
| GET | `/tenants/{id}` | `TenantService.GetTenant` | Get tenant by ID |
| GET | `/tenants` | `TenantService.ListTenants` | List tenants (admin) |
| PATCH | `/tenants/{id}/plan` | `TenantService.UpdatePlan` | Update plan tier |
| POST | `/tenants/{id}/suspend` | `TenantService.SuspendTenant` | Suspend tenant |
| POST | `/tenants/{id}/reactivate` | `TenantService.ReactivateTenant` | Reactivate tenant |
| GET | `/tenants/{id}/quotas` | `TenantService.GetQuotaStatus` | Get quota usage |
| POST | `/tenants/{id}/workspaces` | `TenantService.CreateWorkspace` | Create workspace |
| GET | `/tenants/{id}/workspaces` | `TenantService.ListWorkspaces` | List workspaces |
| POST | `/tenants/{id}/workspaces/{wid}/members` | `TenantService.AddWorkspaceMember` | Add member |
| DELETE | `/tenants/{id}/workspaces/{wid}/members/{uid}` | `TenantService.RemoveWorkspaceMember` | Remove member |
| PUT | `/tenants/{id}/compliance-profile` | `TenantService.SetComplianceProfile` | Set compliance profile |
| GET | `/tenants/{id}/compliance-profile` | `TenantService.GetComplianceProfile` | Get compliance profile |

## Quality Endpoints

| Method | Path | gRPC RPC | Description |
|--------|------|----------|-------------|
| GET | `/quality/gates` | `QualityService.GetGateDefinitions` | List all gate definitions |
| GET | `/quality/reports/{build_id}` | `QualityService.GetQualityReport` | Get quality report |
| GET | `/quality/reports` | `QualityService.ListQualityReports` | List reports |
| GET | `/quality/metrics/{gate_id}` | `QualityService.GetGateMetrics` | Get gate metrics |

## Compliance Endpoints

| Method | Path | gRPC RPC | Description |
|--------|------|----------|-------------|
| GET | `/compliance/reports/{build_id}` | `ComplianceService.GetComplianceReport` | Get compliance report |
| GET | `/compliance/reports` | `ComplianceService.ListComplianceReports` | List reports |
| GET | `/compliance/badges/{build_id}` | `ComplianceService.GetComplianceBadge` | Get compliance badge |
| GET | `/compliance/controls` | `ComplianceService.GetControlCatalog` | Get control catalog |

## Agent Endpoints (Admin Only)

| Method | Path | gRPC RPC | Description |
|--------|------|----------|-------------|
| GET | `/agents/{id}` | `AgentService.GetAgent` | Get agent by ID |
| GET | `/agents` | `AgentService.ListAgents` | List agents |
| GET | `/agents/{id}/health` | `AgentService.GetAgentHealth` | Get health report |
| GET | `/agents/{id}/metrics` | `AgentService.GetAgentMetrics` | Get agent metrics |

## LLM Endpoints (Admin Only)

| Method | Path | gRPC RPC | Description |
|--------|------|----------|-------------|
| GET | `/llm/models` | `LLMService.ListModels` | List available models |
| GET | `/llm/status` | `LLMService.GetModelStatus` | Get model status |
| GET | `/llm/costs` | `LLMService.GetCostReport` | Get cost report |

## Deployment Endpoints

| Method | Path | gRPC RPC | Description |
|--------|------|----------|-------------|
| GET | `/deployments/{id}` | `DeploymentService.GetDeployment` | Get deployment |
| GET | `/deployments` | `DeploymentService.ListDeployments` | List deployments |
| POST | `/deployments/{id}/rollback` | `DeploymentService.RollbackDeployment` | Rollback |
| GET | `/deployments/{id}/logs` | `DeploymentService.GetDeploymentLogs` | Get logs |

## Audit Endpoints

| Method | Path | gRPC RPC | Description |
|--------|------|----------|-------------|
| GET | `/audit-log` | `AuditService.QueryAuditLog` | Query audit log |
| POST | `/audit-log/verify` | `AuditService.VerifyIntegrity` | Verify hash chain |
| POST | `/audit-log/export` | `AuditService.ExportEvidence` | Export evidence |
| GET | `/audit-log/stats` | `AuditService.GetAuditStats` | Get audit stats |

## Webhook Endpoints

| Method | Path | gRPC RPC | Description |
|--------|------|----------|-------------|
| POST | `/webhooks` | `SDXGatewayService.CreateWebhook` | Create webhook |
| GET | `/webhooks` | `SDXGatewayService.ListWebhooks` | List webhooks |
| DELETE | `/webhooks/{id}` | `SDXGatewayService.DeleteWebhook` | Delete webhook |
| POST | `/webhooks/{id}/test` | `SDXGatewayService.TestWebhook` | Test webhook |

## System Endpoints

| Method | Path | gRPC RPC | Description |
|--------|------|----------|-------------|
| GET | `/health` | `SDXGatewayService.HealthCheck` | Health check |
| GET | `/openapi.json` | — | Raw OpenAPI spec |
| GET | `/swagger` | — | Swagger UI |
| GET | `/docs` | — | Redoc |

---

## Query Parameter Conventions

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `page_size` | int | Items per page (max 100) | `?page_size=20` |
| `page_token` | string | Opaque pagination cursor | `?page_token=abc123` |
| `sort_by` | string | Sort field | `?sort_by=created_at` |
| `sort_order` | string | ASC or DESC | `?sort_order=DESC` |
| `date_start` | RFC3339 | Start of date range | `?date_start=2026-01-01T00:00:00Z` |
| `date_end` | RFC3339 | End of date range | `?date_end=2026-03-21T00:00:00Z` |

## Response Headers

| Header | Description |
|--------|-------------|
| `X-Request-ID` | Unique request correlation ID (ULID) |
| `X-RateLimit-Limit` | Max requests per window |
| `X-RateLimit-Remaining` | Remaining requests in window |
| `X-RateLimit-Reset` | Unix timestamp when window resets |
| `X-Tenant-ID` | Confirmed tenant ID |
