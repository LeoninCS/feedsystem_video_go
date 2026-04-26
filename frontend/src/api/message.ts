import { postJson } from './client'
import type { DirectMessage, ListMessagesResponse } from './types'

export function sendMessage(toId: number, content: string) {
  return postJson<DirectMessage>('/message/send', { to_id: toId, content }, { authRequired: true })
}

export function listMessages(peerId: number) {
  return postJson<ListMessagesResponse>('/message/list', { peer_id: peerId }, { authRequired: true })
}
