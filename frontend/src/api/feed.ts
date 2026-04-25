import { postJson } from './client'
import { normalizeFeedVideoList } from './normalize'
import type { ListByFollowingResponse, ListByPopularityResponse, ListLatestResponse, ListLikesCountResponse } from './types'

export async function listLatest(input: { limit: number; latest_time: number }) {
  const res = await postJson<ListLatestResponse>('/feed/listLatest', input)
  return { ...res, video_list: normalizeFeedVideoList(res.video_list) }
}

export async function listLikesCount(input: { limit: number; likes_count_before?: number; id_before?: number }) {
  const body: Record<string, unknown> = { limit: input.limit }
  if (typeof input.likes_count_before === 'number' || typeof input.id_before === 'number') {
    body.likes_count_before = input.likes_count_before ?? 0
    body.id_before = input.id_before ?? 0
  }
  const res = await postJson<ListLikesCountResponse>('/feed/listLikesCount', body)
  return { ...res, video_list: normalizeFeedVideoList(res.video_list) }
}

export async function listByPopularity(input: { limit: number; as_of: number; offset: number }) {
  const res = await postJson<ListByPopularityResponse>('/feed/listByPopularity', input)
  return { ...res, video_list: normalizeFeedVideoList(res.video_list) }
}

export async function listByFollowing(input: { limit: number; latest_time: number }) {
  const res = await postJson<ListByFollowingResponse>('/feed/listByFollowing', input, { authRequired: true })
  return { ...res, video_list: normalizeFeedVideoList(res.video_list) }
}
