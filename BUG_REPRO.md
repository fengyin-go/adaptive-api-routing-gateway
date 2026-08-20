# Bug

Fanout 批次快照跨请求污染，错误分支的任务计数、取消和结果通道关闭顺序也不一致。

# Trigger

先交错提交 fanout-a 与 fanout-b，再让一个上游返回错误并等待汇聚退出，随后执行正常 fanout。

# Error

`stored fanout batch changed: [changed]`
