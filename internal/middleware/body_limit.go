package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodySizeLimit limits the maximum request body size.
// maxBytes is the maximum allowed body size in bytes.
func BodySizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip for GET, HEAD, OPTIONS methods
		if c.Request.Method == http.MethodGet ||
			c.Request.Method == http.MethodHead ||
			c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		// Check Content-Length header first
		if c.Request.ContentLength > maxBytes {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "request body too large",
			})
			c.Abort()
			return
		}

		// Wrap the body reader to enforce limit
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		c.Next()
	}
}

// DefaultBodyLimit returns middleware with 1MB limit
func DefaultBodyLimit() gin.HandlerFunc {
	return BodySizeLimit(1 << 20) // 1 MB
}

// SmallBodyLimit returns middleware with 64KB limit for sensitive endpoints
func SmallBodyLimit() gin.HandlerFunc {
	return BodySizeLimit(64 << 10) // 64 KB
}
