package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const maxLogBodySize = 1 << 12 // 4 KB

func RequestLogGin(logger *zap.Logger, mCounter *prometheus.CounterVec) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions ||
			c.Request.URL.Path == "/favicon.ico" ||
			strings.HasSuffix(c.Request.URL.Path, "/metrics") {
			c.Next()
			return
		}

		start := time.Now()

		// todo: debug level(dev/prod) / mask sensitive data
		var body string
		if c.Request != nil && c.Request.Body != nil {
			ct := c.GetHeader("Content-Type")
			if strings.HasPrefix(ct, "multipart/form-data") {
				body = "<multipart/form-data omitted>"
			} else {
				var buf bytes.Buffer
				limited := io.LimitReader(c.Request.Body, maxLogBodySize)
				_, _ = io.Copy(&buf, limited)
				body = buf.String()
				c.Request.Body.Close()
				c.Request.Body = io.NopCloser(bytes.NewBuffer(buf.Bytes()))
			}
		}

		c.Next()

		if mCounter != nil {
			mCounter.WithLabelValues("app_requests_total").Inc()
		}

		logger.Info("HTTP request",
			zap.String("method", c.Request.Method),
			zap.String("url", c.FullPath()),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", time.Since(start)),
			zap.String("body", body),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}
