<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'

import AppShell from '../components/AppShell.vue'
import UserAvatar from '../components/UserAvatar.vue'
import { ApiError } from '../api/client'
import * as accountApi from '../api/account'
import type { Video } from '../api/types'
import * as videoApi from '../api/video'
import { useAuthStore } from '../stores/auth'
import { useSocialStore } from '../stores/social'
import { useToastStore } from '../stores/toast'

const router = useRouter()
const auth = useAuthStore()
const social = useSocialStore()
const toast = useToastStore()

const busy = ref(false)
const loginForm = reactive({ username: '', password: '' })

const me = computed(() => ({
  id: auth.claims?.account_id ?? 0,
  username: auth.claims?.username ?? '',
}))

const myVideos = reactive({
  loading: false,
  error: '',
  items: [] as Video[],
})

let myVideosReq = 0
async function loadMyVideos() {
  const id = me.value.id
  if (!auth.isLoggedIn || !id) {
    myVideos.items = []
    myVideos.error = ''
    myVideos.loading = false
    return
  }
  if (myVideos.loading) return

  const req = ++myVideosReq
  myVideos.loading = true
  myVideos.error = ''
  try {
    const vids = await videoApi.listByAuthorId(id)
    if (req !== myVideosReq) return
    myVideos.items = vids
  } catch (e) {
    if (req !== myVideosReq) return
    myVideos.error = e instanceof ApiError ? e.message : String(e)
    myVideos.items = []
  } finally {
    if (req === myVideosReq) myVideos.loading = false
  }
}

async function goVideo(id: number) {
  await router.push(`/video/${id}`)
}

async function onLogin() {
  if (busy.value) return
  const username = loginForm.username.trim()
  const password = loginForm.password.trim()
  if (!username || !password) {
    toast.error('请输入用户名和密码')
    return
  }

  busy.value = true
  try {
    const res = await accountApi.login(username, password)
    auth.setToken(res.token)
    toast.success('登录成功')
    await social.refreshMine()
    await loadMyVideos()
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : String(e)
    toast.error(msg)
  } finally {
    busy.value = false
  }
}

async function goRegister() {
  await router.push('/account/register')
}

async function goChangePassword() {
  await router.push('/account/change-password')
}

async function goSettings() {
  await router.push('/settings')
}

type ListTab = 'followers' | 'following'
const drawer = reactive({
  open: false,
  tab: 'followers' as ListTab,
})

function openFollowers() {
  drawer.tab = 'followers'
  drawer.open = true
}

function openFollowing() {
  drawer.tab = 'following'
  drawer.open = true
}

function closeDrawer() {
  drawer.open = false
}

const listTitle = computed(() => (drawer.tab === 'followers' ? '粉丝' : '关注'))
const listItems = computed(() => (drawer.tab === 'followers' ? social.followers : social.vloggers))
const drawerLoading = computed(() => (drawer.tab === 'followers' ? social.followersLoading : social.vloggersLoading))
const drawerError = computed(() => (drawer.tab === 'followers' ? social.followersError : social.vloggersError))
const socialErrorHint = computed(() => social.followersError || social.vloggersError)

async function goUser(id: number) {
  drawer.open = false
  await router.push(`/u/${id}`)
}

watch(
  () => auth.isLoggedIn,
  (v) => {
    if (!v) {
      drawer.open = false
      myVideosReq += 1
      myVideos.items = []
      myVideos.error = ''
    }
  },
)

watch(
  () => me.value.id,
  (id) => {
    if (auth.isLoggedIn && id) void loadMyVideos()
  },
  { immediate: true },
)
</script>

<template>
  <AppShell>
    <div v-if="!auth.isLoggedIn" class="login-wrap">
      <div class="card login-card">
        <p class="title">登录</p>
        <div class="grid" style="margin-top: 10px">
          <div>
            <label>username</label>
            <input v-model.trim="loginForm.username" autocomplete="username" />
          </div>
          <div>
            <label>password</label>
            <input v-model.trim="loginForm.password" type="password" autocomplete="current-password" @keydown.enter="onLogin" />
          </div>
          <button class="primary" type="button" :disabled="busy" @click="onLogin">登录</button>
        </div>

        <div class="row" style="justify-content: space-between; margin-top: 14px">
          <button class="ghost" type="button" :disabled="busy" @click="goRegister">注册账号</button>
          <button class="ghost" type="button" :disabled="busy" @click="goChangePassword">修改密码</button>
        </div>
      </div>
    </div>

    <template v-else>
      <div class="card">
        <div class="row" style="justify-content: space-between; align-items: flex-start">
          <div class="row" style="gap: 12px; align-items: center">
            <UserAvatar :username="me.username" :id="me.id" :size="64" />
            <div>
              <div class="title" style="margin: 0">@{{ me.username }}</div>
              <div class="subtle mono">#{{ me.id }}</div>
            </div>
          </div>

          <div class="row">
            <button class="ghost" type="button" @click="goSettings">设置</button>
          </div>
        </div>

        <div class="row" style="margin-top: 14px">
          <button class="metric" type="button" :disabled="social.followersLoading" @click="openFollowers">
            <div class="metric-num">{{ social.followersLoading ? '…' : social.followerCount }}</div>
            <div class="metric-label">粉丝</div>
          </button>
          <button class="metric" type="button" :disabled="social.vloggersLoading" @click="openFollowing">
            <div class="metric-num">{{ social.vloggersLoading ? '…' : social.followingCount }}</div>
            <div class="metric-label">关注</div>
          </button>
          <div class="metric static">
            <div class="metric-num">{{ myVideos.loading ? '…' : myVideos.items.length }}</div>
            <div class="metric-label">作品</div>
          </div>
          <div v-if="socialErrorHint" class="subtle" style="margin-left: 8px">社交信息加载失败：{{ socialErrorHint }}</div>
        </div>
      </div>

      <div class="card" style="margin-top: 14px">
        <div class="row" style="justify-content: space-between">
          <p class="title" style="margin: 0">作品</p>
          <div class="subtle">点击封面进入播放页</div>
        </div>

        <div v-if="myVideos.loading" class="hint" style="margin-top: 12px">加载中…</div>
        <div v-else-if="myVideos.error" class="hint bad" style="margin-top: 12px">{{ myVideos.error }}</div>
        <div v-else-if="myVideos.items.length === 0" class="hint" style="margin-top: 12px">暂无作品</div>

        <div v-else class="video-grid" style="margin-top: 12px">
          <button v-for="v in myVideos.items" :key="v.id" class="video-card" type="button" @click="goVideo(v.id)">
            <img class="video-cover" :src="v.cover_url" :alt="v.title" loading="lazy" />
            <div class="video-meta">
              <div class="video-title">{{ v.title }}</div>
              <div class="video-sub subtle">❤️ {{ v.likes_count }} · {{ new Date(v.create_time).toLocaleDateString() }}</div>
            </div>
          </button>
        </div>
      </div>
    </template>

    <div v-if="drawer.open" class="drawer-backdrop" @click.self="closeDrawer">
      <div class="drawer">
        <div class="drawer-head">
          <div class="drawer-title">{{ listTitle }}</div>
          <button class="drawer-x" type="button" @click="closeDrawer">×</button>
        </div>
        <div class="drawer-body">
          <div v-if="drawerLoading" class="drawer-hint">加载中…</div>
          <div v-else-if="drawerError" class="drawer-hint bad">{{ drawerError }}</div>
          <div v-else-if="listItems.length === 0" class="drawer-hint">暂无</div>

          <button v-for="u in listItems" v-if="!drawerLoading && !drawerError" :key="u.id" class="user-row" type="button" @click="goUser(u.id)">
            <UserAvatar :username="u.username" :id="u.id" :size="40" />
            <div class="user-meta">
              <div class="user-name">@{{ u.username }}</div>
              <div class="user-id mono">#{{ u.id }}</div>
            </div>
          </button>
        </div>
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
.login-wrap {
  min-height: calc(100vh - 56px);
  display: grid;
  place-items: center;
  padding: 18px 0 40px;
}

.login-card {
  width: min(420px, 100%);
}

.ghost {
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: rgba(0, 0, 0, 0.18);
  color: rgba(255, 255, 255, 0.86);
  border-radius: 12px;
  padding: 10px 12px;
  cursor: pointer;
}

.ghost:hover {
  background: rgba(255, 255, 255, 0.1);
}

.metric {
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.06);
  border-radius: 16px;
  padding: 12px 14px;
  min-width: 120px;
  cursor: pointer;
  display: grid;
  gap: 4px;
  text-align: left;
}

.metric:hover {
  background: rgba(255, 255, 255, 0.1);
}

.metric.static {
  cursor: default;
}

.metric:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.metric-num {
  font-size: 18px;
  font-weight: 900;
  letter-spacing: 0.2px;
}

.metric-label {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.65);
}

.hint {
  color: rgba(255, 255, 255, 0.78);
}

.hint.bad {
  color: rgba(254, 44, 85, 0.92);
}

.drawer-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(10px);
  z-index: 120;
  display: grid;
  justify-items: center;
  align-items: center;
  padding: 16px;
}

.drawer {
  width: min(520px, calc(100vw - 18px));
  max-height: min(78vh, 720px);
  background: rgba(0, 0, 0, 0.65);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 18px;
  overflow: hidden;
  display: grid;
  grid-template-rows: auto 1fr;
}

.drawer-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 14px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.drawer-title {
  font-weight: 900;
}

.drawer-x {
  width: 34px;
  height: 34px;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.9);
  cursor: pointer;
  font-size: 20px;
  line-height: 1;
}

.drawer-body {
  overflow: auto;
  padding: 12px 14px;
  display: grid;
  gap: 10px;
}

.drawer-hint {
  color: rgba(255, 255, 255, 0.78);
  padding: 12px 0;
}

.drawer-hint.bad {
  color: rgba(254, 44, 85, 0.92);
}

.video-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

@media (max-width: 1100px) {
  .video-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 800px) {
  .video-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

.video-card {
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.05);
  border-radius: 16px;
  overflow: hidden;
  cursor: pointer;
  padding: 0;
  text-align: left;
}

.video-card:hover {
  background: rgba(255, 255, 255, 0.08);
}

.video-cover {
  width: 100%;
  aspect-ratio: 9/12;
  object-fit: cover;
  display: block;
  background: rgba(0, 0, 0, 0.35);
}

.video-meta {
  padding: 10px 10px;
}

.video-title {
  font-weight: 800;
  font-size: 13px;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.video-sub {
  margin-top: 6px;
  font-size: 12px;
}

.user-row {
  text-align: left;
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 12px;
  align-items: center;
  padding: 10px 10px;
  border-radius: 14px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.05);
  cursor: pointer;
}

.user-row:hover {
  background: rgba(255, 255, 255, 0.08);
}

.user-meta {
  min-width: 0;
}

.user-name {
  font-weight: 800;
}

.user-id {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.6);
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
}
</style>
