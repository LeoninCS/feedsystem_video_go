<script setup lang="ts">
import { computed, onMounted, reactive, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import AppShell from '../components/AppShell.vue'
import UserAvatar from '../components/UserAvatar.vue'
import { ApiError } from '../api/client'
import * as accountApi from '../api/account'
import * as socialApi from '../api/social'
import type { Account, Video } from '../api/types'
import * as videoApi from '../api/video'
import { useAuthStore } from '../stores/auth'
import { useSocialStore } from '../stores/social'
import { useToastStore } from '../stores/toast'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const social = useSocialStore()
const toast = useToastStore()

const userId = computed(() => Number(route.params.id))
const myId = computed(() => auth.claims?.account_id ?? 0)
const isMe = computed(() => myId.value > 0 && myId.value === userId.value)

const state = reactive({
  loading: false,
  error: '',
  user: null as Account | null,
  videos: [] as Video[],
  followers: [] as Account[],
  vloggers: [] as Account[],
  socialLoading: false,
  socialError: '',
})

const isFollowing = computed(() => (auth.isLoggedIn ? social.isFollowing(userId.value) : false))

async function loadProfile() {
  if (!Number.isFinite(userId.value) || userId.value <= 0) {
    state.error = '无效的用户 id'
    return
  }

  state.loading = true
  state.error = ''
  try {
    const [u, vids] = await Promise.all([accountApi.findById(userId.value), videoApi.listByAuthorId(userId.value)])
    state.user = u
    state.videos = vids
  } catch (e) {
    state.error = e instanceof ApiError ? e.message : String(e)
    state.user = null
    state.videos = []
  } finally {
    state.loading = false
  }

  await loadSocialCounts()
}

async function loadSocialCounts() {
  state.socialError = ''
  state.followers = []
  state.vloggers = []

  if (!auth.isLoggedIn) return
  if (!Number.isFinite(userId.value) || userId.value <= 0) return

  state.socialLoading = true
  try {
    const [followersRes, vloggersRes] = await Promise.all([
      socialApi.getAllFollowers(userId.value),
      socialApi.getAllVloggers(userId.value),
    ])
    state.followers = followersRes.followers
    state.vloggers = vloggersRes.vloggers
  } catch (e) {
    state.socialError = e instanceof ApiError ? e.message : String(e)
  } finally {
    state.socialLoading = false
  }
}

async function toggleFollow() {
  if (isMe.value) return
  if (!auth.isLoggedIn) {
    toast.error('请先登录')
    await router.push('/account')
    return
  }

  try {
    if (isFollowing.value) {
      await social.unfollow(userId.value)
      toast.info('已取关')
    } else {
      await social.follow(userId.value)
      toast.success('已关注')
    }
    await loadSocialCounts()
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : String(e)
    toast.error(msg)
  }
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
const listItems = computed(() => (drawer.tab === 'followers' ? state.followers : state.vloggers))

async function goUser(id: number) {
  drawer.open = false
  await router.push(`/u/${id}`)
}

async function goVideo(videoId: number) {
  await router.push(`/video/${videoId}`)
}

watch(
  () => route.params.id,
  async () => {
    drawer.open = false
    await loadProfile()
  },
)

watch(
  () => auth.isLoggedIn,
  async () => {
    await loadSocialCounts()
  },
)

onMounted(loadProfile)
</script>

<template>
  <AppShell>
    <div class="card">
      <div class="row" style="justify-content: space-between; align-items: flex-start">
        <div class="row" style="gap: 12px; align-items: center">
          <UserAvatar :username="state.user?.username ?? 'User'" :id="state.user?.id ?? userId" :size="64" />
          <div>
            <div class="title" style="margin: 0">@{{ state.user?.username ?? '-' }}</div>
            <div class="subtle mono">#{{ state.user?.id ?? userId }}</div>
          </div>
        </div>

        <div class="row">
          <button v-if="isMe" class="ghost" type="button" @click="router.push('/account')">我的账号</button>
          <button v-else class="primary" type="button" :disabled="!state.user || state.loading" @click="toggleFollow">
            {{ isFollowing ? '已关注' : '关注' }}
          </button>
        </div>
      </div>

      <div v-if="state.loading" class="hint" style="margin-top: 12px">加载中…</div>
      <div v-else-if="state.error" class="hint bad" style="margin-top: 12px">{{ state.error }}</div>

      <div v-else class="row" style="margin-top: 14px">
        <button class="metric" type="button" :disabled="!auth.isLoggedIn || state.socialLoading" @click="openFollowers">
          <div class="metric-num">{{ auth.isLoggedIn ? (state.socialLoading ? '…' : state.followers.length) : '—' }}</div>
          <div class="metric-label">粉丝</div>
        </button>
        <button class="metric" type="button" :disabled="!auth.isLoggedIn || state.socialLoading" @click="openFollowing">
          <div class="metric-num">{{ auth.isLoggedIn ? (state.socialLoading ? '…' : state.vloggers.length) : '—' }}</div>
          <div class="metric-label">关注</div>
        </button>
        <div class="metric static">
          <div class="metric-num">{{ state.videos.length }}</div>
          <div class="metric-label">作品</div>
        </div>
        <div v-if="!auth.isLoggedIn" class="subtle" style="margin-left: 8px">登录后可查看粉丝/关注列表</div>
        <div v-else-if="state.socialError" class="subtle" style="margin-left: 8px">社交信息加载失败：{{ state.socialError }}</div>
      </div>
    </div>

    <div class="card" style="margin-top: 14px">
      <div class="row" style="justify-content: space-between">
        <p class="title" style="margin: 0">作品</p>
        <div class="subtle">点击封面进入播放页</div>
      </div>

      <div v-if="state.videos.length === 0" class="hint" style="margin-top: 12px">暂无作品</div>

      <div v-else class="video-grid" style="margin-top: 12px">
        <button v-for="v in state.videos" :key="v.id" class="video-card" type="button" @click="goVideo(v.id)">
          <img class="video-cover" :src="v.cover_url" :alt="v.title" loading="lazy" />
          <div class="video-meta">
            <div class="video-title">{{ v.title }}</div>
            <div class="video-sub subtle">❤️ {{ v.likes_count }} · {{ new Date(v.create_time).toLocaleDateString() }}</div>
          </div>
        </button>
      </div>
    </div>

    <div v-if="drawer.open" class="drawer-backdrop" @click.self="closeDrawer">
      <div class="drawer">
        <div class="drawer-head">
          <div class="drawer-title">{{ listTitle }}</div>
          <button class="drawer-x" type="button" @click="closeDrawer">×</button>
        </div>
        <div class="drawer-body">
          <div v-if="state.socialLoading" class="drawer-hint">加载中…</div>
          <div v-else-if="state.socialError" class="drawer-hint bad">{{ state.socialError }}</div>
          <div v-else-if="listItems.length === 0" class="drawer-hint">暂无</div>

          <button v-for="u in listItems" :key="u.id" class="user-row" type="button" @click="goUser(u.id)">
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

.metric.static {
  cursor: default;
}

.metric:hover {
  background: rgba(255, 255, 255, 0.1);
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

