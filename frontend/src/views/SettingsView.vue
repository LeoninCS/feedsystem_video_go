<script setup lang="ts">
import { computed, nextTick, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'

import AppShell from '../components/AppShell.vue'
import UserAvatar from '../components/UserAvatar.vue'
import { ApiError } from '../api/client'
import * as accountApi from '../api/account'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'

const router = useRouter()
const auth = useAuthStore()
const toast = useToastStore()

const busy = ref(false)

const me = computed(() => ({
  id: auth.claims?.account_id ?? 0,
  username: auth.claims?.username ?? '',
}))

const rename = reactive({
  open: false,
  newUsername: '',
})

async function openRename() {
  if (!auth.isLoggedIn) return
  rename.open = true
  rename.newUsername = me.value.username
  await nextTick()
}

async function submitRename() {
  if (!auth.isLoggedIn) return
  if (busy.value) return
  const newUsername = rename.newUsername.trim()
  if (!newUsername) {
    toast.error('请输入新用户名')
    return
  }

  busy.value = true
  try {
    const res = await accountApi.rename(newUsername)
    auth.setToken(res.token)
    rename.open = false
    toast.success('改名成功（已刷新 token）')
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : String(e)
    toast.error(msg)
  } finally {
    busy.value = false
  }
}

async function goLogin() {
  await router.push('/account')
}

async function goChangePassword() {
  await router.push('/account/change-password')
}

async function onLogout() {
  if (!auth.isLoggedIn) return
  if (busy.value) return
  if (!window.confirm('确认退出登录？')) return

  busy.value = true
  try {
    await accountApi.logout()
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : String(e)
    toast.error(`登出失败：${msg}`)
  } finally {
    auth.clearToken()
    rename.open = false
    toast.info('已退出登录')
    busy.value = false
    await router.push('/')
  }
}
</script>

<template>
  <AppShell>
    <div v-if="!auth.isLoggedIn" class="grid two">
      <div class="card">
        <p class="title">设置</p>
        <p class="subtle">需要先登录后才能进行改名/退出等操作。</p>
        <div class="row" style="margin-top: 12px; justify-content: flex-end">
          <button class="primary" type="button" @click="goLogin">去登录</button>
        </div>
      </div>
      <div class="card">
        <p class="title">提示</p>
        <p class="muted">登录入口在「账号」页。</p>
      </div>
    </div>

    <div v-else class="grid two">
      <div class="card">
        <div class="row" style="justify-content: space-between; align-items: flex-start">
          <div class="row" style="gap: 12px; align-items: center">
            <UserAvatar :username="me.username" :id="me.id" :size="56" />
            <div>
              <div class="title" style="margin: 0">@{{ me.username }}</div>
              <div class="subtle mono">#{{ me.id }}</div>
            </div>
          </div>
        </div>

        <div class="card" style="margin-top: 14px">
          <div class="row" style="justify-content: space-between; align-items: center">
            <p class="title" style="margin: 0">账号设置</p>
            <button class="ghost" type="button" :disabled="busy" @click="openRename">改名</button>
          </div>

          <div v-if="rename.open" class="grid" style="margin-top: 12px">
            <div>
              <label>new_username</label>
              <input v-model.trim="rename.newUsername" @keydown.enter="submitRename" />
            </div>
            <div class="row" style="justify-content: flex-end">
              <button type="button" :disabled="busy" @click="rename.open = false">取消</button>
              <button class="primary" type="button" :disabled="busy" @click="submitRename">提交</button>
            </div>
          </div>
        </div>

        <div class="card" style="margin-top: 14px">
          <p class="title">账号安全</p>
          <div class="row">
            <button class="ghost" type="button" :disabled="busy" @click="goChangePassword">修改密码</button>
            <button class="danger" type="button" :disabled="busy" @click="onLogout">退出登录</button>
          </div>
        </div>
      </div>

      <div class="card">
        <p class="title">说明</p>
        <div class="grid" style="margin-top: 10px">
          <div class="pill ok">改名后会返回新 token，旧 token 立即失效</div>
          <div class="pill ok">退出登录会清空本地 token</div>
          <div class="pill">修改密码无需登录，但成功后会让旧 token 失效</div>
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
</style>

