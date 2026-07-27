package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLoggerWritesRequestFields(t *testing.T) {
	t.Parallel()

	var logs strings.Builder
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&logs),
		zapcore.InfoLevel,
	)
	logger := zap.New(core)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID(), Logger(logger))
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set(RequestIDHeader, "request-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	output := logs.String()
	for _, expected := range []string{
		`"msg":"request completed"`,
		`"request_id":"request-123"`,
		`"method":"GET"`,
		`"path":"/health"`,
		`"status":204`,
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected log output to contain %s, got %s", expected, output)
		}
	}
}
