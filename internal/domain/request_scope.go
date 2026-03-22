package domain

import (
	"context"
	"strings"
)

const (
	HeaderTenantID       = "X-Tenant-ID"
	HeaderIdempotencyKey = "Idempotency-Key"
)

type contextKey string

const (
	tenantIDContextKey       contextKey = "tenant_id"
	idempotencyKeyContextKey contextKey = "idempotency_key"
)

type IdempotencyRecord struct {
	TenantID       string
	Operation      string
	Key            string
	StatusCode     int
	ResponseBody   []byte
	ResponseDigest string
}

func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDContextKey, strings.TrimSpace(tenantID))
}

func TenantIDFromContext(ctx context.Context, op string) (string, error) {
	tenantID, _ := ctx.Value(tenantIDContextKey).(string)
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return "", NewTenantRequiredError(op)
	}

	return tenantID, nil
}

func WithIdempotencyKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, idempotencyKeyContextKey, strings.TrimSpace(key))
}

func IdempotencyKeyFromContext(ctx context.Context) string {
	key, _ := ctx.Value(idempotencyKeyContextKey).(string)
	return strings.TrimSpace(key)
}
