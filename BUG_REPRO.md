# Bug

请求池对象跨租户复用后，先前保留的快照会变成新租户身份和请求头。

# Trigger

让租户 A 归还对象，再固定由租户 B 复用同一对象，最后读取 A 与 B 的快照。

# Error

`a={Tenant:tenant-b Headers:[b]} b={Tenant:tenant-b Headers:[b]}`
