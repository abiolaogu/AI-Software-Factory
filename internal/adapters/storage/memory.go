package storage

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stratogroup/neuroforge-sdx/internal/ports"
)

type MemoryStore struct {
	mu          sync.RWMutex
	workflows   map[string]map[string]domain.WorkflowResponse
	artifacts   map[string]map[string]map[string]domain.ArtifactDescriptor
	prds        map[string]map[string]domain.PRDResponse
	idempotency map[string]domain.IdempotencyRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		workflows:   make(map[string]map[string]domain.WorkflowResponse),
		artifacts:   make(map[string]map[string]map[string]domain.ArtifactDescriptor),
		prds:        make(map[string]map[string]domain.PRDResponse),
		idempotency: make(map[string]domain.IdempotencyRecord),
	}
}

func (m *MemoryStore) SaveWorkflow(_ context.Context, tenantID string, workflow domain.WorkflowResponse, artifacts map[string]domain.ArtifactDescriptor) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.workflows[tenantID]; !ok {
		m.workflows[tenantID] = make(map[string]domain.WorkflowResponse)
	}
	if _, ok := m.artifacts[tenantID]; !ok {
		m.artifacts[tenantID] = make(map[string]map[string]domain.ArtifactDescriptor)
	}

	m.workflows[tenantID][workflow.WorkflowID] = workflow
	m.artifacts[tenantID][workflow.WorkflowID] = artifacts

	return nil
}

func (m *MemoryStore) ListWorkflows(_ context.Context, filter ports.WorkflowFilter) ([]domain.WorkflowResponse, domain.Pagination, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenantData := m.workflows[filter.TenantID]
	workflows := make([]domain.WorkflowResponse, 0, len(tenantData))
	for _, workflow := range tenantData {
		if filter.Status != "" && workflow.Status != filter.Status {
			continue
		}
		workflows = append(workflows, workflow)
	}

	sort.Slice(workflows, func(i, j int) bool {
		return workflows[i].CreatedAt > workflows[j].CreatedAt
	})

	page := filter.Page
	if page < 1 {
		page = 1
	}
	size := filter.Size
	if size < 1 {
		size = 20
	}

	total := len(workflows)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}

	pages := 0
	if total > 0 {
		pages = int(math.Ceil(float64(total) / float64(size)))
	}

	return workflows[start:end], domain.Pagination{
		Page:  page,
		Size:  size,
		Total: total,
		Pages: pages,
	}, nil
}

func (m *MemoryStore) GetWorkflow(_ context.Context, tenantID, workflowID string) (domain.WorkflowResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	workflow, ok := m.workflows[tenantID][workflowID]
	if !ok {
		return domain.WorkflowResponse{}, domain.NewNotFoundError("storage.memory.get_workflow", "workflow was not found in tenant scope")
	}

	return workflow, nil
}

func (m *MemoryStore) CancelWorkflow(_ context.Context, tenantID, workflowID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	workflow, ok := m.workflows[tenantID][workflowID]
	if !ok {
		return domain.NewNotFoundError("storage.memory.cancel_workflow", "workflow was not found in tenant scope")
	}

	workflow.Status = "failed"
	workflow.CurrentStage = domain.WorkflowStageFailed
	workflow.Errors = append(workflow.Errors, "workflow cancelled by operator")
	m.workflows[tenantID][workflowID] = workflow

	return nil
}

func (m *MemoryStore) GetWorkflowArtifacts(_ context.Context, tenantID, workflowID string) (domain.WorkflowArtifactsResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	artifacts, ok := m.artifacts[tenantID][workflowID]
	if !ok {
		return domain.WorkflowArtifactsResponse{}, domain.NewNotFoundError("storage.memory.get_artifacts", "artifacts were not found in tenant scope")
	}

	return domain.WorkflowArtifactsResponse{Artifacts: artifacts}, nil
}

func (m *MemoryStore) SavePRD(_ context.Context, tenantID string, prd domain.PRDResponse) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.prds[tenantID]; !ok {
		m.prds[tenantID] = make(map[string]domain.PRDResponse)
	}
	m.prds[tenantID][prd.PRDID] = prd

	return nil
}

func (m *MemoryStore) GetPRD(_ context.Context, tenantID, prdID string) (domain.PRDResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	prd, ok := m.prds[tenantID][prdID]
	if !ok {
		return domain.PRDResponse{}, domain.NewNotFoundError("storage.memory.get_prd", "prd was not found in tenant scope")
	}

	return prd, nil
}

func (m *MemoryStore) SaveIdempotencyRecord(_ context.Context, record domain.IdempotencyRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.idempotency[idempotencyKey(record.TenantID, record.Operation, record.Key)] = record
	return nil
}

func (m *MemoryStore) GetIdempotencyRecord(_ context.Context, tenantID, operation, key string) (domain.IdempotencyRecord, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	record, ok := m.idempotency[idempotencyKey(tenantID, operation, key)]
	return record, ok, nil
}

func idempotencyKey(tenantID, operation, key string) string {
	return fmt.Sprintf("%s:%s:%s", tenantID, operation, key)
}
