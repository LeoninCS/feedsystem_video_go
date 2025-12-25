import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { decodeJwtPayload, type JwtPayload } from '../utils/jwt'

const TOKEN_KEY = 'jwt_token'

function readToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY)
  } catch {
    return null
  }
}

function writeToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

function removeToken() {
  localStorage.removeItem(TOKEN_KEY)
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(readToken())

  const isLoggedIn = computed(() => !!token.value)
  const claims = computed<JwtPayload | null>(() => (token.value ? decodeJwtPayload(token.value) : null))

  function setToken(newToken: string) {
    token.value = newToken
    writeToken(newToken)
  }

  function clearToken() {
    token.value = null
    removeToken()
  }

  function syncFromStorage() {
    token.value = readToken()
  }

  return { token, isLoggedIn, claims, setToken, clearToken, syncFromStorage }
})
