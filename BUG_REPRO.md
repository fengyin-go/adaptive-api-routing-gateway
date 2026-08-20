# Bug

连续提交两个路由批次时，第二批会覆写第一批的返回快照和缓存。

# Trigger

先提交包含 `route-a`、`route-b` 的 alpha 批次，再复用输入缓冲提交 beta 批次，随后读取 alpha 回执与缓存。

# Error

`first batch changed: queued=[route-x route-b] cached=[route-x route-b]`
