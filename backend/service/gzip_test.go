package service

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func gzipTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gzipMiddleware())
	router.GET("/json", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"payload": strings.Repeat("market ", 200)})
	})
	router.GET("/binary", func(c *gin.Context) {
		c.Data(http.StatusOK, "image/png", []byte("not really a png"))
	})
	router.GET("/empty", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return router
}

func TestGzipMiddleware_CompressesJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	rec := httptest.NewRecorder()
	gzipTestRouter().ServeHTTP(rec, req)

	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	if !strings.Contains(rec.Header().Get("Vary"), "Accept-Encoding") {
		t.Errorf("Vary = %q, want Accept-Encoding", rec.Header().Get("Vary"))
	}
	reader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("response is not gzip: %s", err)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading gzip body: %s", err)
	}
	if !strings.Contains(string(body), `"payload":"market market`) {
		t.Errorf("unexpected decompressed body: %.60s", body)
	}
}

func TestGzipMiddleware_LeavesOtherResponsesAlone(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		header map[string]string
	}{
		{"client without gzip", http.MethodGet, "/json", nil},
		{"websocket upgrade", http.MethodGet, "/json", map[string]string{"Accept-Encoding": "gzip", "Upgrade": "websocket"}},
		{"head request", http.MethodHead, "/json", map[string]string{"Accept-Encoding": "gzip"}},
		{"non-text content type", http.MethodGet, "/binary", map[string]string{"Accept-Encoding": "gzip"}},
		{"no content", http.MethodGet, "/empty", map[string]string{"Accept-Encoding": "gzip"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			for k, v := range tt.header {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			gzipTestRouter().ServeHTTP(rec, req)
			if got := rec.Header().Get("Content-Encoding"); got != "" {
				t.Errorf("Content-Encoding = %q, want none", got)
			}
		})
	}
}
