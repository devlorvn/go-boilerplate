package middleware

import (
	"net/http"
	"strings"

	userv1 "go-boilerplate/pkg/genproto/user/v1"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(authClient userv1.UserServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is required",
			})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header format must be Bearer {token}",
			})
			return
		}

		token := parts[1]
		resp, err := authClient.ValidateToken(c.Request.Context(), &userv1.ValidateTokenRequest{
			AccessToken: token,
		})
		if err != nil || !resp.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			return
		}

		c.Set("user_id", resp.UserId)
		c.Set("role", resp.Role)

		c.Request.Header.Set("X-User-ID", resp.UserId)
		c.Request.Header.Set("X-User-Role", resp.Role)

		c.Next()
	}
}
