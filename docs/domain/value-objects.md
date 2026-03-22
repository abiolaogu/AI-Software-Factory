# NeuroForge SDX — Value Objects

> All value objects are immutable, equality-by-value, and self-validating.
> Codex MUST implement these as structs (Go) or frozen dataclasses (Python) with validation in constructors.

---

## Identity Value Objects

### WorkItemID
```
Type: ULID (26-char Crockford Base32)
Format: 01JQKN9XYZABCDEF01234567
Validation: Must be valid ULID; monotonically sortable by creation time.
Zero value: Not permitted. Must be generated at aggregate creation.
```

### BuildID
```
Type: ULID
Same constraints as WorkItemID.
```

### ArtifactID
```
Type: ULID
Same constraints as WorkItemID.
```

### TenantID
```
Type: ULID
Same constraints as WorkItemID.
```

### AgentID
```
Type: ULID
Same constraints as WorkItemID.
```

### WorkspaceID
```
Type: ULID
Same constraints as WorkItemID.
```

### EventID
```
Type: ULID
Generated per domain event. Must be globally unique.
```

### GateID
```
Type: string enum
Values: G-01, G-02, G-03, G-04, G-05, G-06, G-07, G-08, G-09, G-10, G-11, G-12, G-13, G-14
Validation: Must be one of the 14 defined gate identifiers.
```

### StageNumber
```
Type: uint8
Range: 1..16
Validation: Must be within range. Maps to pipeline stage name.
Mapping:
  1=Ingestion, 2=DomainModelling, 3=ArchitectureDesign, 4=ContractGeneration,
  5=TestSpecification, 6=CodeGeneration, 7=StaticAnalysis, 8=TestExecution,
  9=SecurityScanning, 10=ComplianceMapping, 11=PerformanceTesting,
  12=Documentation, 13=InfrastructureAsCode, 14=ArtifactSigning,
  15=Deployment, 16=LearningLoop
```

---

## Tenant & Auth Value Objects

### TenantContext
```
Fields:
  tenant_id: TenantID (required)
  workspace_id: WorkspaceID (optional; scoped operations)
  compliance_profile_id: string (optional; override for build)
  nexo_region: NexoRegion (required)

Propagation: Injected into every gRPC metadata, NATS message header, and HTTP request header.
Validation: tenant_id must resolve to an ACTIVE tenant. nexo_region must match tenant's registered region.
```

### NexoRegion
```
Type: string enum
Values: LAGOS, LONDON, ASHBURN, SINGAPORE, SAO_PAULO
Validation: Must be one of the 5 defined regions.
```

### PlanTier
```
Type: string enum
Values: STARTER, GROWTH, ENTERPRISE, SOVEREIGN
Ordering: STARTER < GROWTH < ENTERPRISE < SOVEREIGN (for quota comparison)
```

### TenantQuotas
```
Fields:
  max_pipelines_per_month: uint32
  max_languages: uint8
  max_storage_gb: uint32
  max_concurrent_agents: uint16
  allowed_languages: []LanguageCode

Defaults by PlanTier:
  STARTER:    { 5,  3, 10,  2, [GO, PYTHON, TYPESCRIPT] }
  GROWTH:     { 30, 12, 100, 10, ALL }
  ENTERPRISE: { UNLIMITED, 12, UNLIMITED, UNLIMITED, ALL }
  SOVEREIGN:  { UNLIMITED, 12, UNLIMITED, UNLIMITED, ALL }
```

### SPIFFEID
```
Fields:
  trust_domain: string (e.g., "sdx.stratogroup.io")
  path: string (e.g., "/agent/builder/01JQKN9XYZ")

Format: spiffe://{trust_domain}{path}
Validation: Must conform to SPIFFE ID specification. Trust domain must be registered.
```

---

## Cryptographic & Integrity Value Objects

### ContentHash
```
Fields:
  algorithm: string (always "SHA-256")
  hex_digest: string (64 lowercase hex chars)

Validation: hex_digest must be exactly 64 characters, lowercase hex.
Equality: Two ContentHashes are equal iff their hex_digests are identical.
Usage: Artifact content-addressing, checkpoint verification, audit log chaining.
```

### SignatureRef
```
Fields:
  signer: string (always "COSIGN")
  signature: []byte (DER-encoded signature)
  certificate: string (PEM-encoded Fulcio certificate)
  transparency_log_entry: string (Rekor log index URL)

Validation: signature must verify against certificate. certificate must chain to Fulcio root.
```

### ProvenanceRef
```
Fields:
  slsa_level: uint8 (always 3)
  builder_id: string (e.g., "sdx-artifact-svc@v1.0.0")
  build_type: string (e.g., "https://sdx.stratogroup.io/build/v1")
  invocation: ProvenanceInvocation
  materials: []ProvenanceMaterial
  metadata: map[string]string

Sub-objects:
  ProvenanceInvocation: { config_source, parameters, environment }
  ProvenanceMaterial: { uri, digest: ContentHash }

Validation: slsa_level must be 3. materials must include all input artifacts with verified digests.
```

### SBOMRef
```
Fields:
  format: string enum (SPDX | CYCLONEDX)
  generator: string (always "SYFT")
  storage_uri: S3URI
  content_hash: ContentHash

Validation: format must be SPDX or CYCLONEDX. content_hash must verify against stored file.
```

---

## Storage Value Objects

### S3URI
```
Fields:
  bucket: string
  key: string
  region: NexoRegion

Format: s3://{bucket}/{key}
Validation: bucket must match pattern ^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$. key must not be empty.
Tenant scoping: key must start with /{tenant_id}/ for tenant-owned objects.
```

### VaultMountPath
```
Fields:
  path: string

Format: secret/sdx/{tenant_id}/{scope}
Validation: Must start with "secret/sdx/". Must contain a valid tenant_id segment.
```

### K8sNamespace
```
Fields:
  name: string
  cluster: string

Format: sdx-{tenant_id_short}
Validation: name must be a valid Kubernetes namespace (≤63 chars, lowercase alphanumeric + hyphens).
```

---

## Temporal & Date/Time Value Objects

### Timestamp
```
Type: time.Time (Go) / datetime with timezone (Python)
Format: RFC 3339 with nanosecond precision. Always UTC.
Example: "2026-03-21T14:30:00.123456789Z"
Validation: Must not be zero value. Must not be in the future (for event timestamps).
Serialisation: Protobuf google.protobuf.Timestamp. JSON: RFC 3339 string.
```

### Duration
```
Type: time.Duration (Go) / timedelta (Python)
Format: Protobuf google.protobuf.Duration.
Usage: Timeout configs, retry backoff, heartbeat intervals.
Validation: Must be positive. Max value depends on context (e.g., max workflow duration = 24h).
```

### DateRange
```
Fields:
  start: Timestamp
  end: Timestamp

Validation: start must be before end. Both must be non-zero.
Usage: Billing periods, audit log queries, metrics windows.
```

---

## Monetary & Metering Value Objects

### Money
```
Fields:
  amount: int64 (in smallest unit — cents for USD)
  currency: CurrencyCode

Validation: amount must be non-negative. currency must be a valid ISO 4217 code.
Supported currencies: USD, EUR, GBP, NGN, SGD, BRL.
Arithmetic: Addition only between same currency. No implicit conversion.
```

### CurrencyCode
```
Type: string (3-char ISO 4217)
Values: USD, EUR, GBP, NGN, SGD, BRL
```

### TokenUsage
```
Fields:
  input_tokens: uint64
  output_tokens: uint64
  model: ModelID
  cost_microdollars: uint64 (cost in 1/1,000,000 of a dollar)

Validation: All fields must be non-negative. model must be a registered ModelID.
Usage: Per-build, per-tenant LLM cost attribution.
```

### StorageUsage
```
Fields:
  bytes_used: uint64
  object_count: uint64
  measured_at: Timestamp

Usage: Tenant billing meter for artifact storage.
```

### BillingMeter
```
Fields:
  meter_type: BillingMeterType enum (PIPELINES | LLM_TOKENS | COMPUTE_MINUTES | STORAGE_GB)
  current_value: uint64
  limit: uint64 (0 = unlimited)
  period_start: Timestamp
  period_end: Timestamp

Validation: current_value must not exceed limit (unless limit == 0).
Reset: Meters reset at period_end.
```

---

## Quality & Metrics Value Objects

### CodeBLEUScore
```
Fields:
  score: float64 (0.0..1.0)
  n_gram_match: float64
  syntax_match: float64
  semantic_match: float64
  reference_corpus: string (identifier)

Validation: score must be in [0.0, 1.0]. All sub-scores must be in [0.0, 1.0].
Gate threshold: score >= 0.80 (G-01)
```

### CyclomaticComplexityScore
```
Fields:
  max_per_function: uint32
  average: float64
  functions_exceeding_threshold: uint32
  total_functions: uint32

Validation: max_per_function and average must be non-negative.
Gate threshold: max_per_function <= 10 (G-02)
```

### CoverageReport
```
Fields:
  line_coverage_pct: float64 (0.0..100.0)
  branch_coverage_pct: float64 (0.0..100.0)
  lines_covered: uint64
  lines_total: uint64
  branches_covered: uint64
  branches_total: uint64

Validation: Percentages must be in [0.0, 100.0]. Counts must be consistent with percentages.
Gate threshold: line >= 90.0, branch >= 85.0 (G-05)
```

### MutationScore
```
Fields:
  score_pct: float64 (0.0..100.0)
  mutants_killed: uint64
  mutants_total: uint64
  mutants_survived: uint64

Validation: score_pct = (mutants_killed / mutants_total) * 100. mutants_total = mutants_killed + mutants_survived.
Gate threshold: score_pct >= 70.0 (G-06)
```

### GateResult
```
Fields:
  gate_id: GateID
  metric_name: string
  value: float64
  threshold: float64
  operator: ComparisonOperator enum (GTE | LTE | EQ | LT | GT)
  passed: bool
  evaluated_at: Timestamp

Validation: passed must be consistent with value, threshold, and operator.
```

---

## Model Routing Value Objects

### ModelID
```
Type: string enum
Values: CLAUDE_OPUS_4, CLAUDE_SONNET_4, CLAUDE_HAIKU, GPT_4O, GEMINI_ULTRA, MISTRAL_LARGE, LLAMA_3_1_70B, LLAMA_3_1_8B, MIXTRAL_8X22B, MIXTRAL_8X7B, NOMIC_EMBED, TEXT_EMBEDDING_3_LARGE
```

### ModelRoutingConfig
```
Fields:
  primary_model: ModelID
  fallback_model: ModelID
  on_prem_trigger: OnPremTrigger (NEVER | LATENCY_EXCEEDS_3S | ALWAYS_ON_PREM | COST_OPTIMISED)
  max_tokens: uint32
  temperature: float64 (0.0..2.0)
  timeout: Duration

Validation: primary_model != fallback_model. temperature in [0.0, 2.0]. max_tokens > 0.
```

### HealthStatus
```
Fields:
  status: HealthState enum (HEALTHY | DEGRADED | UNHEALTHY)
  last_check: Timestamp
  latency_ms: uint32
  error_count: uint32
  consecutive_failures: uint32

Derivation:
  HEALTHY: consecutive_failures == 0 and latency_ms < 1000
  DEGRADED: consecutive_failures in [1..2] or latency_ms in [1000..3000]
  UNHEALTHY: consecutive_failures >= 3 or latency_ms > 3000
```

---

## Event Envelope Value Object

### DomainEventEnvelope
```
Fields:
  event_id: EventID (ULID)
  event_type: string (e.g., "sdx.build.stage.completed")
  aggregate_id: string (the affected aggregate's ID)
  aggregate_type: string (e.g., "Build")
  tenant_id: TenantID
  build_id: BuildID (optional; not all events are build-scoped)
  timestamp: Timestamp
  payload: bytes (serialised domain event)
  metadata: map[string]string

Required metadata keys:
  - trace_id: W3C TraceContext trace ID
  - span_id: W3C TraceContext span ID
  - agent_id: AgentID of the producing agent (if applicable)
  - source_service: service name that emitted the event
  - schema_version: semantic version of the payload schema

Validation: event_id must be unique (ULID guarantees). tenant_id must be present. timestamp must be non-zero.
```

---

## Audit Value Objects

### AuditEntry
```
Fields:
  entry_id: EventID (ULID)
  event_type: string
  actor_type: ActorType enum (AGENT | HUMAN | SYSTEM)
  actor_id: string (AgentID or human user ID)
  target_type: string (aggregate type)
  target_id: string (aggregate ID)
  action: string (e.g., "created", "updated", "signed", "deployed")
  tenant_id: TenantID
  build_id: BuildID (optional)
  timestamp: Timestamp
  prev_hash: ContentHash (SHA-256 of previous AuditEntry)
  details: map[string]string

Validation: prev_hash must chain to the immediately preceding entry. entry_id must be monotonically increasing.
Immutability: Once written, an AuditEntry MUST NOT be modified or deleted. EVER.
```

### AuditChainVerification
```
Fields:
  start_entry_id: EventID
  end_entry_id: EventID
  entries_verified: uint64
  chain_valid: bool
  first_broken_entry_id: EventID (optional; only if chain_valid == false)

Usage: Nightly integrity verification job output.
```
