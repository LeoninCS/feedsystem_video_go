<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'

import AppShell from '../components/AppShell.vue'
import { ApiError } from '../api/client'
import * as accountApi from '../api/account'
import { useToastStore } from '../stores/toast'

const router = useRouter()
const toast = useToastStore()

const busy = ref(false)
const form = reactive({ username: '', oldPassword: '', newPassword: '' })

async function submit() {
  if (busy.value) return
  const username = form.username.trim()
  const oldPassword = form.oldPassword.trim()
  const newPassword = form.newPassword.trim()
  if (!username || !oldPassword || !newPassword) {
    toast.error('请把信息填完整')
    return
  }

  busy.value = true
  try {
    await accountApi.changePassword(username, oldPassword, newPassword)
    toast.success('密码已修改，请重新登录')
    await router.push('/account')
  } catch (e) {
    const msg = e instanceof ApiError ? e.message : String(e)
    toast.error(msg)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AppShell>
    <div class="grid two">
      <div class="card">
        <p class="title">修改密码</p>
        <p class="subtle">不需要登录（对应后端 `/account/changePassword`）。</p>
        <div class="grid" style="margin-top: 12px">
          <div>
            <label>username</label>
            <input v-model.trim="form.username" autocomplete="username" />
          </div>
          <div>
            <label>old_password</label>
            <input v-model.trim="form.oldPassword" type="password" autocomplete="current-password" />
          </div>
          <div>
            <label>new_password</label>
            <input v-model.trim="form.newPassword" type="password" autocomplete="new-password" />
          </div>
          <div class="row" style="justify-content: flex-end">
            <button class="primary" type="button" :disabled="busy" @click="submit">提交</button>
          </div>
        </div>
      </div>

      <div class="card">
        <p class="title">提示</p>
        <p class="muted">改密成功后后端会让旧 token 失效；请在「账号」页重新登录。</p>
      </div>
    </div>
  </AppShell>
</template>

