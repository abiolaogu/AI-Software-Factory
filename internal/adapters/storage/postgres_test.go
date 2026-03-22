package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stratogroup/neuroforge-sdx/internal/ports"
	"github.com/stretchr/testify/require"
)

func TestPostgresStorePrepareAndHelpers(t *testing.T) {
	store, mock, closeFn := newMockPostgresStore(t)
	defer closeFn()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(1) FROM schema_migrations WHERE version = $1`)).
		WithArgs("001_initial.sql").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	applied, err := store.migrationApplied(context.Background(), "001_initial.sql")
	require.NoError(t, err)
	require.True(t, applied)
	require.Equal(t, 3, totalPages(5, 2))
	require.Equal(t, 0, totalPages(0, 20))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresStoreWorkflowOperations(t *testing.T) {
	store, mock, closeFn := newMockPostgresStore(t)
	defer closeFn()

	artifacts := map[string]domain.ArtifactDescriptor{
		"manifest": {
			URL:       "https://example.com/manifest.json",
			Type:      "application/json",
			Size:      1,
			CreatedAt: "2026-03-21T00:00:00Z",
		},
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO workflows (`)).
		WithArgs(
			"tenant-a", "workflow-1", "queued", "init", 0.1,
			"2026-03-21T00:00:00Z", "", "", "",
			sqlmock.AnyArg(), sqlmock.AnyArg(), "", "", "",
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, store.SaveWorkflow(context.Background(), "tenant-a", domain.WorkflowResponse{
		WorkflowID:   "workflow-1",
		Status:       "queued",
		CurrentStage: domain.WorkflowStageInit,
		Progress:     0.1,
		CreatedAt:    "2026-03-21T00:00:00Z",
	}, artifacts))

	artifactsJSON, err := json.Marshal(artifacts)
	require.NoError(t, err)
	errorsJSON, err := json.Marshal([]string{})
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{
		"workflow_id", "status", "current_stage", "progress", "created_at", "started_at", "completed_at", "estimated_completion",
		"artifacts_json", "errors_json", "self_link", "artifacts_link", "cancel_link",
	}).AddRow("workflow-1", "queued", "init", 0.1, "2026-03-21T00:00:00Z", "", "", "", artifactsJSON, errorsJSON, "", "", "")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT workflow_id, status, current_stage, progress, created_at, COALESCE(started_at, ''), COALESCE(completed_at, ''), COALESCE(estimated_completion, ''),
				artifacts_json, errors_json, COALESCE(self_link, ''), COALESCE(artifacts_link, ''), COALESCE(cancel_link, '')
				FROM workflows
				WHERE tenant_id = $1 AND ($2 = '' OR status = $2)
				ORDER BY created_at DESC
				LIMIT $3 OFFSET $4`)).
		WithArgs("tenant-a", "", 20, 0).
		WillReturnRows(rows)

	workflows, pagination, err := store.ListWorkflows(context.Background(), ports.WorkflowFilter{TenantID: "tenant-a", Page: 1, Size: 20})
	require.NoError(t, err)
	require.Len(t, workflows, 1)
	require.Equal(t, 1, pagination.Total)

	row := sqlmock.NewRows([]string{
		"workflow_id", "status", "current_stage", "progress", "created_at", "started_at", "completed_at", "estimated_completion",
		"artifacts_json", "errors_json", "self_link", "artifacts_link", "cancel_link",
	}).AddRow("workflow-1", "queued", "init", 0.1, "2026-03-21T00:00:00Z", "", "", "", artifactsJSON, errorsJSON, "", "", "")
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT workflow_id, status, current_stage, progress, created_at, COALESCE(started_at, ''), COALESCE(completed_at, ''), COALESCE(estimated_completion, ''),
				artifacts_json, errors_json, COALESCE(self_link, ''), COALESCE(artifacts_link, ''), COALESCE(cancel_link, '')
				FROM workflows WHERE tenant_id = $1 AND workflow_id = $2`)).
		WithArgs("tenant-a", "workflow-1").
		WillReturnRows(row)

	workflow, err := store.GetWorkflow(context.Background(), "tenant-a", "workflow-1")
	require.NoError(t, err)
	require.Equal(t, "workflow-1", workflow.WorkflowID)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT artifacts_json FROM workflows WHERE tenant_id = $1 AND workflow_id = $2`)).
		WithArgs("tenant-a", "workflow-1").
		WillReturnRows(sqlmock.NewRows([]string{"artifacts_json"}).AddRow(artifactsJSON))

	foundArtifacts, err := store.GetWorkflowArtifacts(context.Background(), "tenant-a", "workflow-1")
	require.NoError(t, err)
	require.Contains(t, foundArtifacts.Artifacts, "manifest")

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE workflows
				SET status = 'failed', current_stage = 'failed', errors_json = '["workflow cancelled by operator"]'::jsonb
				WHERE tenant_id = $1 AND workflow_id = $2`)).
		WithArgs("tenant-a", "workflow-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, store.CancelWorkflow(context.Background(), "tenant-a", "workflow-1"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresStorePRDAndIdempotencyOperations(t *testing.T) {
	store, mock, closeFn := newMockPostgresStore(t)
	defer closeFn()

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO prds (`)).
		WithArgs(
			"tenant-a", "prd-1", "# PRD",
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			"2026-03-21T00:00:00Z",
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, store.SavePRD(context.Background(), "tenant-a", domain.PRDResponse{
		PRDID:                 "prd-1",
		Markdown:              "# PRD",
		JSON:                  map[string]any{"tenant_id": "tenant-a"},
		SuccessMetrics:        []domain.SuccessMetric{{Name: "coverage", Target: ">=90%", Measurement: "line"}},
		UserStories:           []string{"As a lead, I can submit prompts."},
		TechnicalRequirements: []string{"OpenAPI stays in sync."},
		CreatedAt:             "2026-03-21T00:00:00Z",
	}))

	jsonPayload, err := json.Marshal(map[string]any{"tenant_id": "tenant-a"})
	require.NoError(t, err)
	metricsPayload, err := json.Marshal([]domain.SuccessMetric{{Name: "coverage", Target: ">=90%", Measurement: "line"}})
	require.NoError(t, err)
	storiesPayload, err := json.Marshal([]string{"As a lead, I can submit prompts."})
	require.NoError(t, err)
	requirementsPayload, err := json.Marshal([]string{"OpenAPI stays in sync."})
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT prd_id, markdown, json_payload, success_metrics_json, user_stories_json, technical_requirements_json, created_at FROM prds WHERE tenant_id = $1 AND prd_id = $2`)).
		WithArgs("tenant-a", "prd-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"prd_id", "markdown", "json_payload", "success_metrics_json", "user_stories_json", "technical_requirements_json", "created_at",
		}).AddRow("prd-1", "# PRD", jsonPayload, metricsPayload, storiesPayload, requirementsPayload, "2026-03-21T00:00:00Z"))

	prd, err := store.GetPRD(context.Background(), "tenant-a", "prd-1")
	require.NoError(t, err)
	require.Equal(t, "prd-1", prd.PRDID)

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO idempotency_records (tenant_id, operation, idem_key, status_code, response_body, response_digest)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (tenant_id, operation, idem_key) DO UPDATE SET
					status_code = EXCLUDED.status_code,
					response_body = EXCLUDED.response_body,
					response_digest = EXCLUDED.response_digest`)).
		WithArgs("tenant-a", "createWorkflow", "idem-1", 201, []byte(`{"workflow_id":"workflow-1"}`), "digest-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, store.SaveIdempotencyRecord(context.Background(), domain.IdempotencyRecord{
		TenantID:       "tenant-a",
		Operation:      "createWorkflow",
		Key:            "idem-1",
		StatusCode:     201,
		ResponseBody:   []byte(`{"workflow_id":"workflow-1"}`),
		ResponseDigest: "digest-1",
	}))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status_code, response_body, response_digest FROM idempotency_records WHERE tenant_id = $1 AND operation = $2 AND idem_key = $3`)).
		WithArgs("tenant-a", "createWorkflow", "idem-1").
		WillReturnRows(sqlmock.NewRows([]string{"status_code", "response_body", "response_digest"}).AddRow(201, []byte(`{"workflow_id":"workflow-1"}`), "digest-1"))

	record, found, err := store.GetIdempotencyRecord(context.Background(), "tenant-a", "createWorkflow", "idem-1")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 201, record.StatusCode)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestScanWorkflowErrors(t *testing.T) {
	_, err := scanWorkflow(sqlmockRow{err: sql.ErrNoRows})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestPostgresStorePingCloseAndMigrate(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)

	store := &PostgresStore{db: db}

	mock.ExpectPing()
	require.NoError(t, store.Ping(context.Background()))

	mock.ExpectExec(regexp.QuoteMeta(`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(1) FROM schema_migrations WHERE version = $1`)).
		WithArgs("001_initial.sql").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`CREATE TABLE IF NOT EXISTS workflows (`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO schema_migrations (version) VALUES ($1)`)).
		WithArgs("001_initial.sql").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	require.NoError(t, store.migrate(context.Background()))

	mock.ExpectClose()
	require.NoError(t, store.Close())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresStoreWorkflowAndPRDErrorPaths(t *testing.T) {
	store, mock, closeFn := newMockPostgresStore(t)
	defer closeFn()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE workflows
				SET status = 'failed', current_stage = 'failed', errors_json = '["workflow cancelled by operator"]'::jsonb
				WHERE tenant_id = $1 AND workflow_id = $2`)).
		WithArgs("tenant-a", "workflow-1").
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected failed")))

	require.Error(t, store.CancelWorkflow(context.Background(), "tenant-a", "workflow-1"))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT artifacts_json FROM workflows WHERE tenant_id = $1 AND workflow_id = $2`)).
		WithArgs("tenant-a", "missing").
		WillReturnError(sql.ErrNoRows)

	_, err := store.GetWorkflowArtifacts(context.Background(), "tenant-a", "missing")
	require.ErrorIs(t, err, domain.ErrNotFound)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT artifacts_json FROM workflows WHERE tenant_id = $1 AND workflow_id = $2`)).
		WithArgs("tenant-a", "workflow-1").
		WillReturnRows(sqlmock.NewRows([]string{"artifacts_json"}).AddRow([]byte(`{`)))

	_, err = store.GetWorkflowArtifacts(context.Background(), "tenant-a", "workflow-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "decoding workflow artifacts failed")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT prd_id, markdown, json_payload, success_metrics_json, user_stories_json, technical_requirements_json, created_at FROM prds WHERE tenant_id = $1 AND prd_id = $2`)).
		WithArgs("tenant-a", "missing").
		WillReturnError(sql.ErrNoRows)

	_, err = store.GetPRD(context.Background(), "tenant-a", "missing")
	require.ErrorIs(t, err, domain.ErrNotFound)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT prd_id, markdown, json_payload, success_metrics_json, user_stories_json, technical_requirements_json, created_at FROM prds WHERE tenant_id = $1 AND prd_id = $2`)).
		WithArgs("tenant-a", "prd-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"prd_id", "markdown", "json_payload", "success_metrics_json", "user_stories_json", "technical_requirements_json", "created_at",
		}).AddRow("prd-1", "# PRD", []byte(`{`), []byte(`[]`), []byte(`[]`), []byte(`[]`), "2026-03-21T00:00:00Z"))

	_, err = store.GetPRD(context.Background(), "tenant-a", "prd-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "decoding prd json failed")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresStoreSaveAndListErrorPaths(t *testing.T) {
	store, mock, closeFn := newMockPostgresStore(t)
	defer closeFn()

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO workflows (`)).
		WithArgs(
			"tenant-a", "workflow-1", "queued", "init", 0.1,
			"2026-03-21T00:00:00Z", "", "", "",
			sqlmock.AnyArg(), sqlmock.AnyArg(), "", "", "",
		).
		WillReturnError(errors.New("insert failed"))

	err := store.SaveWorkflow(context.Background(), "tenant-a", domain.WorkflowResponse{
		WorkflowID:   "workflow-1",
		Status:       "queued",
		CurrentStage: domain.WorkflowStageInit,
		Progress:     0.1,
		CreatedAt:    "2026-03-21T00:00:00Z",
	}, map[string]domain.ArtifactDescriptor{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "executing workflow upsert failed")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT workflow_id, status, current_stage, progress, created_at, COALESCE(started_at, ''), COALESCE(completed_at, ''), COALESCE(estimated_completion, ''),
				artifacts_json, errors_json, COALESCE(self_link, ''), COALESCE(artifacts_link, ''), COALESCE(cancel_link, '')
				FROM workflows
				WHERE tenant_id = $1 AND ($2 = '' OR status = $2)
				ORDER BY created_at DESC
				LIMIT $3 OFFSET $4`)).
		WithArgs("tenant-a", "", 20, 0).
		WillReturnError(errors.New("select failed"))

	_, _, err = store.ListWorkflows(context.Background(), ports.WorkflowFilter{TenantID: "tenant-a"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "querying workflows failed")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresStoreIdempotencyAndPRDSaveErrorPaths(t *testing.T) {
	store, mock, closeFn := newMockPostgresStore(t)
	defer closeFn()

	err := store.SavePRD(context.Background(), "tenant-a", domain.PRDResponse{
		PRDID:    "prd-1",
		Markdown: "# PRD",
		JSON: map[string]any{
			"invalid": func() {},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "marshalling prd json failed")

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO idempotency_records (tenant_id, operation, idem_key, status_code, response_body, response_digest)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (tenant_id, operation, idem_key) DO UPDATE SET
					status_code = EXCLUDED.status_code,
					response_body = EXCLUDED.response_body,
					response_digest = EXCLUDED.response_digest`)).
		WithArgs("tenant-a", "createWorkflow", "idem-1", 201, []byte(`{"workflow_id":"workflow-1"}`), "digest-1").
		WillReturnError(errors.New("save failed"))

	err = store.SaveIdempotencyRecord(context.Background(), domain.IdempotencyRecord{
		TenantID:       "tenant-a",
		Operation:      "createWorkflow",
		Key:            "idem-1",
		StatusCode:     201,
		ResponseBody:   []byte(`{"workflow_id":"workflow-1"}`),
		ResponseDigest: "digest-1",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "saving idempotency record failed")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT status_code, response_body, response_digest FROM idempotency_records WHERE tenant_id = $1 AND operation = $2 AND idem_key = $3`)).
		WithArgs("tenant-a", "createWorkflow", "idem-1").
		WillReturnError(errors.New("lookup failed"))

	_, found, err := store.GetIdempotencyRecord(context.Background(), "tenant-a", "createWorkflow", "idem-1")
	require.Error(t, err)
	require.False(t, found)
	require.Contains(t, err.Error(), "loading idempotency record failed")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNewPostgresStoreUsesInjectedOpen(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
	}()

	original := openDB
	t.Cleanup(func() {
		openDB = original
	})
	openDB = func(driverName, dsn string) (*sql.DB, error) {
		require.Equal(t, "pgx", driverName)
		require.Equal(t, "postgres://tenant@localhost:5433/yugabyte?sslmode=disable", dsn)
		return db, nil
	}

	mock.ExpectPing()
	mock.ExpectExec(regexp.QuoteMeta(`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(1) FROM schema_migrations WHERE version = $1`)).
		WithArgs("001_initial.sql").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	expectPrepare(mock)

	store, err := NewPostgresStore(context.Background(), "postgres://tenant@localhost:5433/yugabyte?sslmode=disable")
	require.NoError(t, err)
	require.NotNil(t, store)
	require.NoError(t, mock.ExpectationsWereMet())
}

type sqlmockRow struct {
	err error
}

func (r sqlmockRow) Scan(_ ...any) error {
	return r.err
}

func newMockPostgresStore(t *testing.T) (*PostgresStore, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	store := &PostgresStore{db: db}
	expectPrepare(mock)
	require.NoError(t, store.prepare(context.Background()))

	return store, mock, func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
	}
}

func expectPrepare(mock sqlmock.Sqlmock) {
	mock.ExpectPrepare(regexp.QuoteMeta(`INSERT INTO workflows (`))
	mock.ExpectPrepare(regexp.QuoteMeta(`SELECT workflow_id, status, current_stage, progress, created_at, COALESCE(started_at, ''), COALESCE(completed_at, ''), COALESCE(estimated_completion, ''),
				artifacts_json, errors_json, COALESCE(self_link, ''), COALESCE(artifacts_link, ''), COALESCE(cancel_link, '')
				FROM workflows
				WHERE tenant_id = $1 AND ($2 = '' OR status = $2)
				ORDER BY created_at DESC
				LIMIT $3 OFFSET $4`))
	mock.ExpectPrepare(regexp.QuoteMeta(`SELECT workflow_id, status, current_stage, progress, created_at, COALESCE(started_at, ''), COALESCE(completed_at, ''), COALESCE(estimated_completion, ''),
				artifacts_json, errors_json, COALESCE(self_link, ''), COALESCE(artifacts_link, ''), COALESCE(cancel_link, '')
				FROM workflows WHERE tenant_id = $1 AND workflow_id = $2`))
	mock.ExpectPrepare(regexp.QuoteMeta(`UPDATE workflows
				SET status = 'failed', current_stage = 'failed', errors_json = '["workflow cancelled by operator"]'::jsonb
				WHERE tenant_id = $1 AND workflow_id = $2`))
	mock.ExpectPrepare(regexp.QuoteMeta(`SELECT artifacts_json FROM workflows WHERE tenant_id = $1 AND workflow_id = $2`))
	mock.ExpectPrepare(regexp.QuoteMeta(`INSERT INTO prds (`))
	mock.ExpectPrepare(regexp.QuoteMeta(`SELECT prd_id, markdown, json_payload, success_metrics_json, user_stories_json, technical_requirements_json, created_at FROM prds WHERE tenant_id = $1 AND prd_id = $2`))
	mock.ExpectPrepare(regexp.QuoteMeta(`INSERT INTO idempotency_records (tenant_id, operation, idem_key, status_code, response_body, response_digest)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (tenant_id, operation, idem_key) DO UPDATE SET
					status_code = EXCLUDED.status_code,
					response_body = EXCLUDED.response_body,
					response_digest = EXCLUDED.response_digest`))
	mock.ExpectPrepare(regexp.QuoteMeta(`SELECT status_code, response_body, response_digest FROM idempotency_records WHERE tenant_id = $1 AND operation = $2 AND idem_key = $3`))
}
