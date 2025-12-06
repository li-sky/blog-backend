package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var defaultAllowedOrigins = []string{
	"http://localhost:3000",
}

// CORSMiddleware attaches permissive CORS headers so the frontend can call the API from the browser.
func CORSMiddleware(allowedOrigins ...string) gin.HandlerFunc {
	origins := allowedOrigins
	if len(origins) == 0 {
		origins = defaultAllowedOrigins
	}

	allowAll := len(origins) == 1 && origins[0] == "*"
	originSet := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		originSet[strings.TrimSpace(origin)] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		header := c.Writer.Header()
		switch {
		case allowAll:
			header.Set("Access-Control-Allow-Origin", "*")
		case originAllowed(originSet, origin):
			header.Set("Access-Control-Allow-Origin", origin)
		default:
			// Origin not allowed; proceed without CORS headers so browser blocks it.
			c.Next()
			return
		}

		header.Set("Vary", "Origin")
		header.Set("Access-Control-Allow-Credentials", "true")
		header.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With")
		header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		header.Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func originAllowed(originSet map[string]struct{}, origin string) bool {
	origin = strings.TrimSpace(origin)
	_, ok := originSet[origin]
	return ok
}
