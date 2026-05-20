<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import AppShell from '../components/AppShell.vue'
import UserAvatar from '../components/UserAvatar.vue'
import { ApiError } from '../api/client'
import * as commentApi from '../api/comment'
import * as likeApi from '../api/like'
import type { Comment, Video } from '../api/types'
import * as videoApi from '../api/video'
import { useAuthStore } from '../stores/auth'
import { useSocialStore } from '../stores/social'
import { useToastStore } from '../stores/toast'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const social = useSocialStore()
const toast = useToastStore()

const id = computed(() => Number(route.params.id))

const state = reactive({
  loading: false,
  error: '',
  video: null as Video | null,
  isLiked: null as boolean | null,
  busy: false,
})

const muted = ref(true)
const videoEl = ref<HTMLVideoElement | null>(null)

const drawer = reactive({
  open: false,
  loading: false,
  error: '',
  comments: [] as Comment[],
  content: '',
})

async function needLogin() {
  toast.error('请先登录')
  await router.push('/account')
}

async function loadVideo() {
  if (!Number.isFinite(id.value) || id.value <= 0) {
    state.error = '无效的 video id'
    return
  }
  state.loading = true
  state.error = ''
  try {
    state.video = await videoApi.getDetail(id.value)
  } catch (e) {
    state.error = e instanceof ApiError ? e.message : String(e)
  } finally {
    state.loading = false
  }
}

async function loadIsLiked() {
  if (!auth.isLoggedIn) {
    state.isLiked = null
    return
  }
  try {
    const res = await likeApi.isLiked(id.value)
    state.isLiked = res.is_liked
  } catch {
    state.isLiked = null
  }
}

async function play() {
  if (!videoEl.value) return
  videoEl.value.muted = muted.value
  try {
    await videoEl.value.play()
  } catch {
    // ignore
  }
}

function toggleMute() {
  muted.value = !muted.value
  if (videoEl.value) videoEl.value.muted = muted.value
  toast.info(muted.value ? '已静音' : '已取消静音')
}

function togglePlayPause() {
  const v = videoEl.value
  if (!v) return
  if (v.paused) void v.play()
  else v.pause()
}

async function toggleLike() {
  if (!state.video) return
  if (!auth.isLoggedIn) return needLogin()
  if (state.busy) return

  state.busy = true
  try {
    if (state.isLiked) {
      await likeApi.unlike(id.value)
      state.isLiked = false
      state.video.likes_count = Math.max(0, state.video.likes_count - 1)
    } else {
      await likeApi.like(id.value)
      state.isLiked = true
      state.video.likes_count += 1
    }
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : String(e)
    toast.error(msg)
  } finally {
    state.busy = false
  }
}

async function toggleFollow() {
  if (!state.video) return
  if (!auth.isLoggedIn) return needLogin()
  if (state.busy) return
  if (auth.claims?.account_id && auth.claims.account_id === state.video.author_id) return

  state.busy = true
  try {
    if (social.isFollowing(state.video.author_id)) {
      await social.unfollow(state.video.author_id)
      toast.info('已取关')
    } else {
      await social.follow(state.video.author_id)
      toast.success('已关注')
    }
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : String(e)
    toast.error(msg)
  } finally {
    state.busy = false
  }
}

async function share() {
  if (!state.video) return
  const url = `${location.origin}/video/${state.video.id}`
  try {
    await navigator.clipboard.writeText(url)
    toast.success('链接已复制')
  } catch {
    window.prompt('复制链接', url)
  }
}

function closeDrawer() {
  drawer.open = false
  drawer.comments = []
  drawer.content = ''
  drawer.error = ''
}

async function loadComments() {
  if (!state.video) return
  drawer.loading = true
  drawer.error = ''
  try {
    drawer.comments = await commentApi.listAll(state.video.id)
  } catch (e) {
    drawer.error = e instanceof ApiError ? e.message : String(e)
  } finally {
    drawer.loading = false
  }
}

async function openComments() {
  drawer.open = true
  drawer.content = ''
  await loadComments()
}

async function publishComment() {
  if (!state.video) return
  if (!auth.isLoggedIn) return needLogin()
  const content = drawer.content.trim()
  if (!content) return

  drawer.loading = true
  drawer.error = ''
  try {
    await commentApi.publish(state.video.id, content)
    drawer.content = ''
    await loadComments()
    toast.success('评论已发布')
  } catch (e) {
    drawer.error = e instanceof ApiError ? e.message : String(e)
    toast.error(drawer.error)
  } finally {
    drawer.loading = false
  }
}

function canDeleteComment(c: Comment) {
  const myId = auth.claims?.account_id
  return !!myId && myId === c.author_id
}

async function deleteComment(commentId: number) {
  if (!state.video) return
  if (!auth.isLoggedIn) return needLogin()
  if (!window.confirm('确认删除这条评论？')) return

  drawer.loading = true
  drawer.error = ''
  try {
    await commentApi.remove(commentId)
    await loadComments()
    toast.info('评论已删除')
  } catch (e) {
    drawer.error = e instanceof ApiError ? e.message : String(e)
    toast.error(drawer.error)
  } finally {
    drawer.loading = false
  }
}

watch(
  () => id.value,
  async () => {
    closeDrawer()
    await loadVideo()
    await loadIsLiked()
    await nextTick()
    await play()
  },
)

watch(
  () => auth.isLoggedIn,
  async () => {
    await loadIsLiked()
  },
)

onMounted(async () => {
  await loadVideo()
  await loadIsLiked()
  await nextTick()
  await play()
})
</script>

<template>
  <AppShell full>
    <div class="page">
      <div class="top">
        <div class="top-left">
          <RouterLink class="chip" to="/">← 返回推荐</RouterLink>
        </div>
        <div class="top-right">
          <button class="chip" type="button" @click="toggleMute">{{ muted ? '静音' : '有声' }}</button>
        </div>
      </div>

      <div class="wrap">
        <div v-if="state.loading" class="center-hint">加载中…</div>
        <div v-else-if="state.error" class="center-hint bad">{{ state.error }}</div>

        <div v-else-if="state.video" class="stage" @click="togglePlayPause">
          <video
            ref="videoEl"
            class="video"
            :src="state.video.play_url"
            :poster="state.video.cover_url"
            playsinline
            preload="metadata"
            loop
          />
          <div class="grad" />

          <div class="meta">
            <RouterLink class="author-link" :to="`/u/${state.video.author_id}`" @click.stop>
              <UserAvatar :username="state.video.username" :id="state.video.author_id" :size="34" />
              <span class="author-name">@{{ state.video.username }}</span>
            </RouterLink>
            <div class="title">{{ state.video.title }}</div>
            <div v-if="state.video.description" class="desc">{{ state.video.description }}</div>
            <div class="row" style="margin-top: 10px">
              <a class="chip mono" :href="state.video.play_url" target="_blank" rel="noreferrer">play_url</a>
              <a class="chip mono" :href="state.video.cover_url" target="_blank" rel="noreferrer">cover_url</a>
            </div>
          </div>

          <div class="actions">
            <button class="act" type="button" :disabled="state.busy" @click.stop="toggleLike">
              <span class="icon" :class="{ liked: !!state.isLiked }">♥</span>
              <span class="count">{{ state.video.likes_count }}</span>
            </button>

            <button class="act" type="button" @click.stop="openComments">
              <span class="icon">💬</span>
              <span class="count">评论</span>
            </button>

            <button
              v-if="!auth.claims?.account_id || auth.claims.account_id !== state.video.author_id"
              class="act"
              type="button"
              :disabled="state.busy"
              @click.stop="toggleFollow"
            >
              <span class="icon">＋</span>
              <span class="count">{{ social.isFollowing(state.video.author_id) ? '已关注' : '关注' }}</span>
            </button>

            <button class="act" type="button" @click.stop="share">
              <span class="icon">↗</span>
              <span class="count">分享</span>
            </button>
          </div>

          <div class="hint">
            <span class="chip mono">点击 暂停/播放</span>
            <span class="chip mono">双击 点赞</span>
          </div>
        </div>
      </div>

      <div v-if="drawer.open" class="drawer-backdrop" @click.self="closeDrawer">
        <div class="drawer">
          <div class="drawer-head">
            <div class="drawer-title">评论</div>
            <button class="drawer-x" type="button" @click="closeDrawer">×</button>
          </div>

          <div class="drawer-body">
            <div v-if="drawer.loading" class="drawer-hint">加载中…</div>
            <div v-else-if="drawer.error" class="drawer-hint bad">{{ drawer.error }}</div>
            <div v-else-if="drawer.comments.length === 0" class="drawer-hint">暂无评论</div>

            <div class="comment" v-for="c in drawer.comments" :key="c.id">
              <div class="comment-top">
                <div class="comment-user">{{ c.username }}</div>
                <div class="comment-meta mono">
                  #{{ c.id }} · {{ new Date(c.created_at).toLocaleString() }}
                </div>
              </div>
              <div class="comment-content">{{ c.content }}</div>
              <div class="comment-actions">
                <button v-if="canDeleteComment(c)" class="chip danger" type="button" :disabled="drawer.loading" @click="deleteComment(c.id)">
                  删除
                </button>
              </div>
            </div>
          </div>

          <div class="drawer-foot">
            <textarea v-model="drawer.content" placeholder="说点什么…" :disabled="drawer.loading" />
            <div class="row" style="justify-content: space-between; margin-top: 8px">
              <button class="chip" type="button" :disabled="drawer.loading" @click="loadComments">刷新</button>
              <button class="chip primary" type="button" :disabled="drawer.loading || !drawer.content.trim()" @click="publishComment">
                发送
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
.page {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.top {
  height: 52px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 14px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(0, 0, 0, 0.25);
  backdrop-filter: blur(16px);
}

.wrap {
  flex: 1;
  min-height: 0;
  display: grid;
  place-items: center;
  padding: 18px 14px;
}

.center-hint {
  color: rgba(255, 255, 255, 0.78);
}

.center-hint.bad {
  color: rgba(254, 44, 85, 0.92);
}

.stage {
  width: min(980px, calc(100vw - 28px));
  height: calc(100vh - 56px - 52px - 36px);
  position: relative;
  border-radius: 18px;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(0, 0, 0, 0.35);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.55);
}

.video {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  background: rgba(0, 0, 0, 0.4);
}

.grad {
  position: absolute;
  inset: 0;
  background: linear-gradient(to top, rgba(0, 0, 0, 0.68), rgba(0, 0, 0, 0.12) 40%, rgba(0, 0, 0, 0) 70%);
  pointer-events: none;
}

.meta {
  position: absolute;
  left: 16px;
  bottom: 18px;
  max-width: min(620px, calc(100% - 96px));
}

.author-link {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  font-weight: 800;
  letter-spacing: 0.2px;
  margin-bottom: 6px;
  text-decoration: none;
}

.author-link:hover {
  text-decoration: none;
}

.author-name {
  text-shadow: 0 14px 30px rgba(0, 0, 0, 0.55);
}

.title {
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 6px;
}

.desc {
  color: rgba(255, 255, 255, 0.74);
  font-size: 13px;
  line-height: 1.35;
}

.actions {
  position: absolute;
  right: 12px;
  bottom: 18px;
  display: grid;
  gap: 12px;
}

.act {
  width: 70px;
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: rgba(0, 0, 0, 0.32);
  color: rgba(255, 255, 255, 0.92);
  padding: 10px 10px;
  cursor: pointer;
  display: grid;
  gap: 6px;
  justify-items: center;
}

.act:hover {
  background: rgba(255, 255, 255, 0.1);
}

.act:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.icon {
  font-size: 20px;
  line-height: 1;
  opacity: 0.92;
}

.icon.liked {
  color: rgba(254, 44, 85, 1);
  text-shadow: 0 10px 20px rgba(254, 44, 85, 0.25);
}

.count {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.8);
}

.hint {
  position: absolute;
  left: 14px;
  top: 14px;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 7px 10px;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: rgba(0, 0, 0, 0.28);
  color: rgba(255, 255, 255, 0.86);
  font-size: 12px;
  text-decoration: none;
}

.chip.primary {
  border-color: rgba(254, 44, 85, 0.45);
  background: rgba(254, 44, 85, 0.14);
}

.chip.danger {
  border-color: rgba(254, 44, 85, 0.55);
  background: rgba(254, 44, 85, 0.12);
}

.drawer-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(10px);
  z-index: 120;
  display: grid;
  justify-items: end;
}

.drawer {
  width: min(420px, calc(100vw - 18px));
  height: 100vh;
  background: rgba(0, 0, 0, 0.65);
  border-left: 1px solid rgba(255, 255, 255, 0.12);
  display: grid;
  grid-template-rows: auto 1fr auto;
}

.drawer-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 14px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.drawer-title {
  font-weight: 800;
  font-size: 14px;
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

.drawer-foot {
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  padding: 12px 14px;
}

.drawer-foot textarea {
  width: 100%;
  min-height: 82px;
  resize: none;
  border-radius: 14px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.9);
  padding: 10px 12px;
  outline: none;
}

.drawer-hint {
  color: rgba(255, 255, 255, 0.78);
  padding: 12px 0;
}

.drawer-hint.bad {
  color: rgba(254, 44, 85, 0.92);
}

.comment {
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.05);
  border-radius: 14px;
  padding: 10px 10px;
}

.comment-top {
  display: grid;
  gap: 3px;
}

.comment-user {
  font-weight: 700;
  font-size: 13px;
}

.comment-meta {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.55);
}

.comment-content {
  margin-top: 8px;
  font-size: 13px;
  line-height: 1.35;
  color: rgba(255, 255, 255, 0.86);
  white-space: pre-wrap;
  word-break: break-word;
}

.comment-actions {
  margin-top: 10px;
  display: flex;
  justify-content: flex-end;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
}

@media (max-width: 900px) {
  .stage {
    width: calc(100vw - 28px);
    height: calc(100vh - 56px - 52px - 36px);
  }
  .drawer-backdrop {
    justify-items: center;
    align-items: end;
  }
  .drawer {
    width: calc(100vw - 16px);
    height: min(72vh, 560px);
    border-left: none;
    border-top: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: 18px 18px 0 0;
    overflow: hidden;
  }
}
</style>
