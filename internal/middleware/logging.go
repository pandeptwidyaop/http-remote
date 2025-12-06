package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger is a middleware that logs HTTP requests.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method

		log.Printf("[%s] %s %s %d %v",
			method,
			path,
			c.ClientIP(),
			status,
			latency,
		)
	}
}

// PathPrefix is a middleware that stores the path prefix in the context.
func PathPrefix(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("path_prefix", prefix)
		c.Next()
	}
}
