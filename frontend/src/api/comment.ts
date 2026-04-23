import { postJson } from './client'
import { normalizeCommentList } from './normalize'
import type { Comment, MessageResponse } from './types'

export async function listAll(videoId: number) {
  const comments = await postJson<Comment[] | null>('/comment/listAll', { video_id: videoId })
  return normalizeCommentList(comments)
}

export function publish(videoId: number, content: string) {
  return postJson<MessageResponse>('/comment/publish', { video_id: videoId, content }, { authRequired: true })
}

export function remove(commentId: number) {
  return postJson<MessageResponse>('/comment/delete', { comment_id: commentId }, { authRequired: true })
}
