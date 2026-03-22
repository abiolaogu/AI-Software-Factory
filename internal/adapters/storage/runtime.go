package storage

import (
	"context"
	"io"

	"github.com/stratogroup/neuroforge-sdx/internal/ports"
)

var newPostgresStoreFn = NewPostgresStore

type RuntimeStore struct {
	WorkflowRepository    ports.WorkflowRepository
	PRDRepository         ports.PRDRepository
	IdempotencyRepository ports.IdempotencyRepository
	closer                io.Closer
}

func NewRuntimeStore(ctx context.Context, dsn string) (*RuntimeStore, error) {
	if dsn == "" {
		store := NewMemoryStore()
		return &RuntimeStore{
			WorkflowRepository:    store,
			PRDRepository:         store,
			IdempotencyRepository: store,
		}, nil
	}

	store, err := newPostgresStoreFn(ctx, dsn)
	if err != nil {
		return nil, err
	}

	return &RuntimeStore{
		WorkflowRepository:    store,
		PRDRepository:         store,
		IdempotencyRepository: store,
		closer:                store,
	}, nil
}

func (r *RuntimeStore) Close() error {
	if r == nil || r.closer == nil {
		return nil
	}
	return r.closer.Close()
}
