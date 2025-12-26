# feedsystem_video_go

基于 Go 的视频 Feed 系统后端，提供账号、视频、点赞、评论、关注（Social）与 Feed 等接口。默认技术栈：`Gin + GORM + MySQL + JWT (+ Redis 可选)`。

仓库内附 Postman Collection（`test/postman.json`），可用于手工/批量调试接口，并包含部分断言脚本。

## 目录结构
- `backend/cmd/`：程序入口（`backend/cmd/main.go`）
- `backend/configs/`：YAML 配置（`backend/configs/config.yaml`）
- `backend/internal/account/`：账号模块
- `backend/internal/video/`：视频 / 点赞 / 评论模块
- `backend/internal/social/`：关注模块
- `backend/internal/feed/`：Feed 模块
- `backend/internal/http/`：Gin 路由注册（`backend/internal/http/router.go`）
- `backend/internal/middleware/`：JWT 中间件（`backend/internal/middleware/jwt.go`）
- `test/`：Postman Collection（推荐使用 `test/postman.json`）

## 快速开始
1. 准备 MySQL 并创建数据库（库名/账号密码在 `backend/configs/config.yaml` 配置）。
2. 安装依赖：`cd backend && go mod tidy`
3. 启动服务：直接 `./start.sh`（默认同时启动后端+前端），或仅启动后端：`cd backend && go run ./cmd`
4. Postman 导入：`test/postman.json`（默认 `host=http://localhost:8080`）

> 启动时会自动执行 `AutoMigrate`（见 `backend/internal/db/db.go`）。

Redis 为可选依赖：未配置/不可用时服务仍可启动，但不会启用缓存加速。

## 前端（Vue）
前端工程在 `frontend/`，已对接本仓库后端全部接口：
- 启动：`cd frontend && npm install && npm run dev`
- 默认通过 Vite 代理：`/api/...` → `http://localhost:8080/...`（见 `frontend/vite.config.ts`）
- 一键启动：`./start.sh`；只启动后端可用 `START_FRONTEND=0 ./start.sh`
- 页面：
  - `/`：推荐（最新流）、点赞榜、关注流
  - `/hot`：热榜（热度榜）

## 配置
配置文件：`backend/configs/config.yaml`

```yaml
server:
  port: 8080

database:
  host: localhost
  port: 3306
  user: root
  password: 123456
  dbname: feedsystem
```

可选环境变量：
- `JWT_SECRET`：JWT 签名密钥；不设置则使用默认值（仅建议本地调试）。
- `REDIS_ADDR`：Redis 地址（默认 `127.0.0.1:6379`），用于缓存/加速（连不上会自动降级为不使用缓存）。
- `REDIS_PASSWORD`：Redis 密码（可选）。
- `REDIS_DB`：Redis DB 库号（默认 `0`）。

## 认证说明（与代码一致）
- 认证 Header：`Authorization: Bearer <jwt>`
- 校验流程（见 `backend/internal/middleware/jwt.go`）：
  - 校验 JWT 签名与过期时间。
  - 校验该账号“当前有效 token”：优先查 Redis key `account:<accountID>`；Redis 读不到/失败则回退查 DB 的 `account.token`；DB 校验通过会回填 Redis（自愈）。
- 因此：
  - 同一账号再次登录会覆盖 token（旧 token 立即失效）
  - `/account/logout` 会清空 token（立即失效）
  - `/account/changePassword` 成功后会清空 token（需要重新登录）
  - `/account/rename` 成功后会返回新 token 并写回数据库（旧 token 立即失效）
- Feed 接口使用“软鉴权”（`SoftJWTAuth`）：可以不带 token；但如果带了 `Authorization`，必须是合法且未撤销的 token，否则返回 `401`。

## Redis 缓存/加速点（可选）
- 鉴权 token 校验：Redis key `account:<accountID>`（TTL 24h）。
- Feed 匿名流缓存：`/feed/listLatest`（短 TTL，见 `backend/internal/feed/service.go`）。
- 热榜：`/feed/listByPopularity`（Redis 时间窗 ZSET 聚合，见 `backend/internal/feed/service.go` 与 `backend/internal/video/video_service.go`）。
- 视频详情缓存：`/video/getDetail`（见 `backend/internal/video/video_service.go`）。

## 手动自测（推荐）
1. `POST /account/register` → `POST /account/login` 拿到 `token`。
2. 带 `Authorization: Bearer <token>` 调用任意 JWT 保护接口（如 `/like/isLiked`）应返回 `200`。
3. `POST /account/logout` 后，用旧 token 调用保护接口应返回 `401`。
4. Redis 兜底：停掉 Redis 后再请求保护接口应仍可通过（走 DB）；Redis 恢复后再请求会回填 Redis。

## Postman 建议测试流程
使用一体化集合：`test/postman.json`（含预置变量与自动保存脚本）。

建议运行顺序：
1. Account → Register Account
2. Account → Login (save jwt_token)
3. Account → Find By Username (save accountId / vloggerId)
4. Social → Follow / Get All Followers / Get All Vloggers / Unfollow（可选）
5. Video → Publish Video（会保存 `publishedVideoId`）
6. Feed → List By Following（可选；需要带 token 才是“关注流”）
7. Like / Comment / Feed 其它接口（可选）

注意：执行 `Account/Rename` 后，集合会把响应里的 `token` 覆盖到 `jwt_token`，否则后续鉴权接口会因为旧 token 失效而 `401`。

## API（路由与鉴权）
路由注册见 `backend/internal/http/router.go`，以下均为 `POST` + JSON body。

### 账号（`/account`）
| 路径 | 是否需要 JWT | 说明 |
|------|-------------|------|
| `/account/register` | 否 | `{"username":"alice","password":"pass123"}` |
| `/account/login` | 否 | `{"username":"alice","password":"pass123"}` → `{"token":"..."}` |
| `/account/changePassword` | 否 | `{"username":"alice","old_password":"pass123","new_password":"newpass456"}`（成功会登出） |
| `/account/findByID` | 否 | `{"id":1}` |
| `/account/findByUsername` | 否 | `{"username":"alice"}` |
| `/account/rename` | 是 | `{"new_username":"alice_new"}` → `{"token":"..."}`（返回新 token） |
| `/account/logout` | 是 | `{}` |

### 视频（`/video`）
| 路径 | 是否需要 JWT | 说明 |
|------|-------------|------|
| `/video/listByAuthorID` | 否 | `{"author_id":1}` |
| `/video/getDetail` | 否 | `{"id":1}` |
| `/video/publish` | 是 | `{"title":"demo","description":"...","play_url":"http://...","cover_url":"http://..."}`（必填：`title/play_url/cover_url`） |

### 点赞（`/like`）
| 路径 | 是否需要 JWT | 说明 |
|------|-------------|------|
| `/like/isLiked` | 是 | `{"video_id":1}` |
| `/like/like` | 是 | `{"video_id":1}` |
| `/like/unlike` | 是 | `{"video_id":1}` |

### 评论（`/comment`）
| 路径 | 是否需要 JWT | 说明 |
|------|-------------|------|
| `/comment/listAll` | 否 | `{"video_id":1}` |
| `/comment/publish` | 是 | `{"video_id":1,"content":"hello"}` |
| `/comment/delete` | 是 | `{"comment_id":1}`（仅作者可删） |

### 关注（`/social`，JWT 保护）
| 路径 | 是否需要 JWT | 说明 |
|------|-------------|------|
| `/social/follow` | 是 | `{"vlogger_id":1}` |
| `/social/unfollow` | 是 | `{"vlogger_id":1}` |
| `/social/getAllFollowers` | 是 | `{"vlogger_id":1}`（可为空：默认取当前登录账号） |
| `/social/getAllVloggers` | 是 | `{"follower_id":1}`（可为空：默认取当前登录账号） |

### Feed（`/feed`，软鉴权）
| 路径 | 是否需要 JWT | 说明 |
|------|-------------|------|
| `/feed/listLatest` | 否（可选 JWT） | `{"limit":10,"latest_time":0}` |
| `/feed/listLikesCount` | 否（可选 JWT） | `{"limit":10,"likes_count_before":0,"id_before":0}` |
| `/feed/listByPopularity` | 否（可选 JWT） | `{"limit":10,"as_of":0,"offset":0}` |
| `/feed/listByFollowing` | 是 | `{"limit":10}` |

分页说明：
- `/feed/listLatest`：`latest_time` 为 Unix 秒时间戳；响应 `next_time` 也为 Unix 秒（`0` 表示无下一页）。
- `/feed/listLikesCount`：使用复合游标分页：请求携带 `likes_count_before` + `id_before`（两者一起用；全为 `0` 表示第一页）；响应返回 `next_likes_count_before` + `next_id_before` 用于下一页请求。
- `/feed/listByPopularity`：稳定分页：第一页传 `as_of=0, offset=0`；响应返回 `as_of`（分钟时间戳）与 `next_offset`，下一页原样带回即可。

## 数据表（自动迁移）
启动时会执行 `AutoMigrate`（`backend/internal/db/db.go`），创建/更新：`Account`、`Video`、`Like`、`Comment`、`Social`。
