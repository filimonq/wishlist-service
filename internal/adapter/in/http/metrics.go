package httpadapter

import (
	"time"

	"github.com/gin-gonic/gin"
)

func RequestMetrics(observe func(method, route string, status int, seconds float64)) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		defer func() {
			route := c.FullPath()
			if route == "" {
				route = "unmatched"
			}
			observe(c.Request.Method, route, c.Writer.Status(), time.Since(start).Seconds())
		}()
		c.Next()
	}
}
