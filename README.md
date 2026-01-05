# feedsystem_video_go

基于 Go 的短视频 Feed 系统（后端 + 前端），包含账号、视频、点赞、评论、关注与 Feed 流；支持 Redis 缓存与 RabbitMQ 异步 Worker（API 进程与 Worker 进程可拆分部署）。

详细设计与接口说明请阅读：`feedsystem_video_go项目设计.md`（包含模块设计、表结构、流程图与接口清单）。
## [项目演示](https://www.bilibili.com/video/BV1Dti7B9E6Y?vd_source=4b2884373b2c4c4147b10162c1709276)

## Docker Compose 一键启动（推荐）

要求：已安装 Docker Desktop / Docker Engine + Docker Compose。

```bash
docker compose up -d --build
```

访问：
- 前端：`http://localhost:5173`
- 后端 API：`http://localhost:8080`
- RabbitMQ 管理台：`http://localhost:15672`（默认账号 `admin` / `password123`）

说明：
- Compose 会启动 `mysql`、`redis`、`rabbitmq`、`backend`（API）、`worker`、`frontend`。
- 容器内后端配置使用 `backend/configs/config.docker.yaml`（会挂载到 `/app/configs/config.yaml`）。

## 本地开发启动（不容器化）

1) 先启动依赖（也可以只用 compose 拉起依赖）：
```bash
docker compose up -d mysql redis rabbitmq
```

2) 启动后端 API：
```bash
cd backend
go run ./cmd
```

3) 启动 Worker（消费 MQ、异步落库/更新 Redis 热榜）：
```bash
cd backend
go run ./cmd/worker
```

4) 启动前端（开发模式）：
```bash
cd frontend
npm install
npm run dev
```

前端默认使用 Vite 代理 `/api` 到 `http://127.0.0.1:8080`（见 `frontend/vite.config.ts`）。

