import { defineStore } from 'pinia'
import { ref } from 'vue'

export type ToastType = 'success' | 'error' | 'info'

export type Toast = {
  id: number
  type: ToastType
  message: string
}

let nextId = 1

export const useToastStore = defineStore('toast', () => {
  const toasts = ref<Toast[]>([])

  function remove(id: number) {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }

  function push(type: ToastType, message: string, ttlMs = 2600) {
    const id = nextId++
    toasts.value.push({ id, type, message })
    window.setTimeout(() => remove(id), ttlMs)
  }

  function success(message: string) {
    push('success', message)
  }

  function error(message: string) {
    push('error', message, 3600)
  }

  function info(message: string) {
    push('info', message)
  }

  return { toasts, push, remove, success, error, info }
})
