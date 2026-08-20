# API 网关（api-gateway）

一个纯 Go 标准库实现的反向代理 API 网关：管理上游服务、路由、接入应用与密钥、限流规则，支持鉴权、限流、熔断、健康检查、访问日志与统计报表。

## 技术栈

- 纯 Go 标准库（`net/http` + `net/http/httputil`），零第三方依赖
- 标准分层：`cmd` / `internal(app|config|model|store|service|handler|middleware|ratelimit|circuitbreaker)` / `pkg`
- 内存存储（`sync.RWMutex` 保证并发安全）

## 运行

```bash
go run ./cmd/server
# 可选环境变量：
#   PORT(默认 8080)、ADDR、MAX_PAGE_SIZE(默认 100)、LOG_LEVEL
#   ADMIN_TOKEN(默认 admin-secret，管理 API 鉴权)
#   HEALTH_CHECK_INTERVAL(默认 30，健康检查间隔秒数)
```

## 核心特性

- **路由转发**：按 method + path 最长前缀匹配，支持 `strip_prefix` 前缀剥离
- **API Key 鉴权**：`X-API-Key` 请求头校验，密钥支持启用/禁用与过期时间
- **限流**：令牌桶算法，按路由绑定限流规则（`limit` 次 / `window` 秒）
- **熔断**：连续失败 5 次打开熔断器，冷却 30 秒后半开探测，成功即恢复
- **健康检查**：后台定时探测上游，连续失败 3 次标记 `unhealthy`
- **访问日志**：记录每次转发的状态码、耗时、客户端 IP
- **统计报表**：请求量（2xx/4xx/429/5xx）、按路由流量 TOP10
- **配置导入导出**：全量配置导出为 JSON，批量导入服务与限流规则

## 管理 API（需请求头 `X-Admin-Token`）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/services` | 创建上游服务 |
| GET | `/api/services` | 服务列表（`status`/`keyword` 筛选） |
| GET | `/api/services/{id}` | 服务详情 |
| PUT | `/api/services/{id}` | 更新服务 |
| DELETE | `/api/services/{id}` | 删除服务 |
| POST | `/api/routes` | 创建路由 |
| GET | `/api/routes` | 路由列表（`service_id`/`method` 筛选） |
| GET | `/api/routes/{id}` | 路由详情 |
| PUT | `/api/routes/{id}` | 更新路由 |
| DELETE | `/api/routes/{id}` | 删除路由 |
| POST | `/api/apps` | 创建接入应用 |
| GET | `/api/apps` | 应用列表 |
| GET | `/api/apps/{id}` | 应用详情 |
| PUT | `/api/apps/{id}` | 更新应用 |
| DELETE | `/api/apps/{id}` | 删除应用 |
| POST | `/api/api-keys` | 为应用生成密钥 |
| GET | `/api/api-keys` | 密钥列表（`app_id` 筛选） |
| GET | `/api/api-keys/{id}` | 密钥详情 |
| PATCH | `/api/api-keys/{id}/status` | 启用/禁用密钥 |
| DELETE | `/api/api-keys/{id}` | 删除密钥 |
| POST | `/api/rate-limit-rules` | 创建限流规则 |
| GET | `/api/rate-limit-rules` | 限流规则列表 |
| GET | `/api/rate-limit-rules/{id}` | 限流规则详情 |
| PUT | `/api/rate-limit-rules/{id}` | 更新限流规则 |
| DELETE | `/api/rate-limit-rules/{id}` | 删除限流规则 |
| GET | `/api/logs` | 访问日志列表（`route_id`/`app_id`/`status_code` 筛选） |
| GET | `/api/logs/{id}` | 日志详情 |
| GET | `/api/health` | 健康检查列表 |
| GET | `/api/health/{serviceId}` | 某服务健康状态 |
| POST | `/api/health/{serviceId}/probe` | 主动探测某服务 |
| GET | `/api/stats` | 网关统计报表 |
| GET | `/api/export` | 导出全量配置 |
| POST | `/api/import/services` | 批量导入服务 |
| POST | `/api/import/rate-limit-rules` | 批量导入限流规则 |

## 网关转发

所有非 `/api/` 前缀的请求进入网关转发：路由匹配 → 鉴权（可选）→ 限流（可选）→ 熔断检查 → 反向代理转发。

## 冒烟示例

```bash
# 1. 创建上游服务
curl -s -X POST localhost:8080/api/services -H 'X-Admin-Token: admin-secret' \
  -d '{"name":"user","base_url":"http://localhost:9000"}'
# 2. 创建路由（需鉴权）
curl -s -X POST localhost:8080/api/routes -H 'X-Admin-Token: admin-secret' \
  -d '{"path":"/users","method":"GET","service_id":"<服务ID>","requires_auth":true}'
# 3. 创建应用与密钥
curl -s -X POST localhost:8080/api/apps -H 'X-Admin-Token: admin-secret' -d '{"name":"client"}'
curl -s -X POST localhost:8080/api/api-keys -H 'X-Admin-Token: admin-secret' -d '{"app_id":"<应用ID>"}'
# 4. 通过网关转发（带密钥）
curl -s -H 'X-API-Key: <密钥>' localhost:8080/users/123
```
