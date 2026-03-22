CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS workflows (
    tenant_id TEXT NOT NULL,
    workflow_id TEXT NOT NULL,
    status TEXT NOT NULL,
    current_stage TEXT NOT NULL,
    progress DOUBLE PRECISION NOT NULL,
    created_at TEXT NOT NULL,
    started_at TEXT,
    completed_at TEXT,
    estimated_completion TEXT,
    artifacts_json JSONB NOT NULL DEFAULT '{}'::JSONB,
    errors_json JSONB NOT NULL DEFAULT '[]'::JSONB,
    self_link TEXT,
    artifacts_link TEXT,
    cancel_link TEXT,
    PRIMARY KEY (tenant_id, workflow_id)
);

CREATE TABLE IF NOT EXISTS prds (
    tenant_id TEXT NOT NULL,
    prd_id TEXT NOT NULL,
    markdown TEXT NOT NULL,
    json_payload JSONB NOT NULL DEFAULT '{}'::JSONB,
    success_metrics_json JSONB NOT NULL DEFAULT '[]'::JSONB,
    user_stories_json JSONB NOT NULL DEFAULT '[]'::JSONB,
    technical_requirements_json JSONB NOT NULL DEFAULT '[]'::JSONB,
    created_at TEXT NOT NULL,
    PRIMARY KEY (tenant_id, prd_id)
);

CREATE TABLE IF NOT EXISTS idempotency_records (
    tenant_id TEXT NOT NULL,
    operation TEXT NOT NULL,
    idem_key TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    response_body BYTEA NOT NULL,
    response_digest TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, operation, idem_key)
);
