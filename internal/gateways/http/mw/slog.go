package mw

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// GinSlog — log HTTP-request with slog.Logger
func GinSlog(l *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		req := c.Request
		path := req.URL.Path
		query := req.URL.RawQuery

		c.Next()

		lat := time.Since(start)
		status := c.Writer.Status()

		attrs := []any{
			"status", status,
			"method", req.Method,
			"path", path,
			"query", query,
			"ip", c.ClientIP(),
			"ua", req.UserAgent(),
			"latency_ms", float64(lat.Microseconds()) / 1000.0,
			"size", c.Writer.Size(),
		}
		if rid := c.Writer.Header().Get("X-Request-ID"); rid != "" {
			attrs = append(attrs, "request_id", rid)
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.ByType(gin.ErrorTypeAny).String())
		}

		switch {
		case status >= 500:
			l.Error("http request", attrs...)
		case status >= 400:
			l.Warn("http request", attrs...)
		default:
			l.Info("http request", attrs...)
		}
	}
}
