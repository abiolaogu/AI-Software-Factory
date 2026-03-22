package storage

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stratogroup/neuroforge-sdx/internal/ports"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

var openDB = sql.Open

type PostgresStore struct {
	db *sql.DB

	saveWorkflowStmt         *sql.Stmt
	listWorkflowsStmt        *sql.Stmt
	getWorkflowStmt          *sql.Stmt
	cancelWorkflowStmt       *sql.Stmt
	getWorkflowArtifactsStmt *sql.Stmt
	savePRDStmt              *sql.Stmt
	getPRDStmt               *sql.Stmt
	saveIdempotencyStmt      *sql.Stmt
	getIdempotencyStmt       *sql.Stmt
}

func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	db, err := openDB("pgx", dsn)
	if err != nil {
		return nil, domain.WrapError("storage.postgres.open", err, domain.ErrorCodeInternal, "opening postgres connection failed")
	}

	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(20)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, domain.WrapError("storage.postgres.ping", err, domain.ErrorCodeInternal, "pinging postgres failed")
	}

	store := &PostgresStore{db: db}
	if err := store.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.prepare(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *PostgresStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *PostgresStore) SaveWorkflow(ctx context.Context, tenantID string, workflow domain.WorkflowResponse, artifacts map[string]domain.ArtifactDescriptor) error {
	artifactsJSON, err := json.Marshal(artifacts)
	if err != nil {
		return domain.WrapError("storage.postgres.save_workflow", err, domain.ErrorCodeInternal, "marshalling workflow artifacts failed")
	}
	errorsJSON, err := json.Marshal(workflow.Errors)
	if err != nil {
		return domain.WrapError("storage.postgres.save_workflow", err, domain.ErrorCodeInternal, "marshalling workflow errors failed")
	}

	_, err = s.saveWorkflowStmt.ExecContext(
		ctx,
		tenantID,
		workflow.WorkflowID,
		workflow.Status,
		string(workflow.CurrentStage),
		workflow.Progress,
		workflow.CreatedAt,
		workflow.StartedAt,
		workflow.CompletedAt,
		workflow.EstimatedCompletion,
		artifactsJSON,
		errorsJSON,
		workflow.Links.Self,
		workflow.Links.Artifacts,
		workflow.Links.Cancel,
	)
	if err != nil {
		return domain.WrapError("storage.postgres.save_workflow", err, domain.ErrorCodeInternal, "executing workflow upsert failed")
	}

	return nil
}

func (s *PostgresStore) ListWorkflows(ctx context.Context, filter ports.WorkflowFilter) ([]domain.WorkflowResponse, domain.Pagination, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	size := filter.Size
	if size < 1 {
		size = 20
	}
	offset := (page - 1) * size

	rows, err := s.listWorkflowsStmt.QueryContext(ctx, filter.TenantID, filter.Status, size, offset)
	if err != nil {
		return nil, domain.Pagination{}, domain.WrapError("storage.postgres.list_workflows", err, domain.ErrorCodeInternal, "querying workflows failed")
	}
	defer func() {
		_ = rows.Close()
	}()

	workflows := make([]domain.WorkflowResponse, 0)
	for rows.Next() {
		workflow, err := scanWorkflow(rows)
		if err != nil {
			return nil, domain.Pagination{}, err
		}
		workflows = append(workflows, workflow)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.Pagination{}, domain.WrapError("storage.postgres.list_workflows", err, domain.ErrorCodeInternal, "iterating workflows failed")
	}

	total := len(workflows)
	return workflows, domain.Pagination{
		Page:  page,
		Size:  size,
		Total: total,
		Pages: totalPages(total, size),
	}, nil
}

func (s *PostgresStore) GetWorkflow(ctx context.Context, tenantID, workflowID string) (domain.WorkflowResponse, error) {
	row := s.getWorkflowStmt.QueryRowContext(ctx, tenantID, workflowID)
	workflow, err := scanWorkflow(row)
	if err != nil {
		return domain.WorkflowResponse{}, err
	}
	return workflow, nil
}

func (s *PostgresStore) CancelWorkflow(ctx context.Context, tenantID, workflowID string) error {
	result, err := s.cancelWorkflowStmt.ExecContext(ctx, tenantID, workflowID)
	if err != nil {
		return domain.WrapError("storage.postgres.cancel_workflow", err, domain.ErrorCodeInternal, "updating workflow status failed")
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domain.WrapError("storage.postgres.cancel_workflow", err, domain.ErrorCodeInternal, "reading affected row count failed")
	}
	if rowsAffected == 0 {
		return domain.NewNotFoundError("storage.postgres.cancel_workflow", "workflow was not found in tenant scope")
	}
	return nil
}

func (s *PostgresStore) GetWorkflowArtifacts(ctx context.Context, tenantID, workflowID string) (domain.WorkflowArtifactsResponse, error) {
	var payload []byte
	if err := s.getWorkflowArtifactsStmt.QueryRowContext(ctx, tenantID, workflowID).Scan(&payload); err != nil {
		if err == sql.ErrNoRows {
			return domain.WorkflowArtifactsResponse{}, domain.NewNotFoundError("storage.postgres.get_workflow_artifacts", "artifacts were not found in tenant scope")
		}
		return domain.WorkflowArtifactsResponse{}, domain.WrapError("storage.postgres.get_workflow_artifacts", err, domain.ErrorCodeInternal, "querying workflow artifacts failed")
	}

	artifacts := make(map[string]domain.ArtifactDescriptor)
	if err := json.Unmarshal(payload, &artifacts); err != nil {
		return domain.WorkflowArtifactsResponse{}, domain.WrapError("storage.postgres.get_workflow_artifacts", err, domain.ErrorCodeInternal, "decoding workflow artifacts failed")
	}

	return domain.WorkflowArtifactsResponse{Artifacts: artifacts}, nil
}

func (s *PostgresStore) SavePRD(ctx context.Context, tenantID string, prd domain.PRDResponse) error {
	jsonPayload, err := json.Marshal(prd.JSON)
	if err != nil {
		return domain.WrapError("storage.postgres.save_prd", err, domain.ErrorCodeInternal, "marshalling prd json failed")
	}
	successMetrics, err := json.Marshal(prd.SuccessMetrics)
	if err != nil {
		return domain.WrapError("storage.postgres.save_prd", err, domain.ErrorCodeInternal, "marshalling success metrics failed")
	}
	userStories, err := json.Marshal(prd.UserStories)
	if err != nil {
		return domain.WrapError("storage.postgres.save_prd", err, domain.ErrorCodeInternal, "marshalling user stories failed")
	}
	technicalRequirements, err := json.Marshal(prd.TechnicalRequirements)
	if err != nil {
		return domain.WrapError("storage.postgres.save_prd", err, domain.ErrorCodeInternal, "marshalling technical requirements failed")
	}

	_, err = s.savePRDStmt.ExecContext(
		ctx,
		tenantID,
		prd.PRDID,
		prd.Markdown,
		jsonPayload,
		successMetrics,
		userStories,
		technicalRequirements,
		prd.CreatedAt,
	)
	if err != nil {
		return domain.WrapError("storage.postgres.save_prd", err, domain.ErrorCodeInternal, "executing prd upsert failed")
	}

	return nil
}

func (s *PostgresStore) GetPRD(ctx context.Context, tenantID, prdID string) (domain.PRDResponse, error) {
	row := s.getPRDStmt.QueryRowContext(ctx, tenantID, prdID)

	var (
		prd                   domain.PRDResponse
		jsonPayload           []byte
		successMetricsPayload []byte
		userStoriesPayload    []byte
		requirementsPayload   []byte
	)

	if err := row.Scan(
		&prd.PRDID,
		&prd.Markdown,
		&jsonPayload,
		&successMetricsPayload,
		&userStoriesPayload,
		&requirementsPayload,
		&prd.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return domain.PRDResponse{}, domain.NewNotFoundError("storage.postgres.get_prd", "prd was not found in tenant scope")
		}
		return domain.PRDResponse{}, domain.WrapError("storage.postgres.get_prd", err, domain.ErrorCodeInternal, "querying prd failed")
	}

	if err := json.Unmarshal(jsonPayload, &prd.JSON); err != nil {
		return domain.PRDResponse{}, domain.WrapError("storage.postgres.get_prd", err, domain.ErrorCodeInternal, "decoding prd json failed")
	}
	if err := json.Unmarshal(successMetricsPayload, &prd.SuccessMetrics); err != nil {
		return domain.PRDResponse{}, domain.WrapError("storage.postgres.get_prd", err, domain.ErrorCodeInternal, "decoding prd metrics failed")
	}
	if err := json.Unmarshal(userStoriesPayload, &prd.UserStories); err != nil {
		return domain.PRDResponse{}, domain.WrapError("storage.postgres.get_prd", err, domain.ErrorCodeInternal, "decoding prd stories failed")
	}
	if err := json.Unmarshal(requirementsPayload, &prd.TechnicalRequirements); err != nil {
		return domain.PRDResponse{}, domain.WrapError("storage.postgres.get_prd", err, domain.ErrorCodeInternal, "decoding prd requirements failed")
	}

	return prd, nil
}

func (s *PostgresStore) SaveIdempotencyRecord(ctx context.Context, record domain.IdempotencyRecord) error {
	_, err := s.saveIdempotencyStmt.ExecContext(
		ctx,
		record.TenantID,
		record.Operation,
		record.Key,
		record.StatusCode,
		record.ResponseBody,
		record.ResponseDigest,
	)
	if err != nil {
		return domain.WrapError("storage.postgres.save_idempotency", err, domain.ErrorCodeInternal, "saving idempotency record failed")
	}
	return nil
}

func (s *PostgresStore) GetIdempotencyRecord(ctx context.Context, tenantID, operation, key string) (domain.IdempotencyRecord, bool, error) {
	record := domain.IdempotencyRecord{}
	if err := s.getIdempotencyStmt.QueryRowContext(ctx, tenantID, operation, key).Scan(
		&record.StatusCode,
		&record.ResponseBody,
		&record.ResponseDigest,
	); err != nil {
		if err == sql.ErrNoRows {
			return domain.IdempotencyRecord{}, false, nil
		}
		return domain.IdempotencyRecord{}, false, domain.WrapError("storage.postgres.get_idempotency", err, domain.ErrorCodeInternal, "loading idempotency record failed")
	}
	record.TenantID = tenantID
	record.Operation = operation
	record.Key = key
	return record, true, nil
}

func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *PostgresStore) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`); err != nil {
		return domain.WrapError("storage.postgres.migrate", err, domain.ErrorCodeInternal, "creating migration table failed")
	}

	entries, err := migrationEntries()
	if err != nil {
		return err
	}

	for _, entry := range entries {
		version := filepath.Base(entry.Name())
		applied, err := s.migrationApplied(ctx, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if err := s.applyMigration(ctx, entry.Name(), version); err != nil {
			return err
		}
	}

	return nil
}

func migrationEntries() ([]os.DirEntry, error) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return nil, domain.WrapError("storage.postgres.migrate", err, domain.ErrorCodeInternal, "reading migrations failed")
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	return entries, nil
}

func (s *PostgresStore) applyMigration(ctx context.Context, entryName, version string) error {
	body, err := migrationFiles.ReadFile("migrations/" + entryName)
	if err != nil {
		return domain.WrapError("storage.postgres.migrate", err, domain.ErrorCodeInternal, "reading migration body failed")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.WrapError("storage.postgres.migrate", err, domain.ErrorCodeInternal, "starting migration transaction failed")
	}

	if _, err := tx.ExecContext(ctx, string(body)); err != nil {
		_ = tx.Rollback()
		return domain.WrapError("storage.postgres.migrate", err, domain.ErrorCodeInternal, fmt.Sprintf("executing migration %s failed", version))
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
		_ = tx.Rollback()
		return domain.WrapError("storage.postgres.migrate", err, domain.ErrorCodeInternal, fmt.Sprintf("recording migration %s failed", version))
	}
	if err := tx.Commit(); err != nil {
		return domain.WrapError("storage.postgres.migrate", err, domain.ErrorCodeInternal, fmt.Sprintf("committing migration %s failed", version))
	}

	return nil
}

func (s *PostgresStore) migrationApplied(ctx context.Context, version string) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE version = $1`, version).Scan(&count); err != nil {
		return false, domain.WrapError("storage.postgres.migration_applied", err, domain.ErrorCodeInternal, "querying applied migrations failed")
	}
	return count > 0, nil
}

func (s *PostgresStore) prepare(ctx context.Context) error {
	statements := []struct {
		assign **sql.Stmt
		query  string
	}{
		{
			assign: &s.saveWorkflowStmt,
			query: `INSERT INTO workflows (
				tenant_id, workflow_id, status, current_stage, progress, created_at, started_at, completed_at, estimated_completion,
				artifacts_json, errors_json, self_link, artifacts_link, cancel_link
			) VALUES (
				$1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), $10, $11, NULLIF($12, ''), NULLIF($13, ''), NULLIF($14, '')
			)
			ON CONFLICT (tenant_id, workflow_id) DO UPDATE SET
				status = EXCLUDED.status,
				current_stage = EXCLUDED.current_stage,
				progress = EXCLUDED.progress,
				created_at = EXCLUDED.created_at,
				started_at = EXCLUDED.started_at,
				completed_at = EXCLUDED.completed_at,
				estimated_completion = EXCLUDED.estimated_completion,
				artifacts_json = EXCLUDED.artifacts_json,
				errors_json = EXCLUDED.errors_json,
				self_link = EXCLUDED.self_link,
				artifacts_link = EXCLUDED.artifacts_link,
				cancel_link = EXCLUDED.cancel_link`,
		},
		{
			assign: &s.listWorkflowsStmt,
			query: `SELECT workflow_id, status, current_stage, progress, created_at, COALESCE(started_at, ''), COALESCE(completed_at, ''), COALESCE(estimated_completion, ''),
				artifacts_json, errors_json, COALESCE(self_link, ''), COALESCE(artifacts_link, ''), COALESCE(cancel_link, '')
				FROM workflows
				WHERE tenant_id = $1 AND ($2 = '' OR status = $2)
				ORDER BY created_at DESC
				LIMIT $3 OFFSET $4`,
		},
		{
			assign: &s.getWorkflowStmt,
			query: `SELECT workflow_id, status, current_stage, progress, created_at, COALESCE(started_at, ''), COALESCE(completed_at, ''), COALESCE(estimated_completion, ''),
				artifacts_json, errors_json, COALESCE(self_link, ''), COALESCE(artifacts_link, ''), COALESCE(cancel_link, '')
				FROM workflows WHERE tenant_id = $1 AND workflow_id = $2`,
		},
		{
			assign: &s.cancelWorkflowStmt,
			query: `UPDATE workflows
				SET status = 'failed', current_stage = 'failed', errors_json = '["workflow cancelled by operator"]'::jsonb
				WHERE tenant_id = $1 AND workflow_id = $2`,
		},
		{
			assign: &s.getWorkflowArtifactsStmt,
			query:  `SELECT artifacts_json FROM workflows WHERE tenant_id = $1 AND workflow_id = $2`,
		},
		{
			assign: &s.savePRDStmt,
			query: `INSERT INTO prds (
				tenant_id, prd_id, markdown, json_payload, success_metrics_json, user_stories_json, technical_requirements_json, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (tenant_id, prd_id) DO UPDATE SET
				markdown = EXCLUDED.markdown,
				json_payload = EXCLUDED.json_payload,
				success_metrics_json = EXCLUDED.success_metrics_json,
				user_stories_json = EXCLUDED.user_stories_json,
				technical_requirements_json = EXCLUDED.technical_requirements_json,
				created_at = EXCLUDED.created_at`,
		},
		{
			assign: &s.getPRDStmt,
			query:  `SELECT prd_id, markdown, json_payload, success_metrics_json, user_stories_json, technical_requirements_json, created_at FROM prds WHERE tenant_id = $1 AND prd_id = $2`,
		},
		{
			assign: &s.saveIdempotencyStmt,
			query: `INSERT INTO idempotency_records (tenant_id, operation, idem_key, status_code, response_body, response_digest)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (tenant_id, operation, idem_key) DO UPDATE SET
					status_code = EXCLUDED.status_code,
					response_body = EXCLUDED.response_body,
					response_digest = EXCLUDED.response_digest`,
		},
		{
			assign: &s.getIdempotencyStmt,
			query:  `SELECT status_code, response_body, response_digest FROM idempotency_records WHERE tenant_id = $1 AND operation = $2 AND idem_key = $3`,
		},
	}

	for _, statement := range statements {
		stmt, err := s.db.PrepareContext(ctx, statement.query)
		if err != nil {
			return domain.WrapError("storage.postgres.prepare", err, domain.ErrorCodeInternal, "preparing statement failed")
		}
		*statement.assign = stmt
	}

	return nil
}

type workflowScanner interface {
	Scan(dest ...any) error
}

func scanWorkflow(scanner workflowScanner) (domain.WorkflowResponse, error) {
	var (
		workflow      domain.WorkflowResponse
		currentStage  string
		artifactsJSON []byte
		errorsJSON    []byte
	)

	if err := scanner.Scan(
		&workflow.WorkflowID,
		&workflow.Status,
		&currentStage,
		&workflow.Progress,
		&workflow.CreatedAt,
		&workflow.StartedAt,
		&workflow.CompletedAt,
		&workflow.EstimatedCompletion,
		&artifactsJSON,
		&errorsJSON,
		&workflow.Links.Self,
		&workflow.Links.Artifacts,
		&workflow.Links.Cancel,
	); err != nil {
		if err == sql.ErrNoRows {
			return domain.WorkflowResponse{}, domain.NewNotFoundError("storage.postgres.scan_workflow", "workflow was not found in tenant scope")
		}
		return domain.WorkflowResponse{}, domain.WrapError("storage.postgres.scan_workflow", err, domain.ErrorCodeInternal, "scanning workflow row failed")
	}

	workflow.CurrentStage = domain.WorkflowStage(currentStage)

	artifacts := make(map[string]domain.ArtifactDescriptor)
	if err := json.Unmarshal(artifactsJSON, &artifacts); err != nil {
		return domain.WorkflowResponse{}, domain.WrapError("storage.postgres.scan_workflow", err, domain.ErrorCodeInternal, "decoding workflow artifacts failed")
	}
	workflow.Artifacts = make(map[string]string, len(artifacts))
	for key, artifact := range artifacts {
		workflow.Artifacts[key] = artifact.URL
	}

	if err := json.Unmarshal(errorsJSON, &workflow.Errors); err != nil {
		return domain.WorkflowResponse{}, domain.WrapError("storage.postgres.scan_workflow", err, domain.ErrorCodeInternal, "decoding workflow errors failed")
	}

	return workflow, nil
}

func totalPages(total, size int) int {
	if total == 0 || size <= 0 {
		return 0
	}
	pages := total / size
	if total%size != 0 {
		pages++
	}
	return pages
}
