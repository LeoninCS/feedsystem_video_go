import { postJson } from './client'
import type { Comment, MessageResponse } from './types'

export function listAll(videoId: number) {
  return postJson<Comment[]>('/comment/listAll', { video_id: videoId })
}

export function publish(videoId: number, content: string) {
  return postJson<MessageResponse>('/comment/publish', { video_id: videoId, content }, { authRequired: true })
}

export function remove(commentId: number) {
  return postJson<MessageResponse>('/comment/delete', { comment_id: commentId }, { authRequired: true })
}
