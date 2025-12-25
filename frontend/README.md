# feedsystem_video_go frontend

这是对接 `backend/`（Gin + GORM + MySQL + JWT）的一套 Vue3 前端调试 UI，覆盖全部后端路由：
- Account：注册 / 登录 / 改密码 / 查找 / 改名 / 登出
- Video：发布 / 按作者列出 / 详情
- Like：点赞 / 取消点赞 / 是否点赞
- Comment：列表 / 发布 / 删除
- Social：关注 / 取关 / 粉丝列表 / 关注列表
- Feed：最新流 / 点赞数流 / 关注流

## 开发启动

先启动后端：

```bash
cd backend
go run ./cmd
```

再启动前端：

```bash
cd frontend
npm install
npm run dev
```

默认通过 Vite 代理转发请求：前端访问 `/api/...` → `http://localhost:8080/...`（见 `frontend/vite.config.ts`）。
