# Bug

业务拒绝被当成临时错误重试，失败事务还留下已提交状态。

# Trigger

先返回业务拒绝，再执行一次首轮临时失败、次轮成功的请求，检查尝试次数和事务状态。

# Error

`reject count=2 state={Version:2 State:failed Committed:true}`
