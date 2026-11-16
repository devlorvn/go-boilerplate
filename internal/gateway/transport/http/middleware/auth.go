package middleware

import (
	"net/http"
	"strings"

	userv1 "go-boilerplate/pkg/genproto/user/v1"

	"github.com/gin-gonic/gin"
)

// Auth rejects requests without a valid token before they reach upstream
// services. It is an early filter only: upstreams re-verify the token and
// apply their own RBAC, so a bug here cannot grant access.
func Auth(users userv1.UserServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
		if !ok || !strings.EqualFold(scheme, "bearer") || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "Unauthenticated", "message": "Authorization: Bearer <token> required"})
			return
		}
		resp, err := users.ValidateToken(c.Request.Context(), &userv1.ValidateTokenRequest{AccessToken: token})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"code": "Unavailable", "message": "auth service unavailable"})
			return
		}
		if !resp.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "Unauthenticated", "message": "invalid or expired token"})
			return
		}
		c.Set("user_id", resp.UserId)
		c.Set("role", resp.Role)
		c.Next()
	}
}
