import { postForm, postJson } from './client'
import type { Account, MessageResponse, TokenResponse } from './types'

export function register(username: string, password: string) {
  return postJson<MessageResponse>('/account/register', { username, password })
}

export function login(username: string, password: string) {
  return postJson<TokenResponse>('/account/login', { username, password })
}

export function logout() {
  return postJson<MessageResponse>('/account/logout', {}, { authRequired: true })
}

export function rename(newUsername: string) {
  return postJson<TokenResponse>('/account/rename', { new_username: newUsername }, { authRequired: true })
}

export function changePassword(username: string, oldPassword: string, newPassword: string) {
  return postJson<MessageResponse>('/account/changePassword', {
    username,
    old_password: oldPassword,
    new_password: newPassword,
  })
}

export function findById(id: number) {
  return postJson<Account>('/account/findByID', { id })
}

export function findByUsername(username: string) {
  return postJson<Account>('/account/findByUsername', { username })
}

export function uploadAvatar(file: File) {
  const fd = new FormData()
  fd.append('file', file)
  return postForm<{ avatar_url: string }>('/account/uploadAvatar', fd, { authRequired: true })
}

export function updateProfile(data: { avatar_url?: string; bio?: string }) {
  return postJson<MessageResponse>('/account/updateProfile', data, { authRequired: true })
}

export function refresh(refreshToken: string) {
  return postJson<TokenResponse>('/account/refresh', { refresh_token: refreshToken })
}
