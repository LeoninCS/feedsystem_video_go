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
const form = reactive({ username: '', password: '' })

async function submit() {
  if (busy.value) return
  const username = form.username.trim()
  const password = form.password.trim()
  if (!username || !password) {
    toast.error('请输入 username 和 password')
    return
  }

  busy.value = true
  try {
    await accountApi.register(username, password)
    toast.success('注册成功，请登录')
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
        <p class="title">注册</p>
        <p class="subtle">创建新账号（对应后端 `/account/register`）。</p>
        <div class="grid" style="margin-top: 12px">
          <div>
            <label>username</label>
            <input v-model.trim="form.username" autocomplete="username" />
          </div>
          <div>
            <label>password</label>
            <input v-model.trim="form.password" type="password" autocomplete="new-password" />
          </div>
          <div class="row" style="justify-content: flex-end">
            <button class="primary" type="button" :disabled="busy" @click="submit">注册</button>
          </div>
        </div>
      </div>

      <div class="card">
        <p class="title">提示</p>
        <p class="muted">注册成功后会跳回「账号」页进行登录。</p>
      </div>
    </div>
  </AppShell>
</template>

