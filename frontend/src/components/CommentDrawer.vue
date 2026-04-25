<script setup lang="ts">
import { reactive } from 'vue'

import { ApiError } from '../api/client'
import * as commentApi from '../api/comment'
import type { Comment, FeedVideoItem } from '../api/types'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'

const props = defineProps<{ video: FeedVideoItem | null }>()
const emit = defineEmits<{ close: [] }>()

const auth = useAuthStore()
const toast = useToastStore()

const drawer = reactive({
  loading: false,
  error: '',
  comments: [] as Comment[],
  content: '',
})

function needLogin() {
  toast.error('请先登录')
}

function close() {
  drawer.comments = []
  drawer.content = ''
  drawer.error = ''
  emit('close')
}

async function loadComments() {
  if (!props.video) return
  drawer.loading = true
  drawer.error = ''
  try {
    drawer.comments = await commentApi.listAll(props.video.id)
  } catch (e) {
    drawer.error = e instanceof ApiError ? e.message : String(e)
  } finally {
    drawer.loading = false
  }
}

async function publishComment() {
  if (!props.video) return
  if (!auth.isLoggedIn) return needLogin()
  const content = drawer.content.trim()
  if (!content) return
  drawer.loading = true
  drawer.error = ''
  try {
    await commentApi.publish(props.video.id, content)
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
  if (!props.video) return
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

defineExpose({ loadComments })
</script>

<template>
  <div class="drawer-backdrop" @click.self="close">
    <div class="drawer">
      <div class="drawer-head">
        <div class="drawer-title">{{ video?.title ?? '评论' }}</div>
        <button class="drawer-x" type="button" @click="close">×</button>
      </div>

      <div class="drawer-body">
        <div v-if="drawer.loading" class="drawer-hint">加载中…</div>
        <div v-else-if="drawer.error" class="drawer-hint bad">{{ drawer.error }}</div>
        <div v-else-if="drawer.comments.length === 0" class="drawer-hint">暂无评论</div>

        <div class="comment" v-for="c in drawer.comments" :key="c.id">
          <div class="comment-top">
            <div class="comment-user">{{ c.username }}</div>
            <div class="comment-meta mono">#{{ c.id }} · {{ new Date(c.created_at).toLocaleString() }}</div>
          </div>
          <div class="comment-content">{{ c.content }}</div>
          <div class="comment-actions">
            <button v-if="canDeleteComment(c)" class="chip danger" type="button" :disabled="drawer.loading" @click="deleteComment(c.id)">删除</button>
          </div>
        </div>
      </div>

      <div class="drawer-foot">
        <textarea v-model="drawer.content" placeholder="说点什么…" :disabled="drawer.loading" />
        <div class="row" style="justify-content: space-between; margin-top: 8px">
          <button class="chip" type="button" :disabled="drawer.loading" @click="loadComments">刷新</button>
          <button class="chip primary" type="button" :disabled="drawer.loading || !drawer.content.trim()" @click="publishComment">发送</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.drawer-backdrop { position: fixed; inset: 0; background: rgba(0,0,0,0.55); backdrop-filter: blur(10px); z-index: 120; display: grid; justify-items: end; }
.drawer { width: min(420px, calc(100vw - 18px)); height: 100vh; background: rgba(0,0,0,0.65); border-left: 1px solid rgba(255,255,255,0.12); display: grid; grid-template-rows: auto 1fr auto; }
.drawer-head { display: flex; align-items: center; justify-content: space-between; padding: 14px 14px; border-bottom: 1px solid rgba(255,255,255,0.1); }
.drawer-title { font-weight: 800; font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.drawer-x { width: 34px; height: 34px; border-radius: 12px; border: 1px solid rgba(255,255,255,0.14); background: rgba(255,255,255,0.06); color: rgba(255,255,255,0.9); cursor: pointer; font-size: 20px; line-height: 1; }
.drawer-body { overflow: auto; padding: 12px 14px; display: grid; gap: 10px; }
.drawer-foot { border-top: 1px solid rgba(255,255,255,0.1); padding: 12px 14px; }
.drawer-foot textarea { width: 100%; min-height: 82px; resize: none; border-radius: 14px; border: 1px solid rgba(255,255,255,0.14); background: rgba(255,255,255,0.06); color: rgba(255,255,255,0.9); padding: 10px 12px; outline: none; }
.drawer-hint { color: rgba(255,255,255,0.78); padding: 12px 0; }
.drawer-hint.bad { color: rgba(254,44,85,0.92); }
.comment { border: 1px solid rgba(255,255,255,0.1); background: rgba(255,255,255,0.05); border-radius: 14px; padding: 10px 10px; }
.comment-top { display: grid; gap: 3px; }
.comment-user { font-weight: 700; font-size: 13px; }
.comment-meta { font-size: 12px; color: rgba(255,255,255,0.55); }
.comment-content { margin-top: 8px; font-size: 13px; line-height: 1.35; color: rgba(255,255,255,0.86); white-space: pre-wrap; word-break: break-word; }
.comment-actions { margin-top: 10px; display: flex; justify-content: flex-end; }
.chip { display: inline-flex; align-items: center; gap: 8px; padding: 7px 10px; border-radius: 999px; border: 1px solid rgba(255,255,255,0.14); background: rgba(0,0,0,0.28); color: rgba(255,255,255,0.86); font-size: 12px; text-decoration: none; cursor: pointer; }
.chip.primary { border-color: rgba(254,44,85,0.45); background: rgba(254,44,85,0.14); }
.chip.danger { border-color: rgba(254,44,85,0.55); background: rgba(254,44,85,0.12); }
@media (max-width: 900px) {
  .drawer-backdrop { justify-items: center; align-items: end; }
  .drawer { width: calc(100vw - 16px); height: min(72vh, 560px); border-left: none; border-top: 1px solid rgba(255,255,255,0.12); border-radius: 18px 18px 0 0; overflow: hidden; }
}
</style>
