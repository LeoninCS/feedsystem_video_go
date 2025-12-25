import { postJson } from './client'
import type { Video } from './types'

export function publishVideo(input: { title: string; description: string; play_url: string; cover_url: string }) {
  return postJson<Video>('/video/publish', input, { authRequired: true })
}

export function listByAuthorId(authorId: number) {
  return postJson<Video[]>('/video/listByAuthorID', { author_id: authorId })
}

export function getDetail(id: number) {
  return postJson<Video>('/video/getDetail', { id })
}
