package middleware

import (
	"net/http"

	"apigateway/internal/ratelimit"
	"apigateway/pkg/httpx"
)

// RateLimit 通用令牌桶限流中间件。
// rate 为每秒令牌数，burst 为桶容量；被限流时返回 429。
func RateLimit(rate float64, burst int) func(http.Handler) http.Handler {
	bucket := ratelimit.NewTokenBucket(rate, burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !bucket.Allow() {
				httpx.Error(w, http.StatusTooManyRequests, 429, "请求过于频繁，请稍后重试")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
