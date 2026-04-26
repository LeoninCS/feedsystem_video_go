<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import AppShell from '../components/AppShell.vue'
import UserAvatar from '../components/UserAvatar.vue'
import { ApiError } from '../api/client'
import * as accountApi from '../api/account'
import * as messageApi from '../api/message'
import type { Account, DirectMessage } from '../api/types'
import { useAuthStore } from '../stores/auth'
import { useSocialStore } from '../stores/social'
import { useToastStore } from '../stores/toast'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const social = useSocialStore()
const toast = useToastStore()

const peerId = computed(() => {
  const raw = route.params.peerId
  return typeof raw === 'string' ? Number(raw) : 0
})
const myId = computed(() => auth.claims?.account_id ?? 0)
const hasPeer = computed(() => Number.isFinite(peerId.value) && peerId.value > 0)
const listEl = ref<HTMLDivElement | null>(null)
const content = ref('')

const state = reactive({
  loading: false,
  sending: false,
  error: '',
  peer: null as Account | null,
  messages: [] as DirectMessage[],
})

const orderedMessages = computed(() => [...state.messages].sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()))
const canSend = computed(() => content.value.trim().length > 0 && !state.sending && !!state.peer && peerId.value > 0)
const contactItems = computed(() => {
  const map = new Map<number, Account>()
  for (const user of social.vloggers) map.set(user.id, user)
  for (const user of social.followers) map.set(user.id, user)
  return [...map.values()].filter((user) => user.id !== myId.value)
})
const contactLoading = computed(() => social.vloggersLoading || social.followersLoading)
const contactError = computed(() => social.vloggersError || social.followersError)

function formatTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString([], { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

async function scrollToBottom() {
  await nextTick()
  if (listEl.value) listEl.value.scrollTop = listEl.value.scrollHeight
}

async function loadChat() {
  if (!auth.isLoggedIn) {
    await router.push({ path: '/account', query: { redirect: route.fullPath } })
    return
  }
  if (!hasPeer.value) {
    await social.refreshMine()
    state.loading = false
    state.error = ''
    state.peer = null
    state.messages = []
    return
  }
  if (!Number.isFinite(peerId.value) || peerId.value <= 0) {
    state.error = '无效的用户 id'
    return
  }
  if (peerId.value === myId.value) {
    state.error = '请选择其他用户发送私信'
    return
  }

  state.loading = true
  state.error = ''
  try {
    const [peer, res] = await Promise.all([accountApi.findById(peerId.value), messageApi.listMessages(peerId.value)])
    state.peer = peer
    state.messages = res.messages ?? []
    await scrollToBottom()
  } catch (e) {
    state.error = e instanceof ApiError ? e.message : String(e)
    state.peer = null
    state.messages = []
  } finally {
    state.loading = false
  }
}

async function send() {
  const text = content.value.trim()
  if (!text || state.sending || !state.peer) return

  state.sending = true
  try {
    const msg = await messageApi.sendMessage(peerId.value, text)
    state.messages = [msg, ...state.messages]
    content.value = ''
    await scrollToBottom()
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : String(e)
    toast.error(msg)
  } finally {
    state.sending = false
  }
}

async function goPeerProfile() {
  if (state.peer) await router.push(`/u/${state.peer.id}`)
}

async function openChat(userId: number) {
  await router.push(`/messages/${userId}`)
}

watch(
  () => route.params.peerId,
  () => {
    state.peer = null
    state.messages = []
    content.value = ''
    void loadChat()
  },
)

onMounted(loadChat)
</script>

<template>
  <AppShell>
    <section class="chat-shell">
      <div v-if="!hasPeer" class="contact-panel">
        <div class="contact-head">
          <div>
            <p class="title">私信</p>
            <p class="subtle">选择关注或粉丝里的用户开始聊天。</p>
          </div>
          <button class="ghost small" type="button" :disabled="contactLoading" @click="social.refreshMine">刷新</button>
        </div>

        <div class="contact-body">
          <div v-if="contactLoading" class="chat-hint">加载中…</div>
          <div v-else-if="contactError" class="chat-hint bad">{{ contactError }}</div>
          <div v-else-if="contactItems.length === 0" class="empty">
            <div class="empty-title">暂无可聊天用户</div>
            <div class="empty-text">关注别人或拥有粉丝后，可以从这里开始私信。</div>
          </div>

          <button v-for="user in contactItems" :key="user.id" class="contact-row" type="button" @click="openChat(user.id)">
            <UserAvatar :username="user.username" :id="user.id" :size="42" />
            <span class="contact-meta">
              <span class="contact-name">@{{ user.username }}</span>
              <span class="contact-id mono">#{{ user.id }}</span>
            </span>
            <span class="contact-action">聊天</span>
          </button>
        </div>
      </div>

      <div v-else class="chat-panel">
        <header class="chat-head">
          <button class="icon-btn" type="button" title="返回" @click="router.back()">‹</button>
          <button class="peer" type="button" :disabled="!state.peer" @click="goPeerProfile">
            <UserAvatar :username="state.peer?.username ?? 'User'" :id="state.peer?.id ?? peerId" :size="44" />
            <span class="peer-meta">
              <span class="peer-name">@{{ state.peer?.username ?? '加载中' }}</span>
              <span class="peer-id mono">#{{ state.peer?.id ?? peerId }}</span>
            </span>
          </button>
          <button class="ghost small" type="button" :disabled="state.loading" @click="loadChat">刷新</button>
        </header>

        <div ref="listEl" class="message-list">
          <div v-if="state.loading" class="chat-hint">加载中…</div>
          <div v-else-if="state.error" class="chat-hint bad">{{ state.error }}</div>
          <div v-else-if="orderedMessages.length === 0" class="empty">
            <div class="empty-title">开始聊天</div>
            <div class="empty-text">给 @{{ state.peer?.username ?? '对方' }} 发第一条消息。</div>
          </div>

          <div
            v-for="msg in orderedMessages"
            :key="msg.id"
            class="bubble-row"
            :class="{ mine: msg.from_id === myId }"
          >
            <UserAvatar
              v-if="msg.from_id !== myId"
              :username="state.peer?.username ?? 'User'"
              :id="state.peer?.id ?? msg.from_id"
              :size="32"
            />
            <div class="bubble-wrap">
              <div class="bubble">{{ msg.content }}</div>
              <div class="bubble-time">{{ formatTime(msg.created_at) }}</div>
            </div>
          </div>
        </div>

        <footer class="composer">
          <textarea
            v-model="content"
            placeholder="输入私信内容"
            :disabled="!!state.error || state.loading || state.sending"
            @keydown.enter.exact.prevent="send"
          />
          <button class="primary send-btn" type="button" :disabled="!canSend" @click="send">
            {{ state.sending ? '发送中' : '发送' }}
          </button>
        </footer>
      </div>
    </section>
  </AppShell>
</template>

<style scoped>
.chat-shell {
  min-height: calc(100vh - 104px);
  display: grid;
}

.contact-panel,
.chat-panel {
  height: min(760px, calc(100vh - 136px));
  min-height: 560px;
  overflow: hidden;
  border: 1px solid var(--border);
  background: rgba(13, 18, 29, 0.78);
  border-radius: 8px;
  box-shadow: 0 24px 70px rgba(0, 0, 0, 0.28);
}

.contact-panel,
.chat-panel {
  display: grid;
}

.contact-panel {
  grid-template-rows: auto 1fr;
}

.chat-panel {
  grid-template-rows: auto 1fr auto;
}

.contact-head,
.chat-head {
  min-height: 68px;
  display: grid;
  gap: 10px;
  align-items: center;
  padding: 12px 14px;
  border-bottom: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.035);
}

.contact-head {
  grid-template-columns: 1fr auto;
}

.contact-head .title {
  margin: 0 0 4px;
}

.chat-head {
  grid-template-columns: auto 1fr auto;
}

.contact-body {
  min-height: 0;
  overflow: auto;
  padding: 14px;
  display: grid;
  gap: 10px;
  align-content: start;
}

.contact-row {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  text-align: left;
  padding: 10px;
  border: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.05);
  border-radius: 8px;
}

.contact-row:hover {
  background: rgba(255, 255, 255, 0.08);
}

.contact-meta {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.contact-name {
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.contact-id,
.contact-action {
  color: var(--muted);
  font-size: 12px;
}

.contact-action {
  color: rgba(190, 223, 255, 0.92);
}

.icon-btn {
  width: 40px;
  height: 40px;
  padding: 0;
  border-radius: 8px;
  font-size: 28px;
  line-height: 1;
}

.peer {
  min-width: 0;
  display: inline-grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 10px;
  align-items: center;
  justify-self: start;
  padding: 4px 8px 4px 4px;
  border-radius: 8px;
  background: transparent;
  border-color: transparent;
  text-align: left;
}

.peer:disabled {
  opacity: 0.72;
  cursor: default;
}

.peer-meta {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.peer-name {
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.peer-id {
  color: var(--muted);
  font-size: 12px;
}

.message-list {
  min-height: 0;
  overflow: auto;
  padding: 18px 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.chat-hint {
  margin: auto;
  color: var(--muted);
}

.chat-hint.bad {
  color: var(--danger);
}

.empty {
  margin: auto;
  text-align: center;
  color: var(--muted);
}

.empty-title {
  color: var(--text);
  font-weight: 900;
  margin-bottom: 6px;
}

.empty-text {
  font-size: 13px;
}

.bubble-row {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 10px;
  align-items: end;
}

.bubble-row.mine {
  grid-template-columns: minmax(0, 1fr);
  justify-items: end;
}

.bubble-wrap {
  max-width: min(68%, 560px);
  display: grid;
  gap: 4px;
}

.bubble {
  padding: 10px 12px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
}

.mine .bubble {
  border-color: rgba(43, 161, 255, 0.42);
  background: rgba(43, 161, 255, 0.22);
}

.bubble-time {
  color: var(--muted);
  font-size: 11px;
}

.mine .bubble-time {
  text-align: right;
}

.composer {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
  align-items: end;
  padding: 12px 14px;
  border-top: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.035);
}

.composer textarea {
  min-height: 44px;
  max-height: 132px;
  resize: vertical;
}

.send-btn {
  min-width: 86px;
  height: 44px;
}

.ghost.small {
  padding: 8px 12px;
}

@media (max-width: 760px) {
  .chat-shell {
    min-height: calc(100vh - 56px);
  }

  .chat-panel {
    min-height: calc(100vh - 92px);
    height: calc(100vh - 92px);
  }

  .contact-panel {
    min-height: calc(100vh - 92px);
    height: calc(100vh - 92px);
  }

  .contact-head {
    grid-template-columns: 1fr;
  }

  .bubble-wrap {
    max-width: 86%;
  }

  .composer {
    grid-template-columns: 1fr;
  }

  .send-btn {
    width: 100%;
  }
}
</style>
