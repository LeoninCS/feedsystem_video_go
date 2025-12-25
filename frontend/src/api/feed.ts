import { postJson } from './client'
import type { ListByFollowingResponse, ListLatestResponse, ListLikesCountResponse } from './types'

export function listLatest(input: { limit: number; latest_time: number }) {
  return postJson<ListLatestResponse>('/feed/listLatest', input)
}

export function listLikesCount(input: { limit: number; likes_count_before?: number; id_before?: number }) {
  const body: Record<string, unknown> = { limit: input.limit }
  if (typeof input.likes_count_before === 'number' || typeof input.id_before === 'number') {
    body.likes_count_before = input.likes_count_before ?? 0
    body.id_before = input.id_before ?? 0
  }
  return postJson<ListLikesCountResponse>('/feed/listLikesCount', body)
}

export function listByFollowing(input: { limit: number; latest_time: number }) {
  return postJson<ListByFollowingResponse>('/feed/listByFollowing', input, { authRequired: true })
}
