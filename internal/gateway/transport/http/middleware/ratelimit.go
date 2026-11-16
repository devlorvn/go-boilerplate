package middleware

import (
	"fmt"
	"net/http"
	"time"

	"go-boilerplate/internal/golibs/redis"

	"github.com/gin-gonic/gin"
)

func RateLimit(rClient *redis.RedisClient, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rClient == nil || rClient.Client == nil {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		key := fmt.Sprintf("ratelimit:%s:%d", clientIP, time.Now().Unix()/int64(window.Seconds()))

		count, err := rClient.Client.Incr(c.Request.Context(), key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			rClient.Client.Expire(c.Request.Context(), key, window)
		}

		if count > int64(limit) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests, please try again later",
			})
			return
		}

		c.Next()
	}
}
