# Bug

发布状态与事件缺少单调版本保护，晚回调会覆盖成功终态并重复记录副作用。

# Trigger

第一次发布失败、第二次成功后，再交付首轮版本 1 回调并读取最终事件。

# Error

`publish state={Version:1 State:running Committed:false} sideEffects=2`
