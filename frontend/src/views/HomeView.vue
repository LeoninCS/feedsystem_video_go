<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import AppShell from '../components/AppShell.vue'
import CommentDrawer from '../components/CommentDrawer.vue'
import UserAvatar from '../components/UserAvatar.vue'
import type { FeedVideoItem } from '../api/types'
import { useAuthStore } from '../stores/auth'
import { useSocialStore } from '../stores/social'
import { useToastStore } from '../stores/toast'
import { useVideoFeed } from '../composables/useVideoFeed'
import { useVideoPlayer } from '../composables/useVideoPlayer'
import { useLikeFollow } from '../composables/useLikeFollow'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const social = useSocialStore()
const toast = useToastStore()

const { tab, following, currentState, loadFollowing, ensureTabLoaded, loadMoreIfNeeded } = useVideoFeed()
const scroller = ref<HTMLDivElement | null>(null)
const { muted, activeIndex, videoMap, setVideoRef, scrollToIndex, onScroll, playActive, toggleMute, togglePlayPause } = useVideoPlayer(scroller)

async function needLogin() {
  toast.error('请先登录')
  await router.push('/account')
}

const { likeBusy, followBusy, toggleLike, toggleFollow, share } = useLikeFollow(needLogin)

const drawerVideo = ref<FeedVideoItem | null>(null)
const drawerOpen = ref(false)

function openComments(item: FeedVideoItem) { drawerVideo.value = item; drawerOpen.value = true }
function closeDrawer() { drawerOpen.value = false; drawerVideo.value = null }

const q = computed(() => (typeof route.query.q === 'string' ? route.query.q.trim().toLowerCase() : ''))
const filteredItems = computed(() => {
  const items = currentState.value.items
  if (!q.value) return items
  return items.filter((v) => v.title.toLowerCase().includes(q.value) || v.author.username.toLowerCase().includes(q.value))
})
const activeItem = computed(() => filteredItems.value[activeIndex.value] ?? null)
const visibleRange = computed(() => {
  const idx = activeIndex.value
  const len = filteredItems.value.length
  return { start: Math.max(0, idx - 1), end: Math.min(len - 1, idx + 1) }
})
const myAccountId = computed(() => auth.claims?.account_id ?? 0)

watch(activeItem, async () => {
  await nextTick()
  await playActive(activeItem.value?.id)
  await loadMoreIfNeeded(activeIndex.value)
})

watch(() => tab.value, async () => {
  activeIndex.value = 0
  videoMap.clear()
  if (scroller.value) scroller.value.scrollTop = 0
  await ensureTabLoaded()
  await nextTick()
  await playActive(activeItem.value?.id)
})

watch(() => q.value, async () => {
  activeIndex.value = 0
  if (scroller.value) scroller.value.scrollTop = 0
  await nextTick()
  await playActive(activeItem.value?.id)
})

watch(() => filteredItems.value.length, (len) => {
  if (len === 0) activeIndex.value = 0
  else if (activeIndex.value > len - 1) activeIndex.value = len - 1
})

watch(() => auth.isLoggedIn, async (v) => {
  if (tab.value === 'following' && v && following.items.length === 0) await loadFollowing(true)
})

async function onKeydown(e: KeyboardEvent) {
  const t = e.target as HTMLElement | null
  if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA')) return
  if (drawerOpen.value) return
  if (e.key === 'ArrowDown') { e.preventDefault(); scrollToIndex(activeIndex.value + 1, filteredItems.value.length) }
  else if (e.key === 'ArrowUp') { e.preventDefault(); scrollToIndex(activeIndex.value - 1, filteredItems.value.length) }
  else if (e.key === ' ') { e.preventDefault(); togglePlayPause(activeItem.value?.id) }
  else if (e.key.toLowerCase() === 'm') { e.preventDefault(); toggleMute() }
  else if (e.key.toLowerCase() === 'c') { if (activeItem.value) { e.preventDefault(); openComments(activeItem.value) } }
}

onMounted(async () => {
  await ensureTabLoaded()
  await nextTick()
  await playActive(activeItem.value?.id)
  window.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <AppShell full>
    <div class="page">
      <div class="tabs">
        <button class="tab" :class="{ on: tab === 'recommend' }" type="button" @click="tab = 'recommend'">推荐</button>
        <button class="tab" :class="{ on: tab === 'following' }" type="button" @click="tab = 'following'">关注</button>
        <button class="tab" :class="{ on: tab === 'hot' }" type="button" @click="tab = 'hot'">点赞榜</button>
        <div class="tabs-right">
          <button class="chip" type="button" @click="toggleMute">{{ muted ? '静音' : '有声' }}</button>
          <RouterLink class="chip" :to="activeItem ? `/video/${activeItem.id}` : '/video'">详情</RouterLink>
        </div>
      </div>

      <div ref="scroller" class="scroller" @scroll="onScroll">
        <div v-if="currentState.loading && currentState.items.length === 0" class="center-hint">加载中…</div>
        <div v-else-if="currentState.error && currentState.items.length === 0" class="center-hint bad">{{ currentState.error }}</div>
        <div v-else-if="filteredItems.length === 0" class="center-hint">没有匹配内容</div>

        <section
          v-for="(item, idx) in filteredItems"
          :key="`${tab}-${item.id}`"
          v-show="idx >= visibleRange.start && idx <= visibleRange.end"
          class="slide"
          :class="{ active: idx === activeIndex }"
        >
          <div class="stage" @click="togglePlayPause(activeItem?.id)" @dblclick.prevent="toggleLike(item)">
            <video
              class="video"
              :ref="(el) => setVideoRef(item.id, el as HTMLVideoElement | null)"
              :src="item.play_url"
              :poster="item.cover_url"
              playsinline preload="metadata" loop
            />
            <div class="grad" />
            <div class="meta">
              <RouterLink class="author-link" :to="`/u/${item.author.id}`" @click.stop>
                <UserAvatar :username="item.author.username" :id="item.author.id" :size="34" />
                <span class="author-name">@{{ item.author.username }}</span>
              </RouterLink>
              <div class="title">{{ item.title }}</div>
              <div v-if="item.description" class="desc">{{ item.description }}</div>
            </div>
            <div class="actions">
              <button class="act" type="button" :disabled="!!likeBusy[String(item.id)]" @click.stop="toggleLike(item)">
                <span class="icon" :class="{ liked: item.is_liked }">♥</span>
                <span class="count">{{ item.likes_count }}</span>
              </button>
              <button class="act" type="button" @click.stop="openComments(item)">
                <span class="icon">💬</span>
                <span class="count">评论</span>
              </button>
              <button
                v-if="!myAccountId || myAccountId !== item.author.id"
                class="act" type="button"
                :disabled="!!followBusy[String(item.author.id)]"
                @click.stop="toggleFollow(item.author.id)">
                <span class="icon">＋</span>
                <span class="count">{{ social.isFollowing(item.author.id) ? '已关注' : '关注' }}</span>
              </button>
              <button class="act" type="button" @click.stop="share(item)">
                <span class="icon">↗</span>
                <span class="count">分享</span>
              </button>
            </div>
            <div class="hint">
              <span class="chip mono">↑ ↓ 切换</span>
              <span class="chip mono">空格 暂停</span>
              <span class="chip mono">M 静音</span>
              <span class="chip mono">C 评论</span>
            </div>
          </div>
        </section>
      </div>

      <CommentDrawer v-if="drawerOpen" :video="drawerVideo" @close="closeDrawer" />
    </div>
  </AppShell>
</template>

<style scoped>
.page { height: 100%; display: flex; flex-direction: column; }
.tabs { height: 52px; display: flex; align-items: center; gap: 10px; padding: 0 14px; border-bottom: 1px solid rgba(255,255,255,0.08); background: rgba(0,0,0,0.25); backdrop-filter: blur(16px); }
.tab { border: 1px solid rgba(255,255,255,0.14); background: rgba(255,255,255,0.06); color: rgba(255,255,255,0.88); border-radius: 999px; padding: 8px 14px; cursor: pointer; }
.tab.on { border-color: rgba(254,44,85,0.5); background: rgba(254,44,85,0.16); }
.tabs-right { margin-left: auto; display: flex; gap: 10px; align-items: center; }
.scroller { flex: 1; min-height: 0; overflow-y: auto; scroll-snap-type: y mandatory; scroll-behavior: smooth; scrollbar-width: none; -ms-overflow-style: none; }
.scroller::-webkit-scrollbar { width: 0; height: 0; }
.center-hint { height: calc(100% - 60px); display: grid; place-items: center; color: rgba(255,255,255,0.78); }
.center-hint.bad { color: rgba(254,44,85,0.92); }
.slide { height: 100%; box-sizing: border-box; scroll-snap-align: start; padding: 18px 14px; display: grid; place-items: center; }
.stage { width: min(980px, calc(100vw - 28px)); height: calc(100vh - 56px - 52px - 36px); position: relative; border-radius: 18px; overflow: hidden; border: 1px solid rgba(255,255,255,0.12); background: rgba(0,0,0,0.35); box-shadow: 0 20px 60px rgba(0,0,0,0.55); }
.video { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: cover; background: rgba(0,0,0,0.4); }
.grad { position: absolute; inset: 0; background: linear-gradient(to top, rgba(0,0,0,0.68), rgba(0,0,0,0.12) 40%, rgba(0,0,0,0) 70%); pointer-events: none; }
.meta { position: absolute; left: 16px; bottom: 18px; max-width: min(620px, calc(100% - 96px)); }
.author-link { display: inline-flex; align-items: center; gap: 10px; font-weight: 800; letter-spacing: 0.2px; margin-bottom: 6px; text-decoration: none; }
.author-link:hover { text-decoration: none; }
.author-name { text-shadow: 0 14px 30px rgba(0,0,0,0.55); }
.title { font-size: 16px; font-weight: 700; margin-bottom: 6px; }
.desc { color: rgba(255,255,255,0.74); font-size: 13px; line-height: 1.35; }
.actions { position: absolute; right: 12px; bottom: 18px; display: grid; gap: 12px; }
.act { width: 70px; border-radius: 16px; border: 1px solid rgba(255,255,255,0.14); background: rgba(0,0,0,0.32); color: rgba(255,255,255,0.92); padding: 10px 10px; cursor: pointer; display: grid; gap: 6px; justify-items: center; }
.act:hover { background: rgba(255,255,255,0.1); }
.act:disabled { opacity: 0.55; cursor: not-allowed; }
.icon { font-size: 20px; line-height: 1; opacity: 0.92; }
.icon.liked { color: rgba(254,44,85,1); text-shadow: 0 10px 20px rgba(254,44,85,0.25); }
.count { font-size: 12px; color: rgba(255,255,255,0.8); }
.hint { position: absolute; left: 14px; top: 14px; display: flex; gap: 8px; flex-wrap: wrap; }
.chip { display: inline-flex; align-items: center; gap: 8px; padding: 7px 10px; border-radius: 999px; border: 1px solid rgba(255,255,255,0.14); background: rgba(0,0,0,0.28); color: rgba(255,255,255,0.86); font-size: 12px; text-decoration: none; }
.chip.primary { border-color: rgba(254,44,85,0.45); background: rgba(254,44,85,0.14); }
.chip.danger { border-color: rgba(254,44,85,0.55); background: rgba(254,44,85,0.12); }
</style>
