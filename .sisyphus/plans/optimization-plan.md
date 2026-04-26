# feedsystem_video_go 全项目优化实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use executing-plans to implement this plan task-by-task.

**Goal:** 修复 17 项代码审查发现的问题，涵盖安全性、数据库性能、MQ 可靠性、代码质量、前端架构，按三批渐进交付。

**Architecture:** 风险驱动分批 — P1 修复直接威胁稳定性的 Bug，P2 加固安全面和规范化，P3 前端拆分和服务瘦身。每批独立验证可发布。

**Tech Stack:** Go 1.24.5 + Gin + GORM + MySQL + Redis + RabbitMQ + Vue 3 + TypeScript + Pinia

**参考设计文档:** `docs/plans/2025-04-25-optimization-design.md`

---

## P1 止血（4项）

---

### Task 1: Router 变量赋值 Bug

**Files:**
- Modify: `backend/internal/http/router.go:145-148`

**Step 1: 修复**

```go
// 定位到 router.go 第 147 行
timelineMQ, err := rabbitmq.NewTimelineMQ(rmq)
if err != nil {
    log.Printf("timelineMQ init failed (mq disabled): %v", err)
    socialMQ = nil   // ❌ 当前
```

改为：
```go
    timelineMQ = nil // ✅
```

**Step 2: 编译验证**

```bash
cd backend && go build ./...
```
Expected: 编译成功 (exit code 0)

**Step 3: Commit**

```bash
git add backend/internal/http/router.go
git commit -m "fix: router timelineMQ 初始化失败时误将 socialMQ 置空"
```

---

### Task 2: 数据库复合索引

**Files:**
- Modify: `backend/internal/video/video_entity.go:5-16`

**Step 1: 修改 Video 模型**

```go
type Video struct {
    ID          uint      `gorm:"primaryKey" json:"id"`
    AuthorID    uint      `gorm:"index;not null" json:"author_id"`
    Username    string    `gorm:"type:varchar(255);not null" json:"username"`
    Title       string    `gorm:"type:varchar(255);not null" json:"title"`
    Description string    `gorm:"type:varchar(255);" json:"description,omitempty"`
    PlayURL     string    `gorm:"type:varchar(255);not null" json:"play_url"`
    CoverURL    string    `gorm:"type:varchar(255);not null" json:"cover_url"`
    CreateTime  time.Time `gorm:"autoCreateTime;index:idx_videos_create_time,sort:desc" json:"create_time"`
    LikesCount  int64     `gorm:"column:likes_count;not null;default:0;index:idx_videos_likes_count_id,priority:1,sort:desc" json:"likes_count"`
    Popularity  int64     `gorm:"column:popularity;not null;default:0;index:idx_videos_popularity_time_id,priority:1,sort:desc" json:"popularity"`
}
```

**Step 2: 编译验证**

```bash
cd backend && go build ./...
```
Expected: 编译成功

**Step 3: 验证索引创建（启动时 AutoMigrate 自动执行）**

启动后端，观察日志中无 MySQL 错误：
```bash
cd backend && CONFIG_PATH=configs/config.compose-local.yaml go run ./cmd 2>&1 | head -20
```
Expected: 启动成功，GORM AutoMigrate 完成无报错

**Step 4: Commit**

```bash
git add backend/internal/video/video_entity.go
git commit -m "perf: Video 表增加 Feed 流排序查询复合索引（create_time/likes_count/popularity）"
```

---

### Task 3: ListByAuthorID 加 LIMIT

**Files:**
- Modify: `backend/internal/video/video_repo.go:39-48`

**Step 1: 修改查询**

```go
func (vr *VideoRepository) ListByAuthorID(ctx context.Context, authorID uint) ([]Video, error) {
    var videos []Video
    if err := vr.db.WithContext(ctx).
        Where("author_id = ?", authorID).
        Order("create_time desc").
        Limit(200).
        Find(&videos).Error; err != nil {
        return nil, err
    }
    return videos, nil
}
```

**Step 2: 编译验证**

```bash
cd backend && go build ./...
```
Expected: 编译成功

**Step 3: Commit**

```bash
git add backend/internal/video/video_repo.go
git commit -m "fix: ListByAuthorID 加 Limit(200) 防止海量数据内存溢出"
```

---

### Task 4: MQ Worker 死信队列 + 退避重试

**Files:**
- Modify: `backend/internal/middleware/rabbitmq/` — 队列声明增加 DLX 参数
- Modify: `backend/internal/worker/likeworker.go` — handleDelivery 增加重试计数
- Modify: `backend/internal/worker/commentworker.go` — 同上
- Modify: `backend/internal/worker/socialworker.go` — 同上
- Modify: `backend/internal/worker/popularityworker.go` — 同上

**Step 1: 在 rabbitmq 包中声明 DLX**

在 `backend/internal/middleware/rabbitmq/` 中新增 `dlx.go`：

```go
package rabbitmq

import (
    "log"
    amqp "github.com/rabbitmq/amqp091-go"
)

const (
    DLXExchange   = "dlx.events"
    MaxRetryCount = 3
)

// DeclareDLX 声明死信交换和死信队列
func DeclareDLX(ch *amqp.Channel, queueName string) error {
    if err := ch.ExchangeDeclare(
        DLXExchange, "topic", true, false, false, false, nil,
    ); err != nil {
        return err
    }
    dlxQueue := queueName + ".dlx"
    _, err := ch.QueueDeclare(
        dlxQueue, true, false, false, false, nil,
    )
    if err != nil {
        return err
    }
    if err := ch.QueueBind(dlxQueue, "#", DLXExchange, false, nil); err != nil {
        return err
    }
    log.Printf("DLX declared: exchange=%s, queue=%s", DLXExchange, dlxQueue)
    return nil
}

// QueueArgsWithDLX 返回带 DLX 配置的队列参数
func QueueArgsWithDLX() amqp.Table {
    return amqp.Table{
        "x-dead-letter-exchange": DLXExchange,
        "x-message-ttl":          int32(60000), // 死信消息 60s 后移入 DLX 队列
    }
}

// GetRetryCount 从 x-death header 中提取重试次数
func GetRetryCount(d amqp.Delivery) int {
    deaths, ok := d.Headers["x-death"].([]interface{})
    if !ok || len(deaths) == 0 {
        return 0
    }
    death, ok := deaths[0].(amqp.Table)
    if !ok {
        return 0
    }
    count, ok := death["count"].(int64)
    if !ok {
        return 0
    }
    return int(count)
}
```

**Step 2: 修改 Worker 的队列声明，传入 DLX 参数**

以 LikeWorker 为例（其他 Worker 同理），修改 `likeworker.go` 中声明队列的地方。需要在每个 Worker 初始化时调用 `DeclareDLX`，并在 `QueueDeclare` 时传入 args。

在 `middleware/rabbitmq/` 中找到各 MQ 初始化函数（如 `NewLikeMQ`），修改队列声明加上 `QueueArgsWithDLX()`。

**Step 3: 修改 handleDelivery 增加重试上限**

```go
func (w *LikeWorker) handleDelivery(ctx context.Context, d amqp.Delivery) {
    if err := w.process(ctx, d.Body); err != nil {
        retryCount := rabbitmq.GetRetryCount(d)
        if retryCount >= rabbitmq.MaxRetryCount {
            log.Printf("like worker: max retries exceeded (%d), moving to DLX: %v", retryCount, err)
            _ = d.Ack(false) // Ack 触发 DLX
            return
        }
        log.Printf("like worker: failed to process message (retry %d/%d): %v", retryCount+1, rabbitmq.MaxRetryCount, err)
        _ = d.Nack(false, true)
        return
    }
    _ = d.Ack(false)
}
```

**Step 4: 编译验证**

```bash
cd backend && go build ./...
```
Expected: 编译成功

**Step 5: 功能验证**

启动 Worker，观察日志：
- 处理成功 → Ack
- 处理失败 < 3 次 → Nack 重试
- 处理失败 ≥ 3 次 → Ack（进入 DLX）+ 日志告警

**Step 6: Commit**

```bash
git add backend/internal/middleware/rabbitmq/dlx.go backend/internal/worker/
git commit -m "feat: MQ Worker 增加死信队列 — 重试上限 3 次后移入 DLX 并告警"
```

---

## P2 加固（6项）

---

### Task 5: rand.Read 错误处理

**Files:**
- Modify: `backend/internal/video/video_handler.go:164-168`
- Modify: 调用 `randHex()` 的 `UploadVideo` 和 `UploadCover` 方法

**Step 1: 修改 randHex 返回 error**

```go
func randHex(n int) (string, error) {
    b := make([]byte, n)
    if _, err := rand.Read(b); err != nil {
        return "", fmt.Errorf("rand.Read: %w", err)
    }
    return hex.EncodeToString(b), nil
}
```

**Step 2: 修改调用方**

在 `UploadVideo` (line 96) 和 `UploadCover` (line 148) 中：

```go
filename, err := randHex(16)
if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate filename"})
    return
}
filename = filename + ext
```

**Step 3: 编译验证**

```bash
cd backend && go build ./...
```
Expected: 编译成功

**Step 4: Commit**

```bash
git add backend/internal/video/video_handler.go
git commit -m "fix: randHex 不再忽略 rand.Read 错误，防止弱随机文件名冲突"
```

---

### Task 6: Handler 错误码精确化

**Files:**
- Create: `backend/internal/http/errors.go` — 哨兵错误 + 分类函数
- Modify: `backend/internal/video/video_service.go` — Service 返回哨兵错误
- Modify: `backend/internal/video/like_service.go` — 同上
- Modify: `backend/internal/video/video_handler.go` — Handler 使用 classifyHTTPStatus
- Modify: 其他 handler 文件同理

**Step 1: 创建哨兵错误和分类函数**

```go
// backend/internal/http/errors.go
package http

import (
    "errors"
    "net/http"

    "gorm.io/gorm"
)

var (
    ErrUnauthorized = errors.New("unauthorized")
    ErrValidation   = errors.New("validation error")
)

func ClassifyHTTPStatus(err error) int {
    switch {
    case err == nil:
        return http.StatusOK
    case errors.Is(err, ErrUnauthorized):
        return http.StatusUnauthorized
    case errors.Is(err, ErrValidation):
        return http.StatusBadRequest
    case errors.Is(err, gorm.ErrRecordNotFound):
        return http.StatusNotFound
    default:
        return http.StatusInternalServerError
    }
}
```

**Step 2: Service 层返回哨兵错误**

示例 — `video_service.go` 中 `Delete` 方法：

```go
if video.AuthorID != authorID {
    return http.ErrUnauthorized
}
```

`Publish` 方法中的参数校验：

```go
if video.Title == "" || video.PlayURL == "" || video.CoverURL == "" {
    return http.ErrValidation
}
```

**Step 3: Handler 层使用**

```go
// video_handler.go PublishVideo
if err := vh.service.Publish(c.Request.Context(), video); err != nil {
    c.JSON(httputil.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
    return
}
```

注意 package 命名冲突：handler 文件在 `package video`，需要 import `httputil "feedsystem_video_go/internal/http"`。

**Step 4: 编译验证 + 测试**

```bash
cd backend && go build ./... && go vet ./...
```
Expected: 编译通过

**Step 5: Commit**

```bash
git add backend/internal/http/errors.go backend/internal/video/video_handler.go backend/internal/video/video_service.go
git commit -m "refactor: Handler 错误码精确化 — 区分 400/401/404/500"
```

---

### Task 7: JWT Secret 弱默认值加固

**Files:**
- Modify: `backend/internal/auth/jwt.go:12-18`

**Step 1: 修改 jwtSecret**

```go
func jwtSecret() []byte {
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        b := make([]byte, 32)
        if _, err := rand.Read(b); err != nil {
            log.Printf("FATAL: cannot generate JWT secret: %v", err)
            return []byte("fallback-unsafe-key-change-me")
        }
        secret = hex.EncodeToString(b)
        log.Printf("WARNING: JWT_SECRET not set, generated random key. All tokens invalid on restart.")
    }
    return []byte(secret)
}
```

需要增加 import: `"crypto/rand"`, `"encoding/hex"`, `"log"`

**Step 2: 编译验证**

```bash
cd backend && go build ./...
```

**Step 3: Commit**

```bash
git add backend/internal/auth/jwt.go
git commit -m "security: JWT Secret 未设环境变量时生成随机密钥而非使用弱默认值"
```

---

### Task 8: 配置密码集中管理

**Files:**
- Create: `.env.example`
- Modify: `docker-compose.yml`
- Modify: `.gitignore` — 确保 `.env` 被忽略

**Step 1: 创建 .env.example**

```bash
# .env.example — 复制为 .env 后修改实际值
MYSQL_ROOT_PASSWORD=123456
MYSQL_DATABASE=feedsystem
REDIS_PASSWORD=123456
RABBITMQ_USER=admin
RABBITMQ_PASS=password123
JWT_SECRET=change-me-in-production
```

**Step 2: 修改 docker-compose.yml**

```yaml
mysql:
  environment:
    MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD:-123456}
    MYSQL_DATABASE: ${MYSQL_DATABASE:-feedsystem}

redis:
  command: ["redis-server", "--appendonly", "yes", "--requirepass", "${REDIS_PASSWORD:-123456}"]

rabbitmq:
  environment:
    RABBITMQ_DEFAULT_USER: ${RABBITMQ_USER:-admin}
    RABBITMQ_DEFAULT_PASS: ${RABBITMQ_PASS:-password123}
```

**Step 3: 确认 .gitignore 包含 .env**

```bash
grep '\.env' .gitignore || echo '.env' >> .gitignore
```

**Step 4: Commit**

```bash
git add .env.example docker-compose.yml .gitignore
git commit -m "security: 敏感配置迁移至 .env，docker-compose 引用环境变量"
```

---

### Task 9: 前端路由鉴权守卫

**Files:**
- Modify: `frontend/src/router/index.ts`

**Step 1: 添加 beforeEach 守卫**

```typescript
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  // ... 现有配置不变
})

router.beforeEach((to, _from, next) => {
  const auth = useAuthStore()
  const authRequired = ['/settings', '/video']

  if (authRequired.some(p => to.path.startsWith(p)) && !auth.isLoggedIn) {
    next({ path: '/account', query: { redirect: to.fullPath } })
    return
  }
  next()
})

export default router
```

**Step 2: 构建验证**

```bash
cd frontend && npm run build
```
Expected: 构建成功

**Step 3: Commit**

```bash
git add frontend/src/router/index.ts
git commit -m "feat: 前端路由鉴权守卫 — /settings 和 /video 需登录"
```

---

### Task 10: pprof 生产保护（确认）

**Files:** 无需改动

**Step 1: 确认已有配置安全**

```bash
grep -A5 'pprof' backend/configs/config.docker.yaml
```
Expected: `enabled: true`（不对，需要确认是 `false`）

确认 `config.docker.yaml` 中 pprof 已禁用或仅监听 `127.0.0.1`。

**Step 2: 如果未禁用则修改**

```yaml
observability:
  pprof:
    enabled: false
```

**Step 3: Commit（如有改动）**

```bash
git add backend/configs/config.docker.yaml
git commit -m "security: 确认 pprof 容器内部署时禁用"
```

---

## P3 优化（7项）

---

### Task 11: HomeView.vue 拆分为 composable + 子组件

**Files:**
- Create: `frontend/src/composables/useVideoFeed.ts`
- Create: `frontend/src/composables/useVideoPlayer.ts`
- Create: `frontend/src/composables/useLikeFollow.ts`
- Create: `frontend/src/components/CommentDrawer.vue`
- Modify: `frontend/src/views/HomeView.vue`（精简至 ~350 行）

**Step 1: 提取 useVideoFeed composable**

```typescript
// composables/useVideoFeed.ts
import { reactive, ref } from 'vue'
import { ApiError } from '../api/client'
import * as feedApi from '../api/feed'
import type { FeedVideoItem } from '../api/types'

export type TabKey = 'recommend' | 'hot' | 'following'

export function useVideoFeed() {
  const tab = ref<TabKey>('recommend')

  const recommend = reactive({
    items: [] as FeedVideoItem[],
    loading: false, error: '',
    hasMore: false, nextTime: 0,
  })

  const hot = reactive({
    items: [] as FeedVideoItem[],
    loading: false, error: '',
    hasMore: false,
    nextLikesCountBefore: undefined as number | undefined,
    nextIdBefore: undefined as number | undefined,
  })

  const following = reactive({
    items: [] as FeedVideoItem[],
    loading: false, error: '',
    hasMore: false, nextTime: 0,
  })

  // ... 复制原有 loadRecommend / loadHot / loadFollowing 逻辑
  // 各 load 函数保持原样

  return { tab, recommend, hot, following, loadRecommend, loadHot, loadFollowing }
}
```

**Step 2: 提取 useVideoPlayer composable**

```typescript
// composables/useVideoPlayer.ts
import { nextTick, ref } from 'vue'

export function useVideoPlayer() {
  const muted = ref(true)
  const activeIndex = ref(0)
  const videoMap = new Map<number, HTMLVideoElement>()

  function setVideoRef(id: number, el: HTMLVideoElement | null) { /* ... */ }
  function playActive() { /* ... */ }
  function toggleMute() { /* ... */ }
  function togglePlayPause() { /* ... */ }

  return { muted, activeIndex, videoMap, setVideoRef, playActive, toggleMute, togglePlayPause }
}
```

**Step 3: 提取 useLikeFollow composable**

```typescript
// composables/useLikeFollow.ts
import { reactive } from 'vue'
import { ApiError } from '../api/client'
import * as likeApi from '../api/like'
import { useAuthStore } from '../stores/auth'
import { useSocialStore } from '../stores/social'
import { useToastStore } from '../stores/toast'
import type { FeedVideoItem } from '../api/types'

export function useLikeFollow() {
  const likeBusy = reactive<Record<string, boolean>>({})
  const followBusy = reactive<Record<string, boolean>>({})

  async function toggleLike(item: FeedVideoItem) { /* ... */ }
  async function toggleFollow(authorId: number) { /* ... */ }
  function share(item: FeedVideoItem) { /* ... */ }

  return { likeBusy, followBusy, toggleLike, toggleFollow, share }
}
```

**Step 4: 提取 CommentDrawer.vue 组件**

将原 HomeView.vue 中 drawer 相关的 state + 模板 + 样式提取为独立组件。

**Step 5: 精简 HomeView.vue**

```vue
<script setup lang="ts">
import { useVideoFeed } from '../composables/useVideoFeed'
import { useVideoPlayer } from '../composables/useVideoPlayer'
import { useLikeFollow } from '../composables/useLikeFollow'
import CommentDrawer from '../components/CommentDrawer.vue'

const { tab, recommend, hot, following, loadRecommend, loadHot, loadFollowing } = useVideoFeed()
const { muted, activeIndex, videoMap, setVideoRef, playActive, toggleMute, togglePlayPause } = useVideoPlayer()
const { likeBusy, followBusy, toggleLike, toggleFollow, share } = useLikeFollow()

// ... watch/onMounted 逻辑用 composable 返回的方法
</script>
```

**Step 6: 构建验证**

```bash
cd frontend && npm run build
```
Expected: 构建成功，类型检查通过

**Step 7: Commit**

```bash
git add frontend/src/composables/ frontend/src/components/CommentDrawer.vue frontend/src/views/HomeView.vue
git commit -m "refactor: HomeView 拆分为 3 个 composable + CommentDrawer 组件"
```

---

### Task 12: Feed Service 策略拆分

**Files:**
- Create: `backend/internal/feed/strategy_latest.go`
- Create: `backend/internal/feed/strategy_follow.go`
- Create: `backend/internal/feed/strategy_hot.go`
- Create: `backend/internal/feed/build_feed.go`
- Modify: `backend/internal/feed/service.go`（精简入口）

**Step 1: 拆分 strategy_latest.go**

将原 `service.go` 中 `ListLatest` 方法完整移入，包含 ZSET 热冷分离逻辑。

**Step 2: 拆分 strategy_follow.go**

将 `ListByFollowing` 方法完整移入，包含 Redis 缓存穿透防护逻辑。

**Step 3: 拆分 strategy_hot.go**

将 `ListByPopularity` 方法完整移入，包含快照合并 + 降级逻辑。

**Step 4: 拆分 build_feed.go**

将 `buildFeedVideos` 和 `buildOrderedResult` 移入。

**Step 5: 精简 service.go**

```go
type FeedService struct {
    repo         *FeedRepository
    likeRepo     *video.LikeRepository
    rediscache   *rediscache.Client
    localcache   *cache.Cache
    cacheTTL     time.Duration
    requestGroup singleflight.Group
}

func (f *FeedService) ListLatest(ctx context.Context, limit int, latestBefore time.Time, viewerAccountID uint) (ListLatestResponse, error) {
    return listLatestStrategy(ctx, f, limit, latestBefore, viewerAccountID)
}
```

**Step 6: 编译验证 + 运行测试**

```bash
cd backend && go build ./... && go test ./...
```

**Step 7: Commit**

```bash
git add backend/internal/feed/
git commit -m "refactor: Feed Service 按查询策略拆分为 4 个文件"
```

---

### Task 13: 视频列表虚拟滚动

**Files:**
- Modify: `frontend/src/views/HomeView.vue`

**Step 1: 替换 v-for 为虚拟化渲染**

在模板中，将：
```html
<section v-for="(item, idx) in filteredItems" ...>
```
改为只渲染 `visibleRange` 内的 item，其余用占位 div。使用 `v-show` 控制显隐而非 `v-if`（保留 video 实例）。

```typescript
const visibleRange = computed(() => {
  const idx = activeIndex.value
  const len = filteredItems.value.length
  return {
    start: Math.max(0, idx - 1),
    end: Math.min(len - 1, idx + 1),
  }
})
```

模板中：
```html
<section
  v-for="(item, idx) in filteredItems"
  v-show="idx >= visibleRange.start && idx <= visibleRange.end"
  ...
>
```

**Step 2: 离屏视频 pause**

在 `playActive` 中，pause 所有不在 visibleRange 内的视频。

**Step 3: 构建验证**

```bash
cd frontend && npm run build
```

**Step 4: Commit**

```bash
git add frontend/src/views/HomeView.vue
git commit -m "perf: Feed 流虚拟滚动 — 仅渲染当前±1条视频 DOM"
```

---

### Task 14: 缓存键版本化

**Files:**
- Modify: `backend/internal/middleware/redis/redis.go`
- Modify: 所有使用 Redis 键的 service 文件（account, video, feed, social）

**Step 1: 在 Client 增加 keyPrefix**

```go
type Client struct {
    rdb       *redis.Client
    keyPrefix string
}

func (c *Client) Key(format string, args ...any) string {
    return c.keyPrefix + fmt.Sprintf(format, args...)
}
```

在 `NewFromEnv` 中从 config 读入 `keyPrefix`（默认 `"v1:"`）。

**Step 2: 替换所有硬编码键**

- `"feed:global_timeline"` → `c.Key("feed:global_timeline")`
- `"video:detail:id=%d"` → `c.Key("video:detail:id=%d", id)` （注意：Key 内部做 Sprintf）
- 等等...

**Step 3: 编译验证 + 测试**

```bash
cd backend && go build ./... && go test ./...
```

**Step 4: Commit**

```bash
git add backend/internal/middleware/redis/redis.go backend/internal/
git commit -m "refactor: Redis 缓存键增加版本前缀支持（默认 v1:）"
```

---

### Task 15: Docker 健康检查

**Files:**
- Modify: `docker-compose.yml`

**Step 1: 增加 backend healthcheck**

```yaml
backend:
  healthcheck:
    test: ["CMD-SHELL", "wget -qO- http://localhost:8080/account/findByID -d '{}' --header='Content-Type: application/json' || exit 1"]
    interval: 10s
    timeout: 5s
    retries: 3
```

**Step 2: worker healthcheck**

```yaml
worker:
  healthcheck:
    test: ["CMD-SHELL", "pgrep worker || exit 1"]
    interval: 15s
    timeout: 5s
    retries: 3
```

**Step 3: frontend healthcheck**

```yaml
frontend:
  healthcheck:
    test: ["CMD-SHELL", "wget -qO- http://localhost:80/ || exit 1"]
    interval: 10s
    timeout: 5s
    retries: 3
```

**Step 4: Commit**

```bash
git add docker-compose.yml
git commit -m "feat: docker-compose 增加 backend/worker/frontend 健康检查"
```

---

### Task 16: 前端错误监控

**Files:**
- Create: `frontend/src/utils/error-reporter.ts`
- Modify: `frontend/src/api/client.ts`
- Modify: `frontend/src/main.ts`

**Step 1: 创建 error-reporter**

```typescript
// utils/error-reporter.ts
export function reportError(error: Error, context?: Record<string, unknown>) {
  if (import.meta.env.DEV) {
    console.error('[ErrorReporter]', error.message, context)
    return
  }
  // 生产环境发送到日志服务
  fetch('/api/error-report', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      message: error.message,
      stack: error.stack,
      context,
      timestamp: new Date().toISOString(),
    }),
  }).catch(() => { /* 静默失败 */ })
}
```

**Step 2: 在 client.ts 中集成**

在 `ApiError` 抛出前调用 `reportError`。

**Step 3: 在 main.ts 中注册全局错误处理器**

```typescript
app.config.errorHandler = (err, _instance, info) => {
  reportError(err instanceof Error ? err : new Error(String(err)), { info })
}
```

**Step 4: 构建验证**

```bash
cd frontend && npm run build
```

**Step 5: Commit**

```bash
git add frontend/src/utils/error-reporter.ts frontend/src/api/client.ts frontend/src/main.ts
git commit -m "feat: 前端全局错误监控 — 开发 console，生产上报 API"
```

---

### Task 17: Worker 优雅重启

**Files:**
- Modify: `backend/cmd/worker/main.go`

**Step 1: 增加连接重试**

```go
func connectWithRetry(name string, fn func() error, maxRetries int) {
    for i := 0; i < maxRetries; i++ {
        if err := fn(); err == nil {
            return
        }
        wait := time.Duration(math.Min(float64(1<<i), 30)) * time.Second
        log.Printf("%s 不可用，%v 后重试 (%d/%d)...", name, wait, i+1, maxRetries)
        time.Sleep(wait)
    }
    log.Fatalf("%s: 超过最大重试次数", name)
}
```

替换原有的直接 fatal 调用：

```go
// 替换前：
db, err := dbconnect.NewDB(cfg)
if err != nil { log.Fatal(err) }

// 替换后：
var db *gorm.DB
connectWithRetry("MySQL", func() error {
    var err error
    db, err = dbconnect.NewDB(cfg)
    return err
}, 10)
```

**Step 2: 编译验证**

```bash
cd backend && go build ./cmd/worker
```

**Step 3: Commit**

```bash
git add backend/cmd/worker/main.go
git commit -m "fix: Worker 启动时 MySQL/Redis/MQ 不可用改为指数退避重试而非直接 fatal"
```

---

## 验证清单

每完成一批，执行以下验证：

| 批次 | 验证命令 | 预期 |
|------|---------|------|
| P1 | `cd backend && go build ./... && go vet ./...` | 编译通过，无 vet 告警 |
| P2 | `cd backend && go build ./... && go vet ./... && cd ../frontend && npm run build` | 全量构建通过 |
| P3 | `cd backend && go test ./... && cd ../frontend && npm run build && npx vue-tsc --noEmit` | 测试通过 + 类型检查通过 |
