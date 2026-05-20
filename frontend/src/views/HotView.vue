<script setup lang="ts">
import { computed, onMounted, reactive } from 'vue'

import { ApiError } from '../api/client'
import * as feedApi from '../api/feed'
import * as likeApi from '../api/like'
import type { FeedVideoItem } from '../api/types'
import AppShell from '../components/AppShell.vue'
import FeedVideoCard from '../components/FeedVideoCard.vue'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'

const auth = useAuthStore()
const toast = useToastStore()

const canLike = computed(() => auth.isLoggedIn)

const state = reactive({
  loading: false,
  error: '',
  items: [] as FeedVideoItem[],
  hasMore: false,
  limit: 10,
  asOf: 0,
  nextOffset: 0,
})

const likeBusy = reactive<Record<string, boolean>>({})

async function loadHot(reset: boolean) {
  if (state.loading) return
  state.loading = true
  state.error = ''
  try {
    const res = await feedApi.listByPopularity({
      limit: state.limit,
      as_of: reset ? 0 : state.asOf,
      offset: reset ? 0 : state.nextOffset,
    })
    state.hasMore = res.has_more
    state.asOf = res.as_of
    state.nextOffset = res.next_offset
    state.items = reset ? res.video_list : state.items.concat(res.video_list)
  } catch (e) {
    state.error = e instanceof ApiError ? e.message : String(e)
  } finally {
    state.loading = false
  }
}

async function toggleLike(item: FeedVideoItem) {
  if (!auth.isLoggedIn) {
    toast.error('请先登录')
    return
  }
  const key = String(item.id)
  if (likeBusy[key]) return
  likeBusy[key] = true
  try {
    if (item.is_liked) await likeApi.unlike(item.id)
    else await likeApi.like(item.id)
    item.is_liked = !item.is_liked
    item.likes_count = Math.max(0, item.likes_count + (item.is_liked ? 1 : -1))
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : String(e)
    toast.error(msg)
  } finally {
    likeBusy[key] = false
  }
}

onMounted(async () => {
  await loadHot(true)
})
</script>

<template>
  <AppShell>
    <div class="card">
      <div class="row" style="justify-content: space-between; align-items: baseline">
        <div>
          <p class="title" style="margin: 0">热榜</p>
          <p class="subtle" style="margin: 6px 0 0">按热度排序（/feed/listByPopularity）</p>
        </div>

        <div class="row">
          <label class="subtle" style="margin: 0">limit</label>
          <input v-model.number="state.limit" type="number" min="1" max="50" style="width: 90px" :disabled="state.loading" />
          <button class="primary" type="button" :disabled="state.loading" @click="loadHot(true)">刷新</button>
          <button type="button" :disabled="state.loading || !state.hasMore" @click="loadHot(false)">加载更多</button>
        </div>
      </div>

      <div v-if="state.error" class="pill bad" style="margin-top: 12px">错误：{{ state.error }}</div>
      <div v-else-if="state.loading && state.items.length === 0" class="subtle" style="margin-top: 12px">加载中…</div>
      <div v-else-if="state.items.length === 0" class="subtle" style="margin-top: 12px">暂无内容</div>

      <div v-if="state.items.length" class="rank-list" style="margin-top: 14px">
        <div v-for="(item, idx) in state.items" :key="`hot-${item.id}`" class="rank-row">
          <div class="rank-num" :class="idx < 3 ? 'top' : ''">{{ idx + 1 }}</div>
          <FeedVideoCard
            :item="item"
            :can-like="canLike"
            :busy="!!likeBusy[String(item.id)]"
            @toggle-like="toggleLike"
          />
        </div>
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
.rank-list {
  display: grid;
  gap: 12px;
}

.rank-row {
  display: grid;
  grid-template-columns: 44px minmax(0, 1fr);
  gap: 12px;
  align-items: start;
}

.rank-num {
  height: 44px;
  width: 44px;
  border-radius: 16px;
  display: grid;
  place-items: center;
  font-weight: 900;
  letter-spacing: 0.2px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.86);
  user-select: none;
}

.rank-num.top {
  border-color: rgba(254, 44, 85, 0.55);
  background: rgba(254, 44, 85, 0.18);
  color: rgba(255, 255, 255, 0.96);
}

@media (max-width: 900px) {
  .rank-row {
    grid-template-columns: 38px minmax(0, 1fr);
    gap: 10px;
  }
  .rank-num {
    height: 38px;
    width: 38px;
    border-radius: 14px;
  }
}
</style>
