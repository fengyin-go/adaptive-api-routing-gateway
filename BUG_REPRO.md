# Bug

请求取消信号未传到下游，并且前后请求的调用作用域发生复用。

# Trigger

取消第一个请求后立即处理第二个正常请求，记录两次处理的错误和下游调用数。

# Error

`first=<nil> second=<nil> calls=2`
