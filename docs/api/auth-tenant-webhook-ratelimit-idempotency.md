# NeuroForge SDX — API Policy Contracts

> Covers: Authentication, Authorization, Tenant Scoping, Webhooks, Rate Limiting, Idempotency, Pagination.

---

## B6. Authentication & Authorization Contract

### Authentication

| Method | Transport | Mechanism |
|--------|-----------|-----------|
| Human users | REST/gRPC | Bearer JWT via Authentik OIDC. Token in `Authorization: Bearer {jwt}` header. |
| API clients | REST/gRPC | API key in `X-API-Key` header (maps to tenant). |
| Agents (inter-service) | gRPC | mTLS with SPIFFE SVID. No Bearer token. Identity in TLS certificate. |
| Webhooks (inbound) | REST | HMAC-SHA256 signature in `X-SDX-Signature` header. |

### JWT Claims

```json
{
  "sub": "user-ulid",
  "iss": "https://auth.stratogroup.io",
  "aud": "sdx-api",
  "tenant_id": "tenant-ulid",
  "workspace_id": "workspace-ulid",
  "roles": ["ADMIN", "MEMBER"],
  "exp": 1711036800,
  "iat": 1711035900
}
```

### Authorization (RBAC)

| Role | Scope | Permissions |
|------|-------|-------------|
| OWNER | Tenant-wide | All operations including tenant management, plan changes, member management |
| ADMIN | Workspace-wide | Create/manage WorkItems, Builds, Webhooks. View all builds in workspace. |
| MEMBER | Workspace-wide | Create WorkItems, view builds, download artifacts. Cannot manage webhooks. |
| VIEWER | Workspace-wide | Read-only access to builds, artifacts, reports. |
| AUDIT_READER | Tenant-wide | Read-only access to audit log and compliance reports. |
| PLATFORM_ADMIN | Global | StratoGroup internal. Cross-tenant admin operations. |

### Permission Matrix

| Operation | OWNER | ADMIN | MEMBER | VIEWER | AUDIT_READER |
|-----------|-------|-------|--------|--------|-------------|
| Submit WorkItem | Y | Y | Y | N | N |
| Approve Manifest | Y | Y | N | N | N |
| Cancel Build | Y | Y | N | N | N |
| View Build | Y | Y | Y | Y | N |
| Download Artifact | Y | Y | Y | Y | N |
| Revoke Artifact | Y | Y | N | N | N |
| Manage Webhooks | Y | Y | N | N | N |
| Query Audit Log | Y | Y | N | N | Y |
| Manage Tenant | Y | N | N | N | N |
| Manage Workspaces | Y | Y | N | N | N |
| View Agent Status | Y | Y | Y | Y | N |
| View LLM Costs | Y | Y | N | N | N |

### Token Lifecycle

- Access token TTL: 15 minutes
- Refresh token TTL: 7 days
- Agent SVID TTL: 1 hour (auto-rotated by SPIRE)
- API key: No expiry; revocable by OWNER

---

## B7. Tenant Scoping Contract

### Propagation

Every API request MUST carry tenant context:

| Transport | Header/Metadata | Source |
|-----------|-----------------|--------|
| REST | `X-Tenant-ID` header | Extracted from JWT `tenant_id` claim by sdx-gateway |
| gRPC | `x-tenant-id` metadata | Extracted from JWT by auth interceptor |
| NATS | `Sdx-Tenant-Id` message header | Set by publishing service |
| Redpanda | `sdx_tenant_id` record header | Set by producing service |

### Enforcement

1. **sdx-gateway** extracts `tenant_id` from JWT and injects into gRPC metadata.
2. **Every service** reads `tenant_id` from gRPC metadata context.
3. **Every database query** includes `WHERE tenant_id = $1` (enforced by YugabyteDB RLS).
4. **Every RustFS path** starts with `/{tenant_id}/`.
5. **Every DragonflyDB key** is prefixed with `{tenant_id}:`.
6. **Cross-tenant access is prohibited** — returns SDX-1003.

### Data Residency

| Tenant Region | Data Location | Replication |
|---------------|---------------|-------------|
| LAGOS | UGCP Lagos | No cross-region replication |
| LONDON (GDPR) | UGCP London | EU-only; no replication outside EU |
| ASHBURN | UGCP Ashburn | No cross-region replication |
| SINGAPORE | UGCP Singapore | No cross-region replication |
| SAO_PAULO | UGCP Sao Paulo | No cross-region replication |

Exception: Pseudonymised audit events replicated via Redpanda to all regions for LearningAgent analytics.

---

## B8. Webhook Contract

### Supported Events

| Event | Trigger | Payload Proto |
|-------|---------|---------------|
| `build.completed` | Build reaches COMPLETED | `BuildCompletedPayload` |
| `build.failed` | Build reaches FAILED | `BuildFailedPayload` |
| `gate.failed` | Quality gate fails | `GateFailedPayload` |
| `artifact.signed` | Artifact signed and published | `ArtifactSignedPayload` |
| `compliance.certified` | Compliance evidence generated | `ComplianceCertifiedPayload` |

### Delivery Contract

- **Protocol:** HTTPS POST (TLS 1.2+ required)
- **Content-Type:** `application/json`
- **Signature:** HMAC-SHA256 in `X-SDX-Signature` header. Computed as `hex(hmac_sha256(webhook_secret, request_body))`.
- **Timeout:** 10 seconds per delivery attempt
- **Retry policy:** 5 attempts with exponential backoff (0s, 5s, 25s, 2m, 15m)
- **Dead letter:** After 5 failures, event moves to dead letter queue (visible in dashboard)
- **Ordering:** Best-effort ordering per tenant. Not guaranteed across events.
- **Deduplication:** Each delivery has a unique `X-SDX-Delivery` header (ULID). Consumers should deduplicate.

### Request Headers

| Header | Description |
|--------|-------------|
| `Content-Type` | `application/json` |
| `X-SDX-Event` | Event type (e.g., `build.completed`) |
| `X-SDX-Signature` | HMAC-SHA256 hex digest |
| `X-SDX-Delivery` | Unique delivery ID (ULID) |
| `X-SDX-Timestamp` | Event timestamp (RFC3339) |
| `User-Agent` | `NeuroForge-SDX-Webhook/1.0` |

### Webhook Payload (JSON)

```json
{
  "event_id": "01JQKN9XYZABCDEF01234567",
  "event_type": "build.completed",
  "tenant_id": "01JQKNABC123DEF456789012",
  "build_id": "01JQKNXYZ987654321ABCDEF",
  "timestamp": "2026-03-21T14:30:00.000Z",
  "data": {
    "work_item_id": "01JQKN000111222333444555",
    "artifact_ids": ["01JQKNART1", "01JQKNART2"],
    "compliance_badges": {"SOC2_TYPE_II": true, "ISO_27001": true},
    "total_duration_ms": 7200000,
    "codebleu_score": 0.85,
    "coverage_pct": 93.2
  }
}
```

### Signature Verification (Consumer Side)

```python
import hmac, hashlib

def verify_signature(secret: str, body: bytes, signature: str) -> bool:
    expected = hmac.new(secret.encode(), body, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, signature)
```

### Security Constraints

- Webhook target URL MUST be HTTPS
- Webhook target MUST resolve to a public IP (no RFC1918/loopback)
- Redirects (301/302) are NOT followed (SSRF prevention)
- Response body is not inspected; 2xx = success

---

## B9. Rate Limiting Contract

### Limits by Plan Tier

| Tier | Requests/min | Concurrent Builds | Burst (10s) |
|------|-------------|-------------------|-------------|
| STARTER | 60 | 5 | 10 |
| GROWTH | 300 | 30 | 50 |
| ENTERPRISE | 1,000 | Unlimited | 200 |
| SOVEREIGN | Unlimited | Unlimited | Unlimited |

### Implementation

- **Algorithm:** Sliding window counter (DragonflyDB)
- **Key:** `ratelimit:{tenant_id}:{window_start}`
- **Window:** 60 seconds
- **Scope:** Per-tenant, not per-user

### Response Headers

| Header | Description | Example |
|--------|-------------|---------|
| `X-RateLimit-Limit` | Max requests per window | `300` |
| `X-RateLimit-Remaining` | Remaining requests | `247` |
| `X-RateLimit-Reset` | Unix timestamp when window resets | `1711036860` |
| `Retry-After` | Seconds until retry (only on 429) | `12` |

### 429 Response Body

```json
{
  "code": "SDX-9001",
  "message": "Rate limit exceeded",
  "details": "60 requests per minute for STARTER plan",
  "request_id": "01JQKN9XYZ...",
  "doc_url": "https://docs.sdx.stratogroup.io/errors/SDX-9001"
}
```

### Endpoint-Specific Overrides

| Endpoint | Override | Reason |
|----------|----------|--------|
| POST /work-items | 10/min (STARTER) | Prevent build spam |
| POST /webhooks/{id}/test | 5/min | Prevent abuse |
| GET /health | No limit | Monitoring |
| GET /openapi.json | No limit | Documentation |

---

## B10. Idempotency Contract

### Scope

All mutating REST endpoints (POST, PUT, PATCH, DELETE) support idempotency.

### Header

```
Idempotency-Key: {client-generated-uuid-v4}
```

### Behavior

1. **First request:** Execute normally. Cache response in DragonflyDB keyed by `idempotency:{tenant_id}:{key}`. TTL: 24 hours.
2. **Duplicate request (same key within TTL):** Return cached response. HTTP 200 (not 409). Same response body and headers.
3. **Key collision across tenants:** Keys are scoped per-tenant. No cross-tenant collision possible.
4. **Missing key:** Request proceeds without idempotency. Retries may produce duplicates.

### Implementation Details

| Aspect | Value |
|--------|-------|
| Storage | DragonflyDB |
| Key format | `idempotency:{tenant_id}:{idempotency_key}` |
| TTL | 24 hours |
| Cached data | HTTP status code + response headers + response body |
| Max key length | 128 chars |
| Key format | UUID v4 (36 chars, lowercase, with hyphens) |

### gRPC Idempotency

For gRPC, idempotency is handled at different levels:
- **Temporal activities:** Activity ID = `{build_id}-{stage}-{attempt}` provides built-in idempotency.
- **NATS JetStream:** `Nats-Msg-Id` header with 2-minute dedup window.
- **Repository level:** Content-addressed artifacts (SHA-256) are inherently idempotent.

### Error Cases

| Scenario | Behavior |
|----------|----------|
| Same key, different request body | Return cached response from first request (body mismatch is client error) |
| Key expired (>24h) | Request treated as new |
| Concurrent requests with same key | First to complete wins; second gets cached response |
| Server crash mid-processing | Key not cached; retry is treated as new request |

---

## B5. Pagination and Filtering Contract

### Cursor-Based Pagination

All list endpoints use cursor-based pagination (not offset-based).

**Request:**
```json
{
  "page_size": 20,
  "page_token": ""
}
```

**Response:**
```json
{
  "items": [...],
  "next_page_token": "eyJsYXN0X2lkIjoiMDFKUUtOOVhZWiIsInNvcnRfdmFsdWUiOiIyMDI2LTAzLTIxVDE0OjMwOjAwWiJ9",
  "total_count": 142
}
```

### Page Token Encoding

Token is base64url-encoded JSON:
```json
{
  "last_id": "01JQKN9XYZ",
  "sort_value": "2026-03-21T14:30:00Z",
  "direction": "DESC"
}
```

### Constraints

| Parameter | Constraint |
|-----------|-----------|
| page_size | 1-100 (default 20) |
| page_token | Opaque to client; base64url internally |
| total_count | Exact count (may be expensive; cached for 60s) |
| Empty page_token | Returns first page |
| Empty next_page_token | Indicates last page |

### Filtering

Filters are query parameters (REST) or message fields (gRPC):

| Pattern | Example |
|---------|---------|
| Enum filter | `?status=COMPLETED` |
| Date range | `?date_start=2026-01-01T00:00:00Z&date_end=2026-03-21T00:00:00Z` |
| Foreign key | `?build_id=01JQKN9XYZ` |
| Full-text search | `?search=authentication` |
| Sort | `?sort_by=created_at&sort_order=DESC` |

### Sorting

| Default Sort | Endpoints |
|-------------|-----------|
| `created_at DESC` | Most list endpoints |
| `timestamp DESC` | Audit log |
| `stage_number ASC` | Build stages |
