<script setup lang="ts">
import { computed, onMounted, reactive, watch } from 'vue'

import AppShell from '../components/AppShell.vue'
import JsonBox from '../components/JsonBox.vue'
import FeedVideoCard from '../components/FeedVideoCard.vue'
import { ApiError } from '../api/client'
import * as feedApi from '../api/feed'
import * as likeApi from '../api/like'
import type { FeedVideoItem } from '../api/types'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()

type ListState = {
  loading: boolean
  error: string
  items: FeedVideoItem[]
  has_more: boolean
}

const latest = reactive<ListState & { limit: number; next_time: number }>({
  loading: false,
  error: '',
  items: [],
  has_more: false,
  limit: 10,
  next_time: 0,
})

const likesCount = reactive<ListState & { limit: number; next_likes_count_before?: number; next_id_before?: number }>({
  loading: false,
  error: '',
  items: [],
  has_more: false,
  limit: 10,
  next_likes_count_before: undefined,
  next_id_before: undefined,
})

const following = reactive<ListState & { limit: number; next_time: number }>({
  loading: false,
  error: '',
  items: [],
  has_more: false,
  limit: 10,
  next_time: 0,
})

const action = reactive<{ loading: boolean; error: string; payload: unknown; name: string }>({
  loading: false,
  error: '',
  payload: null,
  name: '',
})

const canLike = computed(() => auth.isLoggedIn)

async function runAction(name: string, fn: () => Promise<unknown>) {
  action.name = name
  action.loading = true
  action.error = ''
  action.payload = null
  try {
    action.payload = await fn()
  } catch (e) {
    action.error = e instanceof ApiError ? e.message : String(e)
    action.payload = e instanceof ApiError ? e.payload : null
  } finally {
    action.loading = false
  }
}

async function loadLatest(reset: boolean) {
  latest.loading = true
  latest.error = ''
  try {
    const latest_time = reset ? 0 : latest.next_time
    const res = await feedApi.listLatest({ limit: latest.limit, latest_time })
    latest.has_more = res.has_more
    latest.next_time = res.next_time
    latest.items = reset ? res.video_list : latest.items.concat(res.video_list)
  } catch (e) {
    latest.error = e instanceof ApiError ? e.message : String(e)
  } finally {
    latest.loading = false
  }
}

async function loadLikesCount(reset: boolean) {
  likesCount.loading = true
  likesCount.error = ''
  try {
    const res = await feedApi.listLikesCount({
      limit: likesCount.limit,
      likes_count_before: reset ? undefined : likesCount.next_likes_count_before,
      id_before: reset ? undefined : likesCount.next_id_before,
    })
    likesCount.has_more = res.has_more
    likesCount.next_likes_count_before = res.next_likes_count_before
    likesCount.next_id_before = res.next_id_before
    likesCount.items = reset ? res.video_list : likesCount.items.concat(res.video_list)
  } catch (e) {
    likesCount.error = e instanceof ApiError ? e.message : String(e)
  } finally {
    likesCount.loading = false
  }
}

async function loadFollowing(reset: boolean) {
  following.loading = true
  following.error = ''
  try {
    const latest_time = reset ? 0 : following.next_time
    const res = await feedApi.listByFollowing({ limit: following.limit, latest_time })
    following.has_more = res.has_more
    following.next_time = res.next_time
    following.items = reset ? res.video_list : following.items.concat(res.video_list)
  } catch (e) {
    following.error = e instanceof ApiError ? e.message : String(e)
  } finally {
    following.loading = false
  }
}

async function toggleLike(item: FeedVideoItem) {
  if (!auth.isLoggedIn) return

  await runAction(item.is_liked ? '取消点赞' : '点赞', async () => {
    if (item.is_liked) await likeApi.unlike(item.id)
    else await likeApi.like(item.id)

    item.is_liked = !item.is_liked
    item.likes_count = Math.max(0, item.likes_count + (item.is_liked ? 1 : -1))
    return { ok: true, is_liked: item.is_liked, likes_count: item.likes_count }
  })
}

onMounted(async () => {
  await loadLatest(true)
  await loadLikesCount(true)
  if (auth.isLoggedIn) {
    await loadFollowing(true)
  }
})

watch(
  () => auth.isLoggedIn,
  async (v) => {
    if (v && following.items.length === 0) {
      await loadFollowing(true)
    }
  },
)
</script>

<template>
  <AppShell>
    <div class="grid two">
      <div class="card">
        <p class="title">Feed</p>
        <p class="subtle">`/feed/listLatest` 与 `/feed/listLikesCount` 支持匿名（可选 JWT）；`/feed/listByFollowing` 需要 JWT。</p>

        <div class="card" style="margin-top: 12px">
          <div class="row" style="justify-content: space-between">
            <div>
              <p class="title">最新流（listLatest）</p>
              <div class="subtle">limit：{{ latest.limit }} · next_time：{{ latest.next_time }} · has_more：{{ latest.has_more }}</div>
            </div>
            <div class="row">
              <label class="subtle" style="margin: 0">limit</label>
              <input v-model.number="latest.limit" type="number" min="1" max="50" style="width: 90px" />
              <button class="primary" type="button" :disabled="latest.loading" @click="loadLatest(true)">刷新</button>
              <button type="button" :disabled="latest.loading || !latest.has_more" @click="loadLatest(false)">加载更多</button>
            </div>
          </div>
          <div v-if="latest.error" class="pill bad" style="margin-top: 10px">错误：{{ latest.error }}</div>
          <div class="grid" style="gap: 10px; margin-top: 12px">
            <FeedVideoCard
              v-for="item in latest.items"
              :key="`latest-${item.id}`"
              :item="item"
              :can-like="canLike"
              :busy="action.loading"
              @toggle-like="toggleLike"
            />
          </div>
        </div>

        <div class="card" style="margin-top: 12px">
          <div class="row" style="justify-content: space-between">
            <div>
              <p class="title">点赞数流（listLikesCount）</p>
              <div class="subtle">
                limit：{{ likesCount.limit }} · next=(likes={{ likesCount.next_likes_count_before }}, id={{ likesCount.next_id_before }})
                · has_more：{{ likesCount.has_more }}
              </div>
            </div>
            <div class="row">
              <label class="subtle" style="margin: 0">limit</label>
              <input v-model.number="likesCount.limit" type="number" min="1" max="50" style="width: 90px" />
              <button class="primary" type="button" :disabled="likesCount.loading" @click="loadLikesCount(true)">刷新</button>
              <button type="button" :disabled="likesCount.loading || !likesCount.has_more" @click="loadLikesCount(false)">
                加载更多
              </button>
            </div>
          </div>
          <div v-if="likesCount.error" class="pill bad" style="margin-top: 10px">错误：{{ likesCount.error }}</div>
          <div class="grid" style="gap: 10px; margin-top: 12px">
            <FeedVideoCard
              v-for="item in likesCount.items"
              :key="`likes-${item.id}`"
              :item="item"
              :can-like="canLike"
              :busy="action.loading"
              @toggle-like="toggleLike"
            />
          </div>
        </div>

        <div class="card" style="margin-top: 12px">
          <div class="row" style="justify-content: space-between">
            <div>
              <p class="title">关注流（listByFollowing，JWT）</p>
              <div class="subtle">
                limit：{{ following.limit }} · next_time：{{ following.next_time }} · has_more：{{ following.has_more }}
              </div>
            </div>
            <div class="row">
              <label class="subtle" style="margin: 0">limit</label>
              <input v-model.number="following.limit" type="number" min="1" max="50" style="width: 90px" />
              <button class="primary" type="button" :disabled="following.loading" @click="loadFollowing(true)">刷新</button>
              <button type="button" :disabled="following.loading || !following.has_more" @click="loadFollowing(false)">加载更多</button>
            </div>
          </div>
          <div v-if="!auth.isLoggedIn" class="pill bad" style="margin-top: 10px">未登录：无法访问关注流</div>
          <div v-if="following.error" class="pill bad" style="margin-top: 10px">错误：{{ following.error }}</div>
          <div class="grid" style="gap: 10px; margin-top: 12px">
            <FeedVideoCard
              v-for="item in following.items"
              :key="`following-${item.id}`"
              :item="item"
              :can-like="canLike"
              :busy="action.loading"
              @toggle-like="toggleLike"
            />
          </div>
        </div>
      </div>

      <div class="card">
        <p class="title">动作输出（点赞等）</p>
        <div class="row" style="margin-bottom: 10px">
          <span class="pill">动作：{{ action.name || '-' }}</span>
          <span v-if="action.loading" class="pill">请求中…</span>
          <span v-if="action.error" class="pill bad">错误：{{ action.error }}</span>
        </div>
        <JsonBox :value="action.payload" />
      </div>
    </div>
  </AppShell>
</template>
