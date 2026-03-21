# NeuroForge SDX — Error Catalog

> All errors use the `SDXError` protobuf message (see common.proto).
> HTTP status codes map from gRPC status codes via grpc-gateway.
> Every error includes `request_id` and `doc_url` for support.

---

## Error Code Ranges

| Range | Category | HTTP Status |
|-------|----------|-------------|
| SDX-1xxx | Authentication & Authorization | 401, 403 |
| SDX-2xxx | Validation & Input | 400 |
| SDX-3xxx | Business Rule Violations | 409, 422 |
| SDX-4xxx | Resource Not Found / Conflict | 404, 409 |
| SDX-5xxx | Internal Server Errors | 500 |
| SDX-6xxx | Upstream Dependency Errors | 502, 503 |
| SDX-9xxx | Rate Limiting | 429 |

---

## SDX-1xxx: Authentication & Authorization

| Code | gRPC Status | HTTP | Message | Description |
|------|-------------|------|---------|-------------|
| SDX-1001 | UNAUTHENTICATED | 401 | Invalid or expired token | JWT is missing, malformed, or expired |
| SDX-1002 | PERMISSION_DENIED | 403 | Insufficient permissions | User lacks required role for this operation |
| SDX-1003 | PERMISSION_DENIED | 403 | Tenant mismatch | Requested resource belongs to a different tenant |
| SDX-1004 | UNAUTHENTICATED | 401 | Invalid API key | API key is invalid or revoked |
| SDX-1005 | PERMISSION_DENIED | 403 | Agent identity invalid | SPIFFE SVID is expired or untrusted |
| SDX-1006 | PERMISSION_DENIED | 403 | Admin role required | Operation requires ADMIN or OWNER role |

## SDX-2xxx: Validation & Input

| Code | gRPC Status | HTTP | Message | Description |
|------|-------------|------|---------|-------------|
| SDX-2001 | INVALID_ARGUMENT | 400 | Invalid input | Generic input validation failure. Check `fields` array. |
| SDX-2002 | INVALID_ARGUMENT | 400 | Unsupported language | Requested language not in plan allowance |
| SDX-2003 | INVALID_ARGUMENT | 400 | Invalid plan tier | Plan tier value is not recognized |
| SDX-2004 | INVALID_ARGUMENT | 400 | Invalid NexoRegion | Region value is not recognized |
| SDX-2005 | INVALID_ARGUMENT | 400 | Invalid filter | Query filter parameter is malformed |
| SDX-2006 | INVALID_ARGUMENT | 400 | Invalid webhook URL | URL must be HTTPS and resolve to public IP |
| SDX-2007 | INVALID_ARGUMENT | 400 | Unknown event type | Webhook event type is not recognized |
| SDX-2008 | INVALID_ARGUMENT | 400 | Unknown compliance framework | Compliance framework is not supported |
| SDX-2009 | INVALID_ARGUMENT | 400 | Invalid date range | start must be before end |
| SDX-2010 | INVALID_ARGUMENT | 400 | Invalid tenant name | Name must be 3-100 chars, unique |
| SDX-2011 | INVALID_ARGUMENT | 400 | Missing billing account | billing_account_ref is required |
| SDX-2012 | INVALID_ARGUMENT | 400 | Weak webhook secret | Secret must be ≥ 32 bytes |
| SDX-2013 | INVALID_ARGUMENT | 400 | Invalid agent type | Agent type is not recognized |
| SDX-2014 | INVALID_ARGUMENT | 400 | Invalid SPIFFE ID | SPIFFE ID format is invalid |
| SDX-2015 | INVALID_ARGUMENT | 400 | Invalid model ID | Model ID is not recognized |
| SDX-2016 | INVALID_ARGUMENT | 400 | Same model for primary and fallback | Primary and fallback must differ |
| SDX-2017 | INVALID_ARGUMENT | 400 | Content too large | Raw content exceeds 5MB limit |
| SDX-2018 | INVALID_ARGUMENT | 400 | Invalid page size | page_size must be 1-100 |

## SDX-3xxx: Business Rule Violations

| Code | gRPC Status | HTTP | Message | Description |
|------|-------------|------|---------|-------------|
| SDX-3001 | FAILED_PRECONDITION | 422 | Tenant suspended | Tenant is suspended; no new operations |
| SDX-3002 | RESOURCE_EXHAUSTED | 422 | Quota exhausted | Tenant quota for this resource type is exhausted |
| SDX-3003 | FAILED_PRECONDITION | 422 | Ambiguity already resolved | This ambiguity has already been resolved |
| SDX-3004 | FAILED_PRECONDITION | 422 | WorkItem not approved | WorkItem must be APPROVED to initiate build |
| SDX-3005 | FAILED_PRECONDITION | 422 | SBOM missing | Artifact requires SBOM before signing |
| SDX-3006 | FAILED_PRECONDITION | 422 | Provenance invalid | SLSA Level 3 provenance requirements not met |
| SDX-3007 | FAILED_PRECONDITION | 422 | Max remediation retries exceeded | Gate failed after 3 remediation attempts |
| SDX-3008 | FAILED_PRECONDITION | 422 | Max recovery attempts exceeded | Build recovery failed after 5 attempts |
| SDX-3009 | FAILED_PRECONDITION | 422 | Artifact revoked | Artifact has been revoked and cannot be accessed |
| SDX-3010 | RESOURCE_EXHAUSTED | 422 | Token limit exceeded | LLM token budget for this request exceeded |
| SDX-3011 | FAILED_PRECONDITION | 422 | Smoke test failed | Deployment smoke tests did not pass |
| SDX-3012 | FAILED_PRECONDITION | 422 | SLO violation | Deployment SLO validation failed |
| SDX-3013 | DATA_LOSS | 500 | Audit chain broken | Audit hash chain integrity compromised (critical) |
| SDX-3014 | FAILED_PRECONDITION | 422 | Usage exceeds new quota | Cannot downgrade; current usage exceeds new plan limits |
| SDX-3015 | FAILED_PRECONDITION | 422 | Invalid plan transition | This plan change is not allowed |
| SDX-3016 | FAILED_PRECONDITION | 422 | Invalid state transition | Aggregate cannot transition to requested state |
| SDX-3017 | FAILED_PRECONDITION | 422 | Non-compliant license | Dependency uses a non-compliant license (Zero Software Tax) |
| SDX-3018 | FAILED_PRECONDITION | 422 | Build immutable | Completed or failed build cannot be modified |

## SDX-4xxx: Resource Not Found / Conflict

| Code | gRPC Status | HTTP | Message | Description |
|------|-------------|------|---------|-------------|
| SDX-4001 | NOT_FOUND | 404 | WorkItem not found | No WorkItem with this ID exists for this tenant |
| SDX-4002 | NOT_FOUND | 404 | Ambiguity not found | No Ambiguity with this ID exists on this WorkItem |
| SDX-4003 | NOT_FOUND | 404 | Build not found | No Build with this ID exists for this tenant |
| SDX-4004 | NOT_FOUND | 404 | Artifact not found | No Artifact with this ID exists for this tenant |
| SDX-4005 | NOT_FOUND | 404 | Audit entries not found | No audit entries match the query |
| SDX-4006 | NOT_FOUND | 404 | Tenant not found | No Tenant with this ID exists |
| SDX-4007 | NOT_FOUND | 404 | Agent not found | No Agent with this ID exists |
| SDX-4008 | NOT_FOUND | 404 | Webhook not found | No Webhook with this ID exists |
| SDX-4009 | NOT_FOUND | 404 | Deployment not found | No Deployment with this ID exists |
| SDX-4010 | NOT_FOUND | 404 | Workspace not found | No Workspace with this ID exists |
| SDX-4011 | ALREADY_EXISTS | 409 | Tenant name already exists | Tenant name must be unique |
| SDX-4012 | ALREADY_EXISTS | 409 | Duplicate idempotency key | Request with this key already processed; returning cached response |

## SDX-5xxx: Internal Server Errors

| Code | gRPC Status | HTTP | Message | Description |
|------|-------------|------|---------|-------------|
| SDX-5001 | UNAVAILABLE | 503 | No available agent | No agent of the required type is available |
| SDX-5002 | DEADLINE_EXCEEDED | 504 | Agent timeout | Agent did not respond within timeout |
| SDX-5003 | INTERNAL | 500 | Gate evaluation error | Unexpected error evaluating quality gate |
| SDX-5004 | INTERNAL | 500 | Signing failed | Artifact signing operation failed |
| SDX-5005 | DATA_LOSS | 500 | Checkpoint corrupted | Build checkpoint hash verification failed |
| SDX-5006 | INTERNAL | 500 | Provisioning failed | Tenant infrastructure provisioning failed |
| SDX-5007 | INTERNAL | 500 | Evidence generation failed | Compliance evidence generation failed |
| SDX-5099 | INTERNAL | 500 | Internal error | Unclassified internal error |

## SDX-6xxx: Upstream Dependency Errors

| Code | gRPC Status | HTTP | Message | Description |
|------|-------------|------|---------|-------------|
| SDX-6001 | UNAVAILABLE | 503 | All LLM models unavailable | All models (cloud + on-prem) are unreachable |
| SDX-6002 | UNAVAILABLE | 502 | PagerDuty unavailable | Cannot create escalation incident |
| SDX-6003 | UNAVAILABLE | 502 | Webhook ping failed | Webhook target URL is unreachable |
| SDX-6004 | DEADLINE_EXCEEDED | 504 | LLM timeout | LLM request exceeded timeout |
| SDX-6005 | UNAVAILABLE | 502 | Fleet unavailable | GitOps deployment system unreachable |
| SDX-6006 | UNAVAILABLE | 502 | Vault unavailable | Secrets management system unreachable |

## SDX-9xxx: Rate Limiting

| Code | gRPC Status | HTTP | Message | Description |
|------|-------------|------|---------|-------------|
| SDX-9001 | RESOURCE_EXHAUSTED | 429 | Rate limit exceeded | Too many requests; check X-RateLimit-Reset header |
| SDX-9002 | RESOURCE_EXHAUSTED | 429 | Concurrent build limit | Max concurrent builds for this plan tier reached |
