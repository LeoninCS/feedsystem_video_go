import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { decodeJwtPayload, type JwtPayload } from '../utils/jwt'

const ACCESS_KEY = 'access_token'
const REFRESH_KEY = 'refresh_token'

function readStored(key: string): string | null {
  try { return localStorage.getItem(key) } catch { return null }
}

function writeStored(key: string, value: string) {
  localStorage.setItem(key, value)
}

function removeStored(key: string) {
  localStorage.removeItem(key)
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(readStored(ACCESS_KEY))
  const refreshToken = ref<string | null>(readStored(REFRESH_KEY))

  const isLoggedIn = computed(() => !!token.value)
  const claims = computed<JwtPayload | null>(() => (token.value ? decodeJwtPayload(token.value) : null))

  function setToken(newToken: string) {
    token.value = newToken
    writeStored(ACCESS_KEY, newToken)
  }

  function setTokens(access: string, refresh: string) {
    token.value = access
    refreshToken.value = refresh
    writeStored(ACCESS_KEY, access)
    writeStored(REFRESH_KEY, refresh)
  }

  function clearTokens() {
    token.value = null
    refreshToken.value = null
    removeStored(ACCESS_KEY)
    removeStored(REFRESH_KEY)
  }

  return { token, refreshToken, isLoggedIn, claims, setToken, setTokens, clearTokens }
})
