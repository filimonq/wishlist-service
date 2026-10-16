package httpadapter

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const ctxRequestID = "request_id"

func requestLogging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := uuid.NewString()
		c.Set(ctxRequestID, requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()

		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status >= http.StatusBadRequest:
			level = slog.LevelWarn
		}

		slog.Log(c.Request.Context(), level, "http request",
			"request_id", requestID,
			"method", c.Request.Method,
			"route", c.FullPath(),
			"status", status,
			"duration_ms", float64(time.Since(start))/float64(time.Millisecond),
		)
	}
}

func recoverPanic(c *gin.Context, recovered any) {
	slog.Error("http panic",
		"request_id", c.GetString(ctxRequestID),
		"method", c.Request.Method,
		"route", c.FullPath(),
		"panic", recovered,
		"stack", string(debug.Stack()),
	)
	c.AbortWithStatus(http.StatusInternalServerError)
}
