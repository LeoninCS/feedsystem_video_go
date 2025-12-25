export type JwtPayload = {
  account_id?: number
  username?: string
  exp?: number
  iat?: number
  nbf?: number
  [key: string]: unknown
}

function base64UrlToBase64(input: string) {
  const base64 = input.replace(/-/g, '+').replace(/_/g, '/')
  const pad = base64.length % 4
  return pad === 0 ? base64 : base64 + '='.repeat(4 - pad)
}

export function decodeJwtPayload(token: string): JwtPayload | null {
  const [, payload] = token.split('.')
  if (!payload) return null

  try {
    const json = atob(base64UrlToBase64(payload))
    const parsed = JSON.parse(json)
    if (!parsed || typeof parsed !== 'object') return null
    return parsed as JwtPayload
  } catch {
    return null
  }
}
