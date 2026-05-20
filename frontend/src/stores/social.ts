import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { ApiError } from '../api/client'
import type { Account } from '../api/types'
import * as socialApi from '../api/social'
import { useAuthStore } from './auth'

export const useSocialStore = defineStore('social', () => {
  const auth = useAuthStore()

  const followers = ref<Account[]>([])
  const vloggers = ref<Account[]>([])

  const followersLoading = ref(false)
  const vloggersLoading = ref(false)

  const followersError = ref('')
  const vloggersError = ref('')

  const followerCount = computed(() => followers.value.length)
  const followingCount = computed(() => vloggers.value.length)

  function clear() {
    followers.value = []
    vloggers.value = []
    followersError.value = ''
    vloggersError.value = ''
    followersLoading.value = false
    vloggersLoading.value = false
  }

  function isFollowing(accountId: number) {
    return vloggers.value.some((a) => a.id === accountId)
  }

  async function refreshFollowers(vloggerId?: number) {
    if (!auth.isLoggedIn) {
      clear()
      return
    }

    followersLoading.value = true
    followersError.value = ''
    try {
      const res = await socialApi.getAllFollowers(vloggerId)
      followers.value = res.followers
    } catch (e) {
      followersError.value = e instanceof ApiError ? e.message : String(e)
      followers.value = []
    } finally {
      followersLoading.value = false
    }
  }

  async function refreshVloggers(followerId?: number) {
    if (!auth.isLoggedIn) {
      clear()
      return
    }

    vloggersLoading.value = true
    vloggersError.value = ''
    try {
      const res = await socialApi.getAllVloggers(followerId)
      vloggers.value = res.vloggers
    } catch (e) {
      vloggersError.value = e instanceof ApiError ? e.message : String(e)
      vloggers.value = []
    } finally {
      vloggersLoading.value = false
    }
  }

  async function refreshMine() {
    await Promise.all([refreshFollowers(), refreshVloggers()])
  }

  async function follow(vloggerId: number) {
    if (!auth.isLoggedIn) throw new ApiError('需要先登录', 401)
    await socialApi.follow(vloggerId)
    await refreshVloggers()
  }

  async function unfollow(vloggerId: number) {
    if (!auth.isLoggedIn) throw new ApiError('需要先登录', 401)
    await socialApi.unfollow(vloggerId)
    await refreshVloggers()
  }

  return {
    followers,
    vloggers,
    followerCount,
    followingCount,
    followersLoading,
    vloggersLoading,
    followersError,
    vloggersError,
    clear,
    isFollowing,
    refreshMine,
    refreshFollowers,
    refreshVloggers,
    follow,
    unfollow,
  }
})

