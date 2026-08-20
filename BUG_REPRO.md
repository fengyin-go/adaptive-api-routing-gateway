# Bug

拒绝写入被重试并提交，批处理错误又被审计为成功，资源未及时释放。

# Trigger

先执行拒绝事务，再让三项批处理在第二项失败，最后运行一个正常批次。

# Error

`resource version={Version:1 State:running Committed:false}`
