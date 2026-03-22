package ports

import (
	"context"
	"time"

	"github.com/stratogroup/neuroforge-sdx/internal/domain"
)

type WorkflowFilter struct {
	TenantID string
	Page     int
	Size     int
	Status   string
}

type WorkflowRepository interface {
	SaveWorkflow(context.Context, string, domain.WorkflowResponse, map[string]domain.ArtifactDescriptor) error
	ListWorkflows(context.Context, WorkflowFilter) ([]domain.WorkflowResponse, domain.Pagination, error)
	GetWorkflow(context.Context, string, string) (domain.WorkflowResponse, error)
	CancelWorkflow(context.Context, string, string) error
	GetWorkflowArtifacts(context.Context, string, string) (domain.WorkflowArtifactsResponse, error)
}

type PRDRepository interface {
	SavePRD(context.Context, string, domain.PRDResponse) error
	GetPRD(context.Context, string, string) (domain.PRDResponse, error)
}

type IdempotencyRepository interface {
	SaveIdempotencyRecord(context.Context, domain.IdempotencyRecord) error
	GetIdempotencyRecord(context.Context, string, string, string) (domain.IdempotencyRecord, bool, error)
}

type EventPublisher interface {
	Publish(context.Context, string, []byte) error
	Close() error
}

type ArtifactService interface {
	WorkflowArtifacts(context.Context, string, string, time.Time) (map[string]domain.ArtifactDescriptor, error)
	DesignArtifacts(context.Context, string, string, time.Time) (domain.DesignArtifacts, error)
	SecurityEvidence(context.Context, string, string, time.Time) (domain.SecurityEvidence, error)
}
