package domain_test

import (
	"errors"
	"testing"

	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestErrorConstructorsReturnTypedErrors(t *testing.T) {
	unauthorized := domain.NewUnauthorizedError("auth.login", "invalid credentials")
	require.Equal(t, domain.ErrorCodeUnauthorized, unauthorized.Code)
	require.ErrorIs(t, unauthorized, domain.ErrUnauthorized)

	notFound := domain.NewNotFoundError("workflow.get", "missing workflow")
	require.Equal(t, domain.ErrorCodeNotFound, notFound.Code)
	require.ErrorIs(t, notFound, domain.ErrNotFound)

	conflict := domain.NewConflictError("workflow.create", "duplicate request")
	require.Equal(t, domain.ErrorCodeConflict, conflict.Code)
	require.ErrorIs(t, conflict, domain.ErrConflict)
}

func TestWrapErrorPreservesCauseAndCode(t *testing.T) {
	cause := errors.New("database unavailable")
	err := domain.WrapError("storage.save", cause, domain.ErrorCodeInternal, "write failed")

	require.Equal(t, domain.ErrorCodeInternal, err.Code)
	require.ErrorIs(t, err, cause)
	require.Contains(t, err.Error(), "storage.save")
	require.Contains(t, err.Error(), "write failed")
}
