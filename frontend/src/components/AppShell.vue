<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'

import { useAuthStore } from '../stores/auth'
import { useSocialStore } from '../stores/social'
import Toaster from './Toaster.vue'

const props = defineProps<{ full?: boolean }>()

const auth = useAuthStore()
const social = useSocialStore()
const router = useRouter()
const route = useRoute()

const search = ref(typeof route.query.q === 'string' ? route.query.q : '')
watch(
  () => route.query.q,
  (v) => {
    search.value = typeof v === 'string' ? v : ''
  },
)

watch(
  () => auth.isLoggedIn,
  (v) => {
    if (v) void social.refreshMine()
    else social.clear()
  },
  { immediate: true },
)

const userLabel = computed(() => {
  if (!auth.isLoggedIn) return '未登录'
  const username = auth.claims?.username ?? '(unknown)'
  const accountId = auth.claims?.account_id
  return accountId ? `${username} #${accountId}` : username
})

async function onSearch() {
  const q = search.value.trim()
  await router.push({ path: '/', query: q ? { q } : {} })
}

async function goLogin() {
  await router.push('/account')
}

async function goSettings() {
  await router.push('/settings')
}
</script>

<template>
  <div class="dy-shell">
    <aside class="dy-aside">
      <RouterLink class="dy-logo" to="/">ShortVideo</RouterLink>

      <nav class="dy-nav">
        <RouterLink class="dy-nav-link" to="/">推荐</RouterLink>
        <RouterLink class="dy-nav-link" to="/video">发布</RouterLink>
        <RouterLink class="dy-nav-link" to="/account">账号</RouterLink>
        <RouterLink class="dy-nav-link" to="/settings">设置</RouterLink>
      </nav>

      <div class="dy-aside-foot">
        <div class="dy-user">
          <span class="dy-user-dot" :class="auth.isLoggedIn ? 'ok' : 'bad'" />
          <span class="dy-user-name">{{ userLabel }}</span>
        </div>
        <div class="dy-user-actions">
          <button v-if="!auth.isLoggedIn" class="dy-btn dy-btn-primary" type="button" @click="goLogin">登录</button>
          <button v-else class="dy-btn dy-btn-primary" type="button" @click="goSettings">设置</button>
        </div>
      </div>
    </aside>

    <div class="dy-main">
      <header class="dy-topbar">
        <div class="dy-top-left">
          <div class="dy-tabs-hint">{{ route.name }}</div>
        </div>

        <div class="dy-search">
          <input v-model="search" class="dy-search-input" placeholder="搜索标题 / 作者（本地过滤）" @keydown.enter="onSearch" />
          <button class="dy-btn dy-btn-primary" type="button" @click="onSearch">搜索</button>
        </div>

        <div class="dy-top-right">
          <RouterLink class="dy-btn dy-btn-ghost" to="/video">+ 发布视频</RouterLink>
        </div>
      </header>

      <div class="dy-content" :class="props.full ? 'full' : 'padded'">
        <template v-if="props.full">
          <slot />
        </template>
        <template v-else>
          <div class="container">
            <slot />
          </div>
        </template>
      </div>
    </div>

    <Toaster />
  </div>
</template>

<style scoped>
.dy-shell {
  height: 100vh;
  display: grid;
  grid-template-columns: 240px 1fr;
  background: radial-gradient(1200px 900px at 20% -25%, rgba(254, 44, 85, 0.18), transparent 60%),
    radial-gradient(900px 700px at 90% 10%, rgba(37, 244, 238, 0.12), transparent 55%), transparent;
}

.dy-aside {
  border-right: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(16px);
  padding: 14px 12px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.dy-logo {
  font-weight: 900;
  letter-spacing: 0.4px;
  font-size: 18px;
  padding: 10px 10px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.dy-nav {
  display: grid;
  gap: 8px;
}

.dy-nav-link {
  padding: 10px 10px;
  border-radius: 12px;
  border: 1px solid transparent;
  background: rgba(255, 255, 255, 0.04);
}

.dy-nav-link.router-link-active {
  border-color: rgba(254, 44, 85, 0.42);
  background: rgba(254, 44, 85, 0.12);
}

.dy-aside-foot {
  margin-top: auto;
  display: grid;
  gap: 10px;
  padding-top: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
}

.dy-user {
  display: flex;
  gap: 10px;
  align-items: center;
}

.dy-user-dot {
  width: 10px;
  height: 10px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.25);
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.06);
}

.dy-user-dot.ok {
  background: rgba(34, 197, 94, 1);
  box-shadow: 0 0 0 3px rgba(34, 197, 94, 0.14);
}

.dy-user-dot.bad {
  background: rgba(254, 44, 85, 1);
  box-shadow: 0 0 0 3px rgba(254, 44, 85, 0.14);
}

.dy-user-name {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.86);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dy-user-actions {
  display: flex;
  gap: 10px;
}

.dy-btn {
  appearance: none;
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.9);
  border-radius: 12px;
  padding: 10px 12px;
  cursor: pointer;
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  justify-content: center;
  font-size: 13px;
}

.dy-btn:hover {
  background: rgba(255, 255, 255, 0.1);
}

.dy-btn-primary {
  border-color: rgba(254, 44, 85, 0.5);
  background: rgba(254, 44, 85, 0.16);
}

.dy-btn-primary:hover {
  background: rgba(254, 44, 85, 0.24);
}

.dy-btn-ghost {
  border-color: rgba(255, 255, 255, 0.14);
  background: rgba(0, 0, 0, 0.15);
}

.dy-main {
  height: 100vh;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.dy-topbar {
  height: 56px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(0, 0, 0, 0.28);
  backdrop-filter: blur(16px);
  display: grid;
  grid-template-columns: 180px 1fr 180px;
  gap: 12px;
  align-items: center;
  padding: 0 14px;
}

.dy-tabs-hint {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.55);
  text-transform: uppercase;
  letter-spacing: 0.16em;
}

.dy-search {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 10px;
  align-items: center;
  max-width: 680px;
  width: 100%;
  justify-self: center;
}

.dy-search-input {
  width: 100%;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 999px;
  color: rgba(255, 255, 255, 0.9);
  padding: 10px 14px;
  outline: none;
}

.dy-search-input:focus {
  border-color: rgba(37, 244, 238, 0.42);
  box-shadow: 0 0 0 3px rgba(37, 244, 238, 0.14);
}

.dy-top-right {
  display: flex;
  justify-content: flex-end;
}

.dy-content {
  flex: 1;
  min-height: 0;
}

.dy-content.padded {
  overflow: auto;
}

.dy-content.full {
  overflow: hidden;
}

@media (max-width: 900px) {
  .dy-shell {
    grid-template-columns: 1fr;
  }
  .dy-aside {
    display: none;
  }
  .dy-topbar {
    grid-template-columns: 1fr auto;
  }
  .dy-top-left {
    display: none;
  }
  .dy-top-right {
    display: none;
  }
}
</style>
