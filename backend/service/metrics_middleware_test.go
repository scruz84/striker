package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"

	"github.com/scruz84/striker/common/metrictest"
)

func metricsTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(metricsMiddleware())
	router.GET("/v1/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/v1/live", func(c *gin.Context) { c.Status(http.StatusSwitchingProtocols) })
	return router
}

// TestMetricsMiddleware_WebsocketUpgradeIsCountedButNotTimed: a live match websocket adds to the
// request counter but records no duration, while a plain GET records both.
func TestMetricsMiddleware_WebsocketUpgradeIsCountedButNotTimed(t *testing.T) {
	metrictest.Install()
	router := metricsTestRouter()
	live := attribute.String("route", "/v1/live")
	ok := attribute.String("route", "/v1/ok")

	// Baselines, not absolutes: the meter provider is process-wide, so a repeated run (go test
	// -count=N) starts from what the earlier ones recorded.
	countedBefore := metrictest.Sum(t, "striker.http.requests", live)
	timedBefore := metrictest.HistogramCount(t, "striker.http.request.duration", live)

	req := httptest.NewRequest(http.MethodGet, "/v1/live", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	router.ServeHTTP(httptest.NewRecorder(), req)

	if got := metrictest.Sum(t, "striker.http.requests", live) - countedBefore; got != 1 {
		t.Errorf("websocket request count grew by %v, want 1", got)
	}
	if got := metrictest.HistogramCount(t, "striker.http.request.duration", live) - timedBefore; got != 0 {
		t.Errorf("websocket upgrade recorded %d durations, want 0", got)
	}

	countedBefore = metrictest.Sum(t, "striker.http.requests", ok)
	timedBefore = metrictest.HistogramCount(t, "striker.http.request.duration", ok)
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/ok", nil))
	if got := metrictest.Sum(t, "striker.http.requests", ok) - countedBefore; got != 1 {
		t.Errorf("plain request count grew by %v, want 1", got)
	}
	if got := metrictest.HistogramCount(t, "striker.http.request.duration", ok) - timedBefore; got != 1 {
		t.Errorf("plain request recorded %d durations, want 1", got)
	}
}
