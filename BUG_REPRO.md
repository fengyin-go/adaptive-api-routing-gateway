# Bug

重试成功后的终态会被首轮延迟回调覆盖，并重复计算外部操作。

# Trigger

依次写入版本 1 运行中、版本 2 完成，再让版本 1 回调最后到达。

# Error

`state={Version:1 State:running Committed:false} sideEffects=2`
