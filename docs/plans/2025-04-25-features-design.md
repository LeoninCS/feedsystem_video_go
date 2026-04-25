# 用户体系 + 社交深化 设计文档

> **日期**: 2025-04-25
> **状态**: 待实施
> **方案**: 依赖驱动分批（3 阶段）

## 概述

在现有短视频 Feed 系统基础上，扩展用户 profile 体系（头像、简介、Refresh Token、主页统计）和社交互动能力（通知、私信、话题、@提及）。

---

## P1 — 用户基石（4 项）

### 1. 头像上传 + 个人简介

**Account 模型扩展**:
```go
AvatarURL string `gorm:"type:varchar(512)" json:"avatar_url,omitempty"`
Bio       string `gorm:"type:varchar(255)" json:"bio,omitempty"`
```

**新增接口**:
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/account/uploadAvatar` | multipart 上传头像，校验类型/大小，存 `.run/uploads/avatars/{id}/` |
| POST | `/account/updateProfile` | JSON `{ avatar_url?, bio? }` 更新当前用户 |

**前端**: UserAvatar 组件支持 `src`，AccountView 加头像上传 + bio 编辑，Feed 卡片显示头像。

### 2. 登录态优化（Refresh Token）

**双 Token**:
- Access Token: 15min 过期
- Refresh Token: 7天过期，落库 + Redis 缓存

**新增接口**: `POST /account/refresh` — 接收 refresh token 返回新 access token

**前端**: auth store 存双 token，client.ts 401 自动刷新，登录页"记住我"。

---

## P2 — Feed 可见 + 通知（3 项）

### 3. 粉丝数 / 关注数展示

**后端**: 社交接口返回中加 `follower_count` / `vlogger_count`，通过聚合查询计数。

**前端**: UserProfileView 和 Feed 卡片显示粉丝数。

### 4. 用户主页增强

**后端**: `POST /account/getProfile` — 返回用户信息 + 视频列表 + 获赞总数。

**前端**: UserProfileView 加视频列表网格 + 统计卡片。

### 5. 实时消息通知

**架构**: 复用 MQ 事件（like.events / comment.events / social.events）→ NotificationWorker 消费 → 写 `Notification` 表 + WebSocket 推送。

**新增表**: `Notification` — id, recipient_id, sender_id, type, target_id, is_read, created_at。

**新增接口**: 
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/notification/list` | 返回当前用户未读通知列表 |
| POST | `/notification/markRead` | 标记单条/全部已读 |
| GET | `/ws/notifications` | WebSocket 升级，实时推送 |

**前端**: AppShell 右上角通知铃铛 + 未读红点。

---

## P3 — 互动深化（3 项）

### 6. 私信 / 即时通讯

**新增表**: `Message` — id, from_id, to_id, content, is_read, created_at

**后端**: WebSocket 双向通道，`POST /message/send` + `POST /message/list`

### 7. #话题标签

**新增表**: `Tag` — id, name (unique)；`VideoTag` — video_id, tag_id

**改动**: 视频发布时从 title/description 中提取 `#xxx`，写入 `VideoTag` 关联表；`POST /feed/listByTag` 按话题浏览。

### 8. @提及

**改动**: 评论发布时解析 `@username`，创建 Notification 并推送。
