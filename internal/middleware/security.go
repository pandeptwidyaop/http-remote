package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds security-related HTTP headers to responses.
// These headers help protect against common web vulnerabilities.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking by disallowing the page to be embedded in iframes
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS filter in browsers (legacy, but still useful)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Control referrer information sent with requests
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy (formerly Feature-Policy)
		// Disable access to sensitive browser features
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Content Security Policy
		// Allow scripts and styles from same origin, and inline for SPA compatibility
		// Allow connections to self for API calls and WebSocket
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' 'unsafe-eval'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: blob:; "+
				"font-src 'self' data:; "+
				"connect-src 'self' ws: wss:; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'")

		// Cache control for sensitive pages
		// Don't cache API responses by default
		if len(c.Request.URL.Path) > 4 && c.Request.URL.Path[:4] == "/api" {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}

		c.Next()
	}
}

// StrictTransportSecurity adds HSTS header for HTTPS connections.
// This should only be used when the server is behind HTTPS.
func StrictTransportSecurity(maxAge int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only add HSTS header if the request came over HTTPS
		// Check X-Forwarded-Proto for reverse proxy setups
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}
