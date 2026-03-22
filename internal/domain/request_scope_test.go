package domain_test

import (
	"context"
	"testing"

	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestTenantAndIdempotencyContextHelpers(t *testing.T) {
	ctx := domain.WithTenantID(context.Background(), "tenant-a")
	ctx = domain.WithIdempotencyKey(ctx, "idem-1")

	tenantID, err := domain.TenantIDFromContext(ctx, "test.operation")
	require.NoError(t, err)
	require.Equal(t, "tenant-a", tenantID)
	require.Equal(t, "idem-1", domain.IdempotencyKeyFromContext(ctx))
}

func TestTenantContextMissingReturnsTypedError(t *testing.T) {
	_, err := domain.TenantIDFromContext(context.Background(), "test.operation")
	require.Error(t, err)

	appErr := new(domain.AppError)
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, domain.ErrorCodeTenantRequired, appErr.Code)
}

func TestAppErrorFormattingAndWrapping(t *testing.T) {
	err := domain.NewValidationError("test.operation", "invalid request", domain.ErrorDetail{Field: "prompt", Error: "too short"})
	require.Contains(t, err.Error(), "test.operation")
	require.ErrorIs(t, err, domain.ErrValidation)
}
