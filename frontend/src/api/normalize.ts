import type { Account, Comment, FeedAuthor, FeedVideoItem, Video } from './types'

export function listOrEmpty<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : []
}

export function normalizeAccount(value: Account | null | undefined): Account {
  return {
    id: Number(value?.id ?? 0),
    username: value?.username || '匿名用户',
  }
}

function normalizeAuthor(value: FeedAuthor | null | undefined): FeedAuthor {
  return {
    id: Number(value?.id ?? 0),
    username: value?.username || '匿名用户',
  }
}

export function normalizeFeedVideoItem(value: FeedVideoItem): FeedVideoItem {
  return {
    ...value,
    author: normalizeAuthor(value.author),
    title: value.title || '未命名视频',
    description: value.description || '',
    play_url: value.play_url || '',
    cover_url: value.cover_url || '',
    create_time: Number(value.create_time ?? 0),
    likes_count: Number(value.likes_count ?? 0),
    is_liked: Boolean(value.is_liked),
  }
}

export function normalizeFeedVideoList(value: FeedVideoItem[] | null | undefined): FeedVideoItem[] {
  return listOrEmpty(value).map(normalizeFeedVideoItem)
}

export function normalizeVideoList(value: Video[] | null | undefined): Video[] {
  return listOrEmpty(value).map((video) => ({
    ...video,
    username: video.username || '匿名用户',
    title: video.title || '未命名视频',
    description: video.description || '',
    play_url: video.play_url || '',
    cover_url: video.cover_url || '',
    likes_count: Number(video.likes_count ?? 0),
  }))
}

export function normalizeCommentList(value: Comment[] | null | undefined): Comment[] {
  return listOrEmpty(value).map((comment) => ({
    ...comment,
    username: comment.username || '匿名用户',
    content: comment.content || '',
  }))
}
