<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { RouterLink } from 'vue-router'

import AppShell from '../components/AppShell.vue'
import JsonBox from '../components/JsonBox.vue'
import UserAvatar from '../components/UserAvatar.vue'
import { ApiError } from '../api/client'
import * as videoApi from '../api/video'
import type { Video } from '../api/types'
import { useAuthStore } from '../stores/auth'
import { useSocialStore } from '../stores/social'
import { useToastStore } from '../stores/toast'

const auth = useAuthStore()
const social = useSocialStore()
const toast = useToastStore()
const myId = computed(() => auth.claims?.account_id ?? 0)

const last = reactive<{ action: string; loading: boolean; data: unknown }>({
  action: '',
  loading: false,
  data: null,
})

async function exec(action: string, fn: () => Promise<unknown>) {
  last.action = action
  last.loading = true
  last.data = null
  try {
    const res = await fn()
    last.data = res
    return res
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : String(e)
    toast.error(msg)
    last.data = e instanceof ApiError ? e.payload : null
    return null
  } finally {
    last.loading = false
  }
}

const publishForm = reactive({ title: '', description: '', play_url: '', cover_url: '' })
const listAuthorId = ref<number>(1)
const detailId = ref<number>(1)

const listResult = ref<Video[] | null>(null)
const followBusy = reactive<Record<string, boolean>>({})

async function onPublish() {
  const res = await exec('发布视频', () => videoApi.publishVideo(publishForm))
  if (res) toast.success('已发布')
}

async function onListByAuthor() {
  await exec('按作者列出视频', async () => {
    const res = await videoApi.listByAuthorId(listAuthorId.value)
    listResult.value = res
    return res
  })
}

async function onGetDetail() {
  await exec('获取视频详情', () => videoApi.getDetail(detailId.value))
}

async function toggleFollow(authorId: number) {
  if (!auth.isLoggedIn) {
    toast.error('请先登录')
    return
  }
  if (myId.value && myId.value === authorId) return

  const key = String(authorId)
  if (followBusy[key]) return
  followBusy[key] = true
  try {
    if (social.isFollowing(authorId)) {
      await social.unfollow(authorId)
      toast.info('已取关')
    } else {
      await social.follow(authorId)
      toast.success('已关注')
    }
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : String(e)
    toast.error(msg)
  } finally {
    followBusy[key] = false
  }
}
</script>

<template>
  <AppShell>
    <div class="grid two">
      <div class="card">
        <p class="title">视频</p>
        <p class="subtle">发布需要 JWT；列表/详情无需 JWT。</p>

        <div class="card" style="margin-top: 12px">
          <p class="title">发布视频（JWT）</p>
          <div class="grid two">
            <div>
              <label>title</label>
              <input v-model.trim="publishForm.title" />
            </div>
            <div>
              <label>description</label>
              <input v-model.trim="publishForm.description" />
            </div>
            <div>
              <label>play_url</label>
              <input v-model.trim="publishForm.play_url" placeholder="http://..." />
            </div>
            <div>
              <label>cover_url</label>
              <input v-model.trim="publishForm.cover_url" placeholder="http://..." />
            </div>
          </div>
          <div class="row" style="margin-top: 10px">
            <button class="primary" type="button" :disabled="last.loading" @click="onPublish">发布</button>
          </div>
        </div>

        <div class="grid two" style="margin-top: 12px">
          <div class="card">
            <p class="title">按作者列出</p>
            <div class="grid">
              <div>
                <label>author_id</label>
                <input v-model.number="listAuthorId" type="number" min="1" />
              </div>
              <button class="primary" type="button" :disabled="last.loading" @click="onListByAuthor">查询</button>
              <div v-if="listResult" class="subtle">共 {{ listResult.length }} 条</div>
            </div>
          </div>
          <div class="card">
            <p class="title">详情</p>
            <div class="grid">
              <div>
                <label>id</label>
                <input v-model.number="detailId" type="number" min="1" />
              </div>
              <button class="primary" type="button" :disabled="last.loading" @click="onGetDetail">获取</button>
              <RouterLink class="pill" :to="`/video/${detailId}`">打开详情页（含评论/点赞）</RouterLink>
            </div>
          </div>
        </div>

        <div v-if="listResult" class="card" style="margin-top: 12px">
          <p class="title">列表结果</p>
          <div class="grid" style="gap: 10px">
            <div v-for="v in listResult" :key="v.id" class="card" style="background: rgba(255, 255, 255, 0.05)">
              <div class="row" style="justify-content: space-between">
                <div>
                  <div class="title">
                    <RouterLink :to="`/video/${v.id}`">{{ v.title }}</RouterLink>
                  </div>
                  <div class="row" style="gap: 10px; margin-top: 6px">
                    <RouterLink class="author-link" :to="`/u/${v.author_id}`">
                      <UserAvatar :username="v.username" :id="v.author_id" :size="28" />
                      <span class="author-name">@{{ v.username }}</span>
                    </RouterLink>
                    <span class="subtle mono">#{{ v.author_id }}</span>
                    <span class="subtle">❤️ {{ v.likes_count }}</span>
                    <span class="subtle">{{ new Date(v.create_time).toLocaleString() }}</span>
                  </div>
                </div>
                <div class="row">
                  <button
                    v-if="auth.isLoggedIn && (!myId || myId !== v.author_id)"
                    class="primary"
                    type="button"
                    style="padding: 8px 10px"
                    :disabled="!!followBusy[String(v.author_id)]"
                    @click.stop="toggleFollow(v.author_id)"
                  >
                    {{ social.isFollowing(v.author_id) ? '已关注' : '关注' }}
                  </button>
                  <RouterLink class="pill" :to="`/video/${v.id}`">详情</RouterLink>
                </div>
              </div>
              <div class="row" style="margin-top: 10px">
                <a class="pill mono" :href="v.play_url" target="_blank" rel="noreferrer">play_url</a>
                <a class="pill mono" :href="v.cover_url" target="_blank" rel="noreferrer">cover_url</a>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="card">
        <p class="title">最近响应</p>
        <div class="row" style="margin-bottom: 10px">
          <span class="pill">动作：{{ last.action || '-' }}</span>
          <span v-if="last.loading" class="pill">请求中…</span>
        </div>
        <JsonBox :value="last.data" />
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
.author-link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  text-decoration: none;
}

.author-link:hover {
  text-decoration: none;
}

.author-name {
  font-weight: 800;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
}
</style>
