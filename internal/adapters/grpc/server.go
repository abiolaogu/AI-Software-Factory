package grpc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	sdxv1 "github.com/stratogroup/neuroforge-sdx/api/proto/neuroforge/sdx/v1"
	cfgadapter "github.com/stratogroup/neuroforge-sdx/internal/adapters/config"
	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stratogroup/neuroforge-sdx/internal/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	gogrpc "google.golang.org/grpc"
	grpcCodes "google.golang.org/grpc/codes"
	grpchealth "google.golang.org/grpc/health"
	grpcHealthV1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type Server struct {
	sdxv1.UnimplementedSDXServiceServer
	application ports.Application
}

func NewServer(cfg cfgadapter.Config, application ports.Application, logger *slog.Logger) *gogrpc.Server {
	if logger == nil {
		logger = slog.Default()
	}

	server := gogrpc.NewServer(
		gogrpc.ChainUnaryInterceptor(
			observabilityInterceptor(logger, otel.Tracer(cfg.ServiceName+".grpc")),
			requestScopeInterceptor(cfg),
		),
	)

	sdxv1.RegisterSDXServiceServer(server, &Server{application: application})
	registerHealthAndReflection(server)

	return server
}

func registerHealthAndReflection(server *gogrpc.Server) {
	healthServer := grpchealth.NewServer()
	healthServer.SetServingStatus("", grpcHealthV1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(sdxv1.SDXService_ServiceDesc.ServiceName, grpcHealthV1.HealthCheckResponse_SERVING)
	grpcHealthV1.RegisterHealthServer(server, healthServer)
	reflection.Register(server)
}

func observabilityInterceptor(logger *slog.Logger, tracer trace.Tracer) gogrpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *gogrpc.UnaryServerInfo, handler gogrpc.UnaryHandler) (any, error) {
		started := time.Now()
		tenantID := metadataValue(ctx, domain.HeaderTenantID)

		ctx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", sdxv1.SDXService_ServiceDesc.ServiceName),
				attribute.String("rpc.method", shortMethodName(info.FullMethod)),
			),
		)
		if tenantID != "" {
			span.SetAttributes(attribute.String("tenant.id", tenantID))
		}

		response, err := handler(ctx, req)
		grpcCode := status.Code(err)
		recordSpanResult(span, grpcCode, err)

		logger.Info("grpc_request",
			"method", info.FullMethod,
			"status", grpcCode.String(),
			"latency_ms", time.Since(started).Milliseconds(),
			"tenant_id", tenantID,
			"error", errorString(err),
		)

		return response, err
	}
}

func requestScopeInterceptor(cfg cfgadapter.Config) gogrpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *gogrpc.UnaryServerInfo, handler gogrpc.UnaryHandler) (any, error) {
		if !requiresTenantScope(info.FullMethod) {
			return handler(ctx, req)
		}

		authorization := metadataValue(ctx, "authorization")
		if !cfg.AllowAnonymousAuth && authorization == "" {
			return nil, status.Error(grpcCodes.Unauthenticated, "missing Authorization header")
		}

		tenantID := metadataValue(ctx, domain.HeaderTenantID)
		if tenantID == "" {
			return nil, status.Error(grpcCodes.InvalidArgument, "missing tenant scope")
		}

		if tokenTenantID, ok := tenantClaimFromAuthorization(authorization); ok && tokenTenantID != tenantID {
			return nil, status.Error(grpcCodes.Unauthenticated, "tenant claim does not match request scope")
		}

		ctx = domain.WithTenantID(ctx, tenantID)
		if key := metadataValue(ctx, domain.HeaderIdempotencyKey); key != "" {
			ctx = domain.WithIdempotencyKey(ctx, key)
		}

		return handler(ctx, req)
	}
}

func requiresTenantScope(fullMethod string) bool {
	switch fullMethod {
	case sdxv1.SDXService_Login_FullMethodName,
		sdxv1.SDXService_RefreshToken_FullMethodName,
		sdxv1.SDXService_HealthLive_FullMethodName,
		sdxv1.SDXService_HealthReady_FullMethodName:
		return false
	default:
		return true
	}
}

func recordSpanResult(span trace.Span, grpcCode grpcCodes.Code, err error) {
	span.SetAttributes(attribute.String("rpc.grpc.status_code", grpcCode.String()))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, grpcCode.String())
		span.End()
		return
	}

	span.SetStatus(otelcodes.Ok, "OK")
	span.End()
}

func shortMethodName(fullMethod string) string {
	value := strings.TrimSpace(fullMethod)
	if value == "" {
		return "unknown"
	}

	parts := strings.Split(value, "/")
	return parts[len(parts)-1]
}

func errorString(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}

func metadataValue(ctx context.Context, key string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	values := md.Get(strings.ToLower(strings.TrimSpace(key)))
	if len(values) == 0 {
		return ""
	}

	return strings.TrimSpace(values[0])
}

func tenantClaimFromAuthorization(header string) (string, bool) {
	token := bearerToken(header)
	if token == "" {
		return "", false
	}

	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "", false
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}

	claims := map[string]any{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", false
	}

	for _, key := range []string{"tenant_id", "tenant", "org_id"} {
		claim, ok := claims[key].(string)
		if ok && strings.TrimSpace(claim) != "" {
			return strings.TrimSpace(claim), true
		}
	}

	return "", false
}

func bearerToken(header string) string {
	value := strings.TrimSpace(header)
	if value == "" {
		return ""
	}

	if !strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return ""
	}

	return strings.TrimSpace(value[len("Bearer "):])
}

func toStatusError(err error) error {
	if err == nil {
		return nil
	}

	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		return status.Error(grpcStatusCode(appErr.Code), appErr.Error())
	}

	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		return status.Error(grpcCodes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrValidation), errors.Is(err, domain.ErrTenantRequired):
		return status.Error(grpcCodes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(grpcCodes.NotFound, err.Error())
	case errors.Is(err, domain.ErrConflict):
		return status.Error(grpcCodes.AlreadyExists, err.Error())
	default:
		return status.Error(grpcCodes.Internal, err.Error())
	}
}

func grpcStatusCode(code domain.ErrorCode) grpcCodes.Code {
	switch code {
	case domain.ErrorCodeUnauthorized:
		return grpcCodes.Unauthenticated
	case domain.ErrorCodeNotFound:
		return grpcCodes.NotFound
	case domain.ErrorCodeConflict:
		return grpcCodes.AlreadyExists
	case domain.ErrorCodeTenantRequired, domain.ErrorCodeValidation:
		return grpcCodes.InvalidArgument
	default:
		return grpcCodes.Internal
	}
}
