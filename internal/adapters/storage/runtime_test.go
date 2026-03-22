package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type closerStub struct {
	err    error
	called bool
}

func (c *closerStub) Close() error {
	c.called = true
	return c.err
}

func TestNewRuntimeStoreWithoutDSNUsesMemoryStore(t *testing.T) {
	store, err := NewRuntimeStore(context.Background(), "")
	require.NoError(t, err)
	require.NotNil(t, store.WorkflowRepository)
	require.NotNil(t, store.PRDRepository)
	require.NotNil(t, store.IdempotencyRepository)
	require.NoError(t, store.Close())
}

func TestNewRuntimeStoreWithDSNUsesPostgresFactory(t *testing.T) {
	original := newPostgresStoreFn
	t.Cleanup(func() {
		newPostgresStoreFn = original
	})

	newPostgresStoreFn = func(context.Context, string) (*PostgresStore, error) {
		return &PostgresStore{}, nil
	}

	store, err := NewRuntimeStore(context.Background(), "postgres://tenant@localhost:5433/yugabyte?sslmode=disable")
	require.NoError(t, err)
	require.NotNil(t, store.WorkflowRepository)
	require.NotNil(t, store.PRDRepository)
	require.NotNil(t, store.IdempotencyRepository)
}

func TestNewRuntimeStoreReturnsFactoryErrors(t *testing.T) {
	original := newPostgresStoreFn
	t.Cleanup(func() {
		newPostgresStoreFn = original
	})

	newPostgresStoreFn = func(context.Context, string) (*PostgresStore, error) {
		return nil, errors.New("dial failed")
	}

	_, err := NewRuntimeStore(context.Background(), "postgres://tenant@localhost:5433/yugabyte?sslmode=disable")
	require.EqualError(t, err, "dial failed")
}

func TestRuntimeStoreCloseDelegatesToCloser(t *testing.T) {
	closer := &closerStub{}
	store := &RuntimeStore{closer: closer}

	require.NoError(t, store.Close())
	require.True(t, closer.called)
}
