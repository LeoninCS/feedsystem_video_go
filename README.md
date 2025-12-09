# feedsystem_video_go

## 项目简介
基于 Go 的视频 feed 系统，提供用户注册 / 登录、视频发布与点赞、Feed 拉取等接口，默认使用 Gin + GORM + MySQL。仓库附带 Postman Collection 方便调试 API。

## 目录结构
- `cmd/`：程序入口（`main.go`）
- `configs/`：YAML 配置（`config.yaml`）
- `internal/account/`：账号实体、仓储与业务逻辑
- `internal/video/`：视频与点赞模块
- `internal/feed/`：Feed 列表
- `internal/http/`：HTTP 路由
- `test/`：Postman Collection（`account.json` 等）
- `start.sh`：启动脚本示例

## 快速开始
1. 准备 Go 1.21+ 与 MySQL 8+ 环境。
2. 创建 `feedsystem` 数据库并根据需要修改 `configs/config.yaml`。
3. 安装依赖：`go mod tidy`
4. 启动服务：`go run ./cmd`（或执行 `sh start.sh`）
5. 在 Postman 中导入 `test/*.json`，设置 `host=http://localhost:8080`、`jwt_token` 为登录返回的 token。

## 身份认证说明
- 公共接口：`/account/register`、`/account/login`、`/video/listByAuthorID`、`/video/getDetail`、`/like/getLikesCount`、`/feed/listLatest`。
- 需要 JWT 的接口：`/account/rename`、`/account/changePassword`、`/account/findByID`、`/account/findByUsername`、`/account/logout` 以及 `/video/publish`、`/like/*`。
- 授权头格式：`Authorization: Bearer <jwt>`。
- 用户修改密码后会立即使旧 token 失效，需要重新登录获取新的 JWT。

## API（账号相关）
| 方法 | 路径 | 功能 | 请求体 |
|------|------|------|--------|
| POST | `/account/register` | 注册账号 | `{"username": "...","password": "..."}` |
| POST | `/account/login` | 登录，返回 `token` | `{"username": "...","password": "..."}` |
| POST | `/account/changePassword` | 修改当前登录用户的密码（需要 JWT） | `{"old_password": "...","new_password": "..."}` |
| POST | `/account/rename` | 修改当前登录用户的昵称（需要 JWT） | `{"new_username": "..."}` |
| POST | `/account/findByID` | 根据 ID 查询账号（需要 JWT） | `{"id": 1}` |
| POST | `/account/findByUsername` | 根据用户名查询账号（需要 JWT） | `{"username": "alice"}` |
| POST | `/account/logout` | 注销当前 token（需要 JWT） | `{}` |

- 所有接口统一使用 JSON 请求 / 响应。
- 启动时调用 `db.AutoMigrate` 保证表结构最新。

## API（视频相关）
| 方法 | 路径 | 功能 | 请求体 |
|------|------|------|--------|
| POST | `/video/listByAuthorID` | 根据作者 ID 获取视频列表 | `{"author_id": 1}` |
| POST | `/video/getDetail` | 查询指定视频详情 | `{"id": 1}` |
| POST | `/video/publish` | 发布视频（需要 JWT，自动读取登录用户为作者） | `{"title": "...","description": "...","play_url": "http://..."}` |

## API（点赞相关）
| 方法 | 路径 | 功能 | 请求体 |
|------|------|------|--------|
| POST | `/like/getLikesCount` | 获取视频点赞总数 | `{"video_id": 1}` |
| POST | `/like/like` | 给视频点赞（需要 JWT） | `{"video_id": 1}` |
| POST | `/like/unlike` | 取消点赞（需要 JWT） | `{"video_id": 1}` |
| POST | `/like/isLiked` | 判断当前用户是否点赞（需要 JWT） | `{"video_id": 1}` |

## API（Feed 相关）
| 方法 | 路径 | 功能 | 请求体 |
|------|------|------|--------|
| POST | `/feed/listLatest` | 拉取瀑布流，`limit` 最大 50，`latest_time` 为空时默认当前时间 | `{"limit": 10,"latest_time": 0}` |
