import api from '../api'

export function isProtectedFileUrl(url = '') {
  return /^\/api\/v1\/files\/[^/]+\/download(?:\?.*)?$/i.test(String(url || '').trim())
}

export async function resolveProtectedImageUrl(url = '') {
  const value = String(url || '').trim()
  if (!value || !isProtectedFileUrl(value)) return value
  const blob = await api.get(value, { responseType: 'blob' })
  return URL.createObjectURL(blob instanceof Blob ? blob : new Blob([blob]))
}

export function revokeObjectUrl(url = '') {
  if (String(url || '').startsWith('blob:')) URL.revokeObjectURL(url)
}
