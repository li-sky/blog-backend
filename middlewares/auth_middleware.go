package middlewares

import (
	"blog-backend/models"
	"blog-backend/services"
	"blog-backend/utils"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	contextUserKey   = "currentUser"
	contextClaimsKey = "jwtClaims"
)

// AuthMiddleware enforces JWT authentication and loads the current user into the context.
func AuthMiddleware(userService *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			unauthorized(c, "missing authorization token")
			return
		}

		claims, err := utils.ParseToken(token)
		if err != nil {
			unauthorized(c, "invalid or expired token")
			return
		}

		user, err := userService.GetByID(claims.UserID)
		if err != nil || user == nil {
			unauthorized(c, "user not found")
			return
		}

		c.Set(contextUserKey, user)
		c.Set(contextClaimsKey, claims)
		c.Next()
	}
}

// OptionalAuthMiddleware attempts to authenticate the user but does not enforce it.
func OptionalAuthMiddleware(userService *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			fmt.Println("OptionalAuthMiddleware: No token found")
			c.Next()
			return
		}

		claims, err := utils.ParseToken(token)
		if err != nil {
			fmt.Println("OptionalAuthMiddleware: Invalid token:", err)
			// Invalid token, treat as anonymous
			c.Next()
			return
		}

		user, err := userService.GetByID(claims.UserID)
		if err != nil || user == nil {
			fmt.Println("OptionalAuthMiddleware: User not found for ID:", claims.UserID)
			// User not found, treat as anonymous
			c.Next()
			return
		}

		fmt.Printf("OptionalAuthMiddleware: User authenticated: %s (ID: %d)\n", user.Username, user.ID)
		c.Set(contextUserKey, user)
		c.Set(contextClaimsKey, claims)
		c.Next()
	}
}

// CurrentUser pulls the authenticated user from the Gin context.
func CurrentUser(c *gin.Context) (*models.User, bool) {
	value, ok := c.Get(contextUserKey)
	if !ok {
		return nil, false
	}
	user, ok := value.(*models.User)
	return user, ok
}

// ClaimsFromContext retrieves the parsed JWT claims from the context.
func ClaimsFromContext(c *gin.Context) (*utils.Claims, bool) {
	value, ok := c.Get(contextClaimsKey)
	if !ok {
		return nil, false
	}
	claims, ok := value.(*utils.Claims)
	return claims, ok
}

func extractToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if header != "" {
		parts := strings.SplitN(header, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	if token := c.Query("token"); token != "" {
		return token
	}

	if token, err := c.Cookie("token"); err == nil && token != "" {
		return token
	}

	return ""
}

func unauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": message})
}
