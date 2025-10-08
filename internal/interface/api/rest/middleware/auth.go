package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"user-manager-api/internal/infrastructure/jwt"
)

const (
	CtxUserRole = "userRole"
	CtxUserID   = "userID"
)

func AuthMiddleware(jwtService *jwt.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": "missing Authorization header"},
			)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": "invalid token format"},
			)
			return
		}
		
		claims, err := jwtService.ValidateToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": "invalid token"},
			)
			return
		}

		c.Set(CtxUserRole, claims.Role)
		c.Set(CtxUserID, claims.UserID)

		c.Next()
	}
}
