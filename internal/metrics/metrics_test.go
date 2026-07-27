package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMetricsMiddlewareRecordsRequests(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	m := New()
	router := gin.New()
	router.Use(m.Middleware())
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.GET("/metrics", m.Handler())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	router.ServeHTTP(metricsRec, metricsReq)

	output := metricsRec.Body.String()
	expected := `gateway_requests_total{method="GET",path="/health",status="204"} 1`
	if !strings.Contains(output, expected) {
		t.Fatalf("expected metrics to contain %q, got %s", expected, output)
	}
}
