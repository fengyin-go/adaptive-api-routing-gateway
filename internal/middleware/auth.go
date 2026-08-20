// Package middleware 提供网关通用 HTTP 中间件。
package middleware

import (
	"net/http"

	"apigateway/pkg/httpx"
)

// AdminAuth 管理 API 鉴权中间件：校验 X-Admin-Token 请求头。
func AdminAuth(adminToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-Admin-Token")
			if adminToken == "" || token != adminToken {
				httpx.Unauthorized(w, "无效的管理员令牌")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
