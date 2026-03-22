package http

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/config"
	"github.com/stratogroup/neuroforge-sdx/internal/adapters/spec"
	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stratogroup/neuroforge-sdx/internal/ports"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func NewRouter(cfg config.Config, application ports.Application, logger *slog.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(requestLogger(logger))
	engine.Use(otelgin.Middleware(cfg.ServiceName))

	engine.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "swagger")
	})
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	engine.GET("/openapi.json", serveSpecJSON)
	engine.GET("/openapi.yaml", serveSpecYAML)
	engine.GET("/swagger", serveSwaggerUI)
	engine.GET("/docs", serveRedoc)
	registerHealthRoutes(engine, application)

	v1 := engine.Group("/v1")
	registerVersionedRoutes(v1, cfg, application)

	return engine
}

func registerVersionedRoutes(group *gin.RouterGroup, cfg config.Config, application ports.Application) {
	group.Use(rateLimitHeadersMiddleware())
	registerHealthRoutes(group, application)
	registerAuthRoutes(group, application)

	protected := protectedGroup(group, cfg)
	registerWorkflowRoutes(protected, application)
	registerPRDRoutes(protected, application)
	registerDesignRoutes(protected, application)
	registerCodeRoutes(protected, application)
	registerSecurityRoutes(protected, application)
	registerPerformanceRoutes(protected, application)
}

func registerAuthRoutes(group *gin.RouterGroup, application ports.Application) {
	group.POST("/auth/login", func(c *gin.Context) {
		var request domain.LoginRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, domain.ErrValidation)
			return
		}
		response, err := application.Login(c.Request.Context(), request)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})
	group.POST("/auth/refresh", func(c *gin.Context) {
		var request domain.RefreshTokenRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, domain.ErrValidation)
			return
		}
		response, err := application.RefreshToken(c.Request.Context(), request)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})
}

func protectedGroup(group *gin.RouterGroup, cfg config.Config) *gin.RouterGroup {
	protected := group.Group("/")
	if !cfg.AllowAnonymousAuth {
		protected.Use(authMiddleware())
	}
	protected.Use(tenantScopeMiddleware())
	return protected
}

func registerWorkflowRoutes(group *gin.RouterGroup, application ports.Application) {
	group.POST("/workflows", func(c *gin.Context) {
		var request domain.WorkflowRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, domain.ErrValidation)
			return
		}
		response, err := application.CreateWorkflow(c.Request.Context(), request)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, response)
	})

	group.GET("/workflows", func(c *gin.Context) {
		response, err := application.ListWorkflows(c.Request.Context(), ports.WorkflowFilter{
			Page:   queryInt(c, "page", 1),
			Size:   queryInt(c, "size", 20),
			Status: c.Query("status"),
		})
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})

	group.GET("/workflows/:workflow_id", func(c *gin.Context) {
		response, err := application.GetWorkflow(c.Request.Context(), c.Param("workflow_id"))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})

	group.DELETE("/workflows/:workflow_id", func(c *gin.Context) {
		if err := application.CancelWorkflow(c.Request.Context(), c.Param("workflow_id")); err != nil {
			writeError(c, err)
			return
		}
		c.Status(http.StatusOK)
	})

	group.GET("/workflows/:workflow_id/artifacts", func(c *gin.Context) {
		response, err := application.GetWorkflowArtifacts(c.Request.Context(), c.Param("workflow_id"))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})
}

func registerPRDRoutes(group *gin.RouterGroup, application ports.Application) {
	group.POST("/prd/generate", func(c *gin.Context) {
		var request domain.PRDRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, domain.ErrValidation)
			return
		}
		response, err := application.GeneratePRD(c.Request.Context(), request)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, response)
	})

	group.GET("/prd/:prd_id", func(c *gin.Context) {
		response, err := application.GetPRD(c.Request.Context(), c.Param("prd_id"))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})
}

func registerDesignRoutes(group *gin.RouterGroup, application ports.Application) {
	group.POST("/design/wireframes", func(c *gin.Context) {
		var request domain.DesignRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, domain.ErrValidation)
			return
		}
		response, err := application.GenerateWireframes(c.Request.Context(), request)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, response)
	})

	group.POST("/design/tokens", func(c *gin.Context) {
		var request domain.ExtractTokensRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, domain.ErrValidation)
			return
		}
		response, err := application.ExtractTokens(c.Request.Context(), request)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})
}

func registerCodeRoutes(group *gin.RouterGroup, application ports.Application) {
	group.POST("/code/generate", func(c *gin.Context) {
		var request domain.CodeRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, domain.ErrValidation)
			return
		}
		response, err := application.GenerateCode(c.Request.Context(), request)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, response)
	})

	group.POST("/code/test", func(c *gin.Context) {
		var request domain.GenerateTestsRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, domain.ErrValidation)
			return
		}
		response, err := application.GenerateTests(c.Request.Context(), request)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})
}

func registerSecurityRoutes(group *gin.RouterGroup, application ports.Application) {
	group.POST("/security/scan", func(c *gin.Context) {
		var request domain.SecurityScanRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, domain.ErrValidation)
			return
		}
		response, err := application.SecurityScan(c.Request.Context(), request)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, response)
	})

	group.GET("/security/findings/:scan_id", func(c *gin.Context) {
		response, err := application.GetSecurityFindings(c.Request.Context(), c.Param("scan_id"))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})
}

func registerPerformanceRoutes(group *gin.RouterGroup, application ports.Application) {
	group.POST("/performance/test", func(c *gin.Context) {
		var request domain.PerformanceTestRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, domain.ErrValidation)
			return
		}
		response, err := application.PerformanceTest(c.Request.Context(), request)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, response)
	})

	group.GET("/performance/results/:test_id", func(c *gin.Context) {
		response, err := application.GetPerformanceResults(c.Request.Context(), c.Param("test_id"))
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})
}

func serveSpecJSON(c *gin.Context) {
	body, err := spec.JSON()
	if err != nil {
		writeError(c, err)
		return
	}

	c.Data(http.StatusOK, "application/json", body)
}

func serveSpecYAML(c *gin.Context) {
	body, err := spec.YAML()
	if err != nil {
		writeError(c, err)
		return
	}

	c.Data(http.StatusOK, "application/yaml", body)
}

func serveSwaggerUI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>NeuroForge SDX Swagger UI</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: %q,
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis]
    });
  </script>
</body>
</html>`, "openapi.json")))
}

func serveRedoc(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>NeuroForge SDX ReDoc</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>body { margin: 0; }</style>
</head>
<body>
  <redoc spec-url="openapi.json"></redoc>
  <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>`))
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.TrimSpace(c.GetHeader("Authorization")) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, domain.ErrorResponse{
				Code:    "UNAUTHORIZED",
				Message: "missing Authorization header",
			})
			return
		}

		c.Next()
	}
}

func tenantScopeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := strings.TrimSpace(c.GetHeader(domain.HeaderTenantID))
		if tenantID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, domain.ErrorResponse{
				Code:    string(domain.ErrorCodeTenantRequired),
				Message: "missing tenant scope",
				Details: []domain.ErrorDetail{
					{Field: domain.HeaderTenantID, Error: "header is required"},
				},
			})
			return
		}

		if tokenTenantID, ok := tenantClaimFromAuthorization(c.GetHeader("Authorization")); ok && tokenTenantID != tenantID {
			c.AbortWithStatusJSON(http.StatusUnauthorized, domain.ErrorResponse{
				Code:    string(domain.ErrorCodeUnauthorized),
				Message: "tenant claim does not match request scope",
				Details: []domain.ErrorDetail{
					{Field: domain.HeaderTenantID, Error: "header does not match token tenant claim"},
				},
			})
			return
		}

		ctx := domain.WithTenantID(c.Request.Context(), tenantID)
		if key := strings.TrimSpace(c.GetHeader(domain.HeaderIdempotencyKey)); key != "" {
			ctx = domain.WithIdempotencyKey(ctx, key)
		}

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func rateLimitHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-RateLimit-Limit", "100")
		c.Header("X-RateLimit-Remaining", "99")
		c.Header("X-RateLimit-Reset", "60")
		c.Next()
	}
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		logger.Info("http_request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(started).Milliseconds(),
			"client_ip", c.ClientIP(),
			"tenant_id", c.GetHeader(domain.HeaderTenantID),
		)
	}
}

func queryInt(c *gin.Context, name string, fallback int) int {
	raw := c.Query(name)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return value
}

func writeError(c *gin.Context, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		c.JSON(statusFromCode(appErr.Code), domain.ErrorResponse{
			Code:    string(appErr.Code),
			Message: appErr.Message,
			Details: appErr.Details,
		})
		return
	}

	switch {
	case errors.Is(err, domain.ErrValidation):
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{
			Code:    "VALIDATION_ERROR",
			Message: "invalid input parameters",
		})
	case errors.Is(err, domain.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{
			Code:    "UNAUTHORIZED",
			Message: "authorization failed",
		})
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, domain.ErrorResponse{
			Code:    "NOT_FOUND",
			Message: "resource not found",
		})
	default:
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		})
	}
}

func registerHealthRoutes(group gin.IRoutes, application ports.Application) {
	group.GET("/health/live", func(c *gin.Context) {
		response, err := application.HealthLive(c.Request.Context())
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})
	group.GET("/health/ready", func(c *gin.Context) {
		response, err := application.HealthReady(c.Request.Context())
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})
	group.GET("/livez", func(c *gin.Context) {
		response, err := application.HealthLive(c.Request.Context())
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})
	group.GET("/readyz", func(c *gin.Context) {
		response, err := application.HealthReady(c.Request.Context())
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})
	group.GET("/healthz", func(c *gin.Context) {
		response, err := application.HealthReady(c.Request.Context())
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})
}

func statusFromCode(code domain.ErrorCode) int {
	switch code {
	case domain.ErrorCodeUnauthorized:
		return http.StatusUnauthorized
	case domain.ErrorCodeNotFound:
		return http.StatusNotFound
	case domain.ErrorCodeConflict:
		return http.StatusConflict
	case domain.ErrorCodeTenantRequired, domain.ErrorCodeValidation:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
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
		value, ok := claims[key]
		if !ok {
			continue
		}
		claim, ok := value.(string)
		if !ok || strings.TrimSpace(claim) == "" {
			return "", false
		}
		return claim, true
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
