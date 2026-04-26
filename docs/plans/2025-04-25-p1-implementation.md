# P1 用户基石 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use executing-plans to implement this plan task-by-task.

**Goal:** 扩展 Account 模型（头像+简介）、实现双 Token 登录态优化（Access + Refresh Token）

**Architecture:** Account 模型加 avatar_url/bio/refresh_token 字段；复用 UploadCover 的文件上传逻辑做头像；JWT 双 Token 机制 — Access 15min / Refresh 7天

**Tech Stack:** Go + Gin + GORM + JWT + Vue 3 + Pinia

---

### Task 1: Account 模型扩展

**Files:**
- Modify: `backend/internal/account/entity.go:3-8`

**Step 1: 修改 Account struct**

```go
type Account struct {
    ID           uint   `gorm:"primaryKey" json:"id"`
    Username     string `gorm:"unique" json:"username"`
    Password     string `json:"-"`
    Token        string `json:"-"`
    RefreshToken string `json:"-"`
    AvatarURL    string `gorm:"type:varchar(512)" json:"avatar_url,omitempty"`
    Bio          string `gorm:"type:varchar(255)" json:"bio,omitempty"`
}
```

**Step 2: 编译验证**

Run: `go build ./...`
Expected: 编译通过（AutoMigrate 自动加列）

**Step 3: Commit**

```bash
git add backend/internal/account/entity.go
git commit -m "feat: Account 模型加 avatar_url/bio/refresh_token 字段"
```

---

### Task 2: 头像上传 Handler

**Files:**
- Modify: `backend/internal/account/handler.go` — 新增 UploadAvatar 方法
- Modify: `backend/internal/http/router.go` — 注册路由

**Step 1: 添加 UploadAvatar handler**

参考 `video/video_handler.go` 的 `UploadCover`，在 `account/handler.go` 中新增：

```go
func (ah *AccountHandler) UploadAvatar(c *gin.Context) {
    accountID, err := jwt.GetAccountID(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
        return
    }
    f, err := c.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
        return
    }
    const maxSize = 10 << 20
    if f.Size <= 0 || f.Size > maxSize {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file size"})
        return
    }
    ext := strings.ToLower(filepath.Ext(f.Filename))
    switch ext {
    case ".jpg", ".jpeg", ".png", ".webp":
    default:
        c.JSON(http.StatusBadRequest, gin.H{"error": "only .jpg/.jpeg/.png/.webp allowed"})
        return
    }
    dir := filepath.Join(".run", "uploads", "avatars", strconv.FormatUint(uint64(accountID), 10))
    if err := os.MkdirAll(dir, 0o755); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    filename, err := randHex(16)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    filename = filename + ext
    absPath := filepath.Join(dir, filename)
    if err := c.SaveUploadedFile(f, absPath); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    urlPath := path.Join("/static", "avatars", strconv.FormatUint(uint64(accountID), 10), filename)
    avatarURL := buildAbsoluteURL(c, urlPath)

    // 更新数据库
    if err := ah.accountService.UpdateAvatar(c.Request.Context(), accountID, avatarURL); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"avatar_url": avatarURL})
}
```

需要新增 import: `"os"`, `"path"`, `"path/filepath"`, `"crypto/rand"`, `"encoding/hex"`, `"strconv"`, `"strings"`, `"net/http"` — 但 account/handler.go 已有部分，按需补。

同时需要从 `video_handler.go` 复制 `randHex` 和 `buildAbsoluteURL` 函数（或提取到公共 util）。

**Step 2: 在 router.go 注册路由**

```go
protectedAccountGroup.POST("/uploadAvatar", accountHandler.UploadAvatar)
```

**Step 3: 添加 AccountService.UpdateAvatar 方法**

```go
func (as *AccountService) UpdateAvatar(ctx context.Context, accountID uint, avatarURL string) error {
    return as.accountRepo.UpdateAvatar(ctx, accountID, avatarURL)
}
```

**Step 4: 添加 AccountRepository.UpdateAvatar 方法**

```go
func (ar *AccountRepository) UpdateAvatar(ctx context.Context, accountID uint, avatarURL string) error {
    return ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", accountID).Update("avatar_url", avatarURL).Error
}
```

**Step 5: 编译验证**

Run: `go build ./...`
Expected: 通过

**Step 6: Commit**

```bash
git add backend/internal/account/handler.go backend/internal/account/service.go backend/internal/account/repo.go backend/internal/http/router.go
git commit -m "feat: 头像上传接口 /account/uploadAvatar"
```

---

### Task 3: 更新个人简介接口

**Files:**
- Modify: `backend/internal/account/handler.go` — 新增 UpdateProfile
- Modify: `backend/internal/http/router.go` — 注册路由

**Step 1: 新增 request struct + handler**

在 `entity.go` 加：
```go
type UpdateProfileRequest struct {
    AvatarURL string `json:"avatar_url"`
    Bio       string `json:"bio"`
}
```

Handler:
```go
func (ah *AccountHandler) UpdateProfile(c *gin.Context) {
    accountID, err := jwt.GetAccountID(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
        return
    }
    var req UpdateProfileRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if err := ah.accountService.UpdateProfile(c.Request.Context(), accountID, &req); err != nil {
        c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}
```

**Step 2: Service + Repo 层**

```go
func (as *AccountService) UpdateProfile(ctx context.Context, accountID uint, req *UpdateProfileRequest) error {
    updates := map[string]interface{}{}
    if req.Bio != "" {
        updates["bio"] = strings.TrimSpace(req.Bio)
    }
    if req.AvatarURL != "" {
        updates["avatar_url"] = strings.TrimSpace(req.AvatarURL)
    }
    if len(updates) == 0 {
        return errors.New("nothing to update")
    }
    return as.accountRepo.UpdateFields(ctx, accountID, updates)
}
```

**Step 3: 注册路由**

```go
protectedAccountGroup.POST("/updateProfile", accountHandler.UpdateProfile)
```

**Step 4: 编译 + 提交**

Run: `go build ./...`
Expected: 通过

```bash
git add backend/internal/account/ && git add backend/internal/http/router.go
git commit -m "feat: 个人简介更新接口 /account/updateProfile"
```

---

### Task 4: Refresh Token 机制

**Files:**
- Modify: `backend/internal/auth/jwt.go` — 新增 GenerateRefreshToken + ValidateRefreshToken
- Modify: `backend/internal/account/handler.go` — 新增 Refresh handler
- Modify: `backend/internal/account/service.go` — Login 返回双 token
- Modify: `backend/internal/http/router.go` — 注册 refresh 路由

**Step 1: auth/jwt.go 增加 Refresh Token**

```go
const (
    AccessTokenTTL  = 15 * time.Minute
    RefreshTokenTTL = 7 * 24 * time.Hour
)

func GenerateAccessToken(accountID uint, username string) (string, error) {
    // 原 GenerateToken 逻辑，TTL 改为 15min
}

func GenerateRefreshToken(accountID uint) (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}
```

**Step 2: Login 返回双 token**

修改 `account/service.go` 的 `Login` 方法，返回值从 `(string, error)` 改为 `(accessToken, refreshToken string, err error)`，并更新 `entity.go` 中的 `LoginResponse`。

```go
type LoginResponse struct {
    Token        string `json:"token"`         // access token
    RefreshToken string `json:"refresh_token"` // refresh token
    AccountID    uint   `json:"account_id"`
    Username     string `json:"username"`
}
```

Login 时生成两个 token，access token 落库 `account.token`，refresh token 落库 `account.refresh_token`，两者都缓存到 Redis。

**Step 3: Refresh handler**

新增 `POST /account/refresh`：

```go
type RefreshRequest struct {
    RefreshToken string `json:"refresh_token"`
}

func (ah *AccountHandler) Refresh(c *gin.Context) {
    var req RefreshRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    newAccessToken, err := ah.accountService.RefreshAccessToken(c.Request.Context(), req.RefreshToken)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"token": newAccessToken})
}
```

`AccountService.RefreshAccessToken`：查 Redis `account:{id}:refresh` → 匹配 → 生成新 access token → 更新 token 字段。

**Step 4: 登出/改密时同时清空 refresh_token**

在 `Logout` 和 `ChangePassword` 中增加 `Del("account:{id}:refresh")`。

**Step 5: 编译 + 提交**

```bash
git add backend/internal/auth/jwt.go backend/internal/account/
git commit -m "feat: Refresh Token 机制 — Access 15min + Refresh 7天"
```

---

### Task 5: 前端 auth store + client.ts 适配双 Token

**Files:**
- Modify: `frontend/src/stores/auth.ts`
- Modify: `frontend/src/api/client.ts`
- Modify: `frontend/src/api/account.ts`

**Step 1: auth store 存储双 token**

```typescript
const ACCESS_KEY = 'access_token'
const REFRESH_KEY = 'refresh_token'

// 新增字段
const refreshToken = ref<string | null>(readToken(REFRESH_KEY))

function setTokens(access: string, refresh: string) {
    token.value = access; refreshToken.value = refresh
    writeToken(ACCESS_KEY, access); writeToken(REFRESH_KEY, refresh)
}

function clearTokens() {
    token.value = null; refreshToken.value = null
    removeToken(ACCESS_KEY); removeToken(REFRESH_KEY)
}
```

**Step 2: client.ts 401 自动刷新**

```typescript
async function tryRefresh(): Promise<string | null> {
    const auth = useAuthStore()
    if (!auth.refreshToken) return null
    try {
        const res = await postJson<{ token: string }>('/account/refresh', { refresh_token: auth.refreshToken })
        auth.setToken(res.token)
        return res.token
    } catch {
        auth.clearTokens()
        return null
    }
}
```

在 `postJson` 和 `postForm` 的 `!res.ok` 分支中，401 时先尝试刷新，成功则重试原请求。

**Step 3: 编译验证**

Run: `npm run build`
Expected: 通过

**Step 4: Commit**

```bash
git add frontend/src/stores/auth.ts frontend/src/api/client.ts frontend/src/api/account.ts
git commit -m "feat: 前端双 Token 适配 — 401 自动刷新 + Refresh Token 存储"
```

---

### Task 6: 前端用户 Profile UI

**Files:**
- Modify: `frontend/src/views/AccountView.vue`
- Modify: `frontend/src/components/UserAvatar.vue`
- Modify: `frontend/src/views/HomeView.vue` — Feed 卡片中 UserAvatar 传递头像 URL
- Modify: `frontend/src/api/account.ts` — 新增 API 调用

**Step 1: UserAvatar 支持 src**

```vue
<script setup>
defineProps<{ username: string; id: number; size?: number; src?: string }>()
</script>
<template>
  <img v-if="src" :src="src" :width="size" :height="size" class="avatar-img" />
  <svg v-else ...> <!-- 默认 SVG -->
</template>
```

**Step 2: AccountView 加头像上传 + bio 编辑**

在登录后的 AccountView 中增加：头像上传按钮（调用 `/account/uploadAvatar`）、bio 编辑输入框（调用 `/account/updateProfile`）。

**Step 3: 编译验证**

Run: `npm run build`
Expected: 通过

**Step 4: Commit**

```bash
git add frontend/src/components/UserAvatar.vue frontend/src/views/AccountView.vue frontend/src/views/HomeView.vue frontend/src/api/account.ts
git commit -m "feat: 前端用户 Profile UI — 头像上传 + bio 编辑 + 登录记住我"
```

---

## 验证清单

完成所有 Task 后：

```bash
cd backend && go build ./... && go vet ./... && go test ./...
cd frontend && npm run build
```

Expected: 全部通过
