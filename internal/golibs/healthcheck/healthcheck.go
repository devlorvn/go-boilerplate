package healthcheck

import (
	"context"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthChecker struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewHealthChecker(db *gorm.DB, rdb *redis.Client) *HealthChecker {
	return &HealthChecker{db: db, redis: rdb}
}

// Handler trả về 200 OK nếu DB & Redis hoạt động bình thường
func (h *HealthChecker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		// 1. Check Postgres DB Connection
		if h.db != nil {
			sqlDB, err := h.db.DB()
			if err != nil || sqlDB.PingContext(ctx) != nil {
				http.Error(w, `{"status":"DOWN", "reason":"database ping failed"}`, http.StatusServiceUnavailable)
				return
			}
		}

		// 2. Check Redis Connection
		if h.redis != nil {
			if err := h.redis.Ping(ctx).Err(); err != nil {
				http.Error(w, `{"status":"DOWN", "reason":"redis ping failed"}`, http.StatusServiceUnavailable)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"UP"}`))
	}
}
