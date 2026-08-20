# Bug

构建 panic 被恢复后，半初始化配置进入缓存并污染下一次同 key 构建。

# Trigger

第一次构建写入 partial 字段后 panic，接着用相同 key 正常构建并读取就绪状态。

# Error

`result={Key:broken Fields:map[status:reused] Ready:true} ready=false err=build broken: decoder`
