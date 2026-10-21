package service

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

var gzipWriterPool = sync.Pool{New: func() any { return gzip.NewWriter(nil) }}

// gzipMiddleware compresses JSON and text responses for clients that send Accept-Encoding: gzip -
// the market listings alone are ~1 MB of JSON that shrinks to ~80 KB. Requests that never carry a
// compressible body are left alone: the live-match WebSocket upgrades (hijacked connections can't
// be wrapped), HEAD requests, responses that already set a Content-Encoding, and anything whose
// Content-Type isn't JSON or text.
func gzipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodHead ||
			c.Request.Header.Get("Upgrade") != "" ||
			!strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		gw := &gzipResponseWriter{ResponseWriter: c.Writer}
		c.Writer = gw
		defer gw.close()
		c.Next()
	}
}

// gzipResponseWriter decides on the first write whether to compress, once the handler has set the
// Content-Type, since only then is it known whether the body is worth compressing.
type gzipResponseWriter struct {
	gin.ResponseWriter
	gz      *gzip.Writer
	decided bool
}

func (w *gzipResponseWriter) decide() {
	if w.decided {
		return
	}
	w.decided = true

	header := w.Header()
	if header.Get("Content-Encoding") != "" || !isCompressibleContentType(header.Get("Content-Type")) {
		return
	}
	if status := w.Status(); status == http.StatusNoContent || status == http.StatusNotModified {
		return
	}

	header.Set("Content-Encoding", "gzip")
	header.Add("Vary", "Accept-Encoding")
	header.Del("Content-Length")
	w.gz = gzipWriterPool.Get().(*gzip.Writer)
	w.gz.Reset(w.ResponseWriter)
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	w.decide()
	if w.gz == nil {
		return w.ResponseWriter.Write(data)
	}
	return w.gz.Write(data)
}

func (w *gzipResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *gzipResponseWriter) Flush() {
	if w.gz != nil {
		w.gz.Flush()
	}
	w.ResponseWriter.Flush()
}

// close finishes the gzip stream and returns the writer to the pool.
func (w *gzipResponseWriter) close() {
	if w.gz == nil {
		return
	}
	w.gz.Close()
	gzipWriterPool.Put(w.gz)
	w.gz = nil
}

func isCompressibleContentType(contentType string) bool {
	contentType = strings.ToLower(contentType)
	return strings.HasPrefix(contentType, "application/json") || strings.HasPrefix(contentType, "text/")
}
