# Bug

请求取消与服务关闭信号在多层被切断，后台重试持续运行并写入旧状态。

# Trigger

取消首请求后处理正常请求，再启动后台重试并在第二次回调时发出关闭信号。

# Error

`shutdown version={Version:1 State:running Committed:false}`
