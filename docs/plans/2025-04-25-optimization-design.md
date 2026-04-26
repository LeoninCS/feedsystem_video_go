# feedsystem_video_go 全项目优化设计文档

> **日期**: 2025-04-25
> **状态**: 待实施
> **方案**: 风险驱动分批（方案 A）

## 概述

基于全项目代码审查，识别出 17 项优化点，覆盖安全、数据库、MQ 可靠性、代码质量、架构、前端六大维度。按风险优先级分为三批实施。

---

## P1 止血（4项）— 消除生产风险

### 1. Router 变量赋值 Bug

- **文件**: `backend/internal/http/router.go:147`
- **问题**: `timelineMQ` 初始化失败时错误地将 `socialMQ` 设为 nil
- **修复**: `socialMQ = nil` → `timelineMQ = nil`
- **影响**: 1 行

### 2. 数据库复合索引

- **文件**: `backend/internal/video/video_entity.go`
- **问题**: Feed 流排序查询缺少索引，可能全表扫描
- **修复**: 在 Video 模型 GORM tag 中添加 3 个复合索引
  - `idx_videos_create_time` — `ListLatest`
  - `idx_videos_likes_count_id` — `ListLikesCountWithCursor`
  - `idx_videos_popularity_time_id` — `ListByPopularity`
- **实施**: 修改 model tag + AutoMigrate 自动创建

### 3. ListByAuthorID 加 LIMIT

- **文件**: `backend/internal/video/video_repo.go:39-48`
- **问题**: 查询无上限，单作者海量视频可导致内存溢出
- **修复**: 加 `Limit(200)` 硬上限

### 4. MQ Worker 死信队列 + 退避重试

- **文件**: `middleware/rabbitmq/` + 4 个 Worker 文件
- **问题**: 所有 Worker 使用 `Nack(false, true)` 无限重试
- **修复**: 
  - 声明死信交换 + 死信队列
  - 利用 `x-death` header 判断重试次数，≥3 次 Ack 并告警

---

## P2 加固（6项）— 安全隐患 + 规范化

### 5. rand.Read 错误处理

- **文件**: `backend/internal/video/video_handler.go:164-168`
- **问题**: 忽略 `rand.Read` 错误，失败时文件名全零可能覆盖
- **修复**: `randHex()` 返回 error，调用方处理

### 6. Handler 错误码精确化

- **文件**: 所有 handler 文件
- **问题**: DB/内部错误统一返回 400
- **修复**: 新增 `classifyHTTPStatus()` 辅助函数，Service 层返回哨兵错误区分 400/401/404/500

### 7. JWT Secret 弱默认值

- **文件**: `backend/internal/auth/jwt.go`
- **问题**: 默认值 `"change-me-in-env"` 过于明显
- **修复**: 未设环境变量时生成随机密钥并警告

### 8. 配置密码集中管理

- **文件**: `docker-compose.yml` + 3 个 config YAML
- **问题**: 多处重复硬编码密码
- **修复**: docker-compose 引用 `.env`，创建 `.env.example`，config YAML 保持现状

### 9. 前端路由鉴权守卫

- **文件**: `frontend/src/router/index.ts`
- **问题**: Settings/Video 页面无登录拦截
- **修复**: 添加 `router.beforeEach` 守卫

### 10. pprof 生产保护

- **现状**: 已监听 `127.0.0.1`，`config.docker.yaml` 已禁用
- **动作**: 确认安全，仅需注释说明

---

## P3 优化（7项）— 架构 + 可维护性

### 11. HomeView.vue 拆分

- **文件**: `frontend/src/views/HomeView.vue` (918 行)
- **拆分目标**:
  - `composables/useVideoFeed.ts`
  - `composables/useVideoPlayer.ts`
  - `composables/useLikeFollow.ts`
  - `components/CommentDrawer.vue`
  - `views/HomeView.vue`（精简至 ~350 行）

### 12. Feed Service 策略拆分

- **文件**: `backend/internal/feed/service.go` (547 行)
- **拆分目标**: 按查询策略拆为 4 个文件
  - `strategy_latest.go` — 热冷分离 + ZSET
  - `strategy_follow.go` — 缓存穿透防护
  - `strategy_hot.go` — 快照合并 + 降级
  - `build_feed.go` — 公共方法

### 13. 视频列表虚拟滚动

- **问题**: 所有视频渲染 DOM，内存压力大
- **修复**: 仅保留当前 ±1 条 DOM，离屏 `display:none` + `pause()`

### 14. 缓存键版本化

- **文件**: `middleware/redis/redis.go` + 所有 service
- **修复**: `Client` 增加 `keyPrefix`，所有键通过 `c.Key()` 生成

### 15. Docker 健康检查

- **文件**: `docker-compose.yml`
- **修复**: 为 backend/worker/frontend 增加 healthcheck

### 16. 前端错误监控

- **文件**: `frontend/src/api/client.ts` + 新增 `utils/error-reporter.ts`
- **修复**: 增加全局错误上报钩子

### 17. Worker 优雅重启

- **文件**: `backend/cmd/worker/main.go`
- **修复**: 替换 `log.Fatal` 为指数退避重试

---

## 实施顺序

```
P1 (第1周)     P2 (第2周)      P3 (第3-4周)
────────────────────────────────────────────
#1 Router Bug   #5 rand.Read    #11 HomeView 拆分
#2 DB 索引      #6 错误码       #12 Feed 拆分
#3 LIMIT        #7 JWT          #13 虚拟滚动
#4 MQ 死信      #8 密码管理     #14 缓存版本化
                #9 路由守卫     #15 健康检查
                #10 pprof       #16 错误监控
                                #17 Worker 重启
```

每批独立验证：`go test ./...` + `npm run build` + 冒烟测试。
