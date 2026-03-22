package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestStatusFromCodeMappings(t *testing.T) {
	require.Equal(t, http.StatusUnauthorized, statusFromCode(domain.ErrorCodeUnauthorized))
	require.Equal(t, http.StatusNotFound, statusFromCode(domain.ErrorCodeNotFound))
	require.Equal(t, http.StatusConflict, statusFromCode(domain.ErrorCodeConflict))
	require.Equal(t, http.StatusBadRequest, statusFromCode(domain.ErrorCodeValidation))
	require.Equal(t, http.StatusInternalServerError, statusFromCode(domain.ErrorCodeInternal))
}

func TestQueryIntFallbacks(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/?page=oops&size=7", nil)

	require.Equal(t, 3, queryInt(c, "page", 3))
	require.Equal(t, 7, queryInt(c, "size", 20))
	require.Equal(t, 9, queryInt(c, "missing", 9))
}

func TestWriteErrorUsesTypedAndSentinelMappings(t *testing.T) {
	t.Run("app error", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		writeError(c, domain.NewConflictError("workflow.create", "duplicate"))
		require.Equal(t, http.StatusConflict, recorder.Code)
		require.Contains(t, recorder.Body.String(), "CONFLICT")
	})

	t.Run("sentinel error", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		writeError(c, domain.ErrUnauthorized)
		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		require.Contains(t, recorder.Body.String(), "UNAUTHORIZED")
	})

	t.Run("generic error", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		writeError(c, errors.New("boom"))
		require.Equal(t, http.StatusInternalServerError, recorder.Code)
		require.Contains(t, recorder.Body.String(), "boom")
	})
}

func TestAuthMiddlewareRejectsMissingAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(authMiddleware())
	engine.GET("/secure", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/secure", nil)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Contains(t, recorder.Body.String(), "missing Authorization header")
}
