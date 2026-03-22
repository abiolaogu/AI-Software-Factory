# NeuroForge SDX Domain Model

## Aggregates

### Workflow

The `Workflow` aggregate is the primary orchestration boundary for SDX.

Core attributes:

- `workflow_id`
- `status`
- `current_stage`
- `progress`
- `created_at`
- `started_at`
- `completed_at`
- `estimated_completion`
- `artifacts`
- `_links`

Invariants:

- A workflow always belongs to exactly one tenant scope.
- A workflow identifier is unique only within a tenant-safe repository boundary.
- Cancellation transitions a workflow into the `failed` stage with an operator-generated error entry.

### PRD

The `PRD` aggregate captures the generated product requirements document associated with a tenant and request.

Core attributes:

- `prd_id`
- `markdown`
- `json`
- `success_metrics`
- `user_stories`
- `technical_requirements`
- `created_at`

Invariants:

- A PRD is persisted and retrieved only inside the originating tenant scope.
- PRD generation can be replayed idempotently for the same tenant, operation, and idempotency key.

### Idempotency Record

The `IdempotencyRecord` ensures mutating operations replay the original successful response instead of creating duplicate work.

Core attributes:

- `tenant_id`
- `operation`
- `key`
- `status_code`
- `response_body`
- `response_digest`

Invariants:

- Idempotency keys are isolated by tenant and operation.
- A repeated key for a different tenant must not return another tenant's response.

## Value Objects

### Tenant Scope

Tenant scope is carried through request context and enforced on every protected API path and repository operation.

### Artifact Descriptor

Artifact descriptors represent immutable workflow outputs:

- `url`
- `type`
- `size`
- `created_at`

### Error Detail

Error details capture field-level validation failures and operational diagnostics without leaking infrastructure-specific types into the domain layer.
