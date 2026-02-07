const KEY = 'token'

export function getToken(): string | null {
  return localStorage.getItem(KEY)
}

export function setToken(t: string) {
  localStorage.setItem(KEY, t)
}

export function clearToken() {
  localStorage.removeItem(KEY)
}

export function getTokenPayload(token: string): any | null {
  const parts = token.split('.')
  if (parts.length < 2) return null
  const basePart = parts[1]
  if (!basePart) return null
  const base = basePart.replace(/-/g, '+').replace(/_/g, '/')
  const pad = '='.repeat((4 - (base.length % 4)) % 4)
  try {
    if (typeof globalThis.atob !== 'function') return null
    const json = globalThis.atob(base + pad)
    return JSON.parse(json)
  } catch {
    return null
  }
}
