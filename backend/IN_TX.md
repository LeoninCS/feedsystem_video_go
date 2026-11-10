# Repository InTx 事务模式

Service 决定事务边界，Repository 执行数据库操作。Service 通过仓储的
`Transaction(ctx, fn)` 开启事务，再将回调提供的 `tx` 传给每一步的
`XxxInTx(ctx, tx, ...)` 方法。

```go
err := likeRepo.Transaction(ctx, func(tx *gorm.DB) error {
    created, err := likeRepo.LikeInTx(ctx, tx, like)
    if err != nil {
        return err
    }
    if !created {
        return nil // 调用方决定重复点赞的业务语义
    }
    if err := videoRepo.ChangeLikesCountInTx(ctx, tx, like.VideoID, 1); err != nil {
        return err
    }
    return videoRepo.ChangePopularityInTx(ctx, tx, like.VideoID, 1)
})
```

回调返回 `nil` 时提交，返回错误时回滚；panic 时回滚并继续传播 panic。
所有参与操作的仓储必须属于同一数据库，回调内必须始终传入同一个 `tx`。
请求的 `context.Context` 同时用于开启事务和执行操作。

## 方法与现有业务

| Repository | 事务内方法 |
| --- | --- |
| LikeRepository | `LikeInTx`、`UnlikeInTx` |
| VideoRepository | `RequireVideoInTx`、`CreateVideoInTx`、`CreateMsgInTx`、`AddTagsInTx`、`ChangeLikesCountInTx`、`ChangePopularityInTx` |
| CommentRepository | `CreateCommentInTx` |
| AccountRepository | `RenameInTx`、`UpdateTokenInTx` |

四个 Repository 均提供 `Transaction`。

- `LikeInTx`：插入成功返回 `true, nil`；MySQL 1062 重复键返回 `false, nil`；其他数据库错误原样返回。空对象或零 ID 返回 `false, nil`，与原有 Worker 幂等入口一致。
- `UnlikeInTx`：根据 `RowsAffected` 判断是否实际删除。空对象或零 ID 返回 `false, nil`。
- 两个计数方法使用数据库表达式更新，并将结果下限限制为零。
- 点赞/取消点赞的同步降级、评论同步降级、视频与 Outbox/标签发布，以及账号改名与 token 更新均由 Service 组织事务。
- 原有不带 `InTx` 的公开入口保留；`RenameWithToken` 作为兼容入口保留，Service 已改用显式事务方法。
- 评论 @提醒的查询与写入已移至 Repository，仍在评论事务提交后执行。

本次改动保留点赞预检查、重复点赞错误文案和 MQ 行为。

## `tx == nil` 的含义

为兼容非事务调用，所有 `InTx` 方法允许传 `nil`，此时使用 Repository
持有的数据库连接，操作独立提交。现有非事务入口也可以复用此路径。

**事务回调内不要传 `nil`。** 它会使用另一条连接，既不能加入外层回滚，
又可能等待外层事务自己持有的行锁，形成自锁，直到
`innodb_lock_wait_timeout` 超时。该降级路径必须由调用方明确选择。

## 测试

从 `backend` 目录执行：

```powershell
go build ./...
go vet ./...
go test -count=1 ./...
```

默认单元测试使用 `go-sqlmock`，检查提交、错误回滚、panic 回滚、上下文、
事务句柄复用、重复键和删除行数。

真实 MySQL 测试需要设置以下环境变量；未设置时会明确跳过这些集成测试。
账号需要创建/删除数据库的权限。每个测试创建并清理独立的
`intx_test_<timestamp>` 数据库，不迁移或清空 DSN 指定的已有数据库。

```powershell
# 示例：连接一个专用于测试的本地 MySQL 实例。
$env:INTX_MYSQL_DSN = 'root:password@tcp(127.0.0.1:3306)/?parseTime=true'
go test -count=1 -v ./internal/video -run TestMySQL
go test -count=1 ./...
Remove-Item Env:INTX_MYSQL_DSN
```

集成测试检查实际数据：点赞后续步骤失败时回滚插入与计数，成功点赞/取消，
`nil tx` 独立提交与计数下限，视频/Outbox/标签原子性，评论回滚，改名/token 回滚。

具备 CGO 和 C 编译器的环境还应运行：

```powershell
go test -race -count=1 ./...
```
