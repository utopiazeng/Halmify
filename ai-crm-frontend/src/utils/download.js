import api from '../api'
import { t } from '../i18n'

export async function downloadProtectedFile(url, filename = 'download') {
  if (!url) return
  const data = await api.get(url, { responseType: 'blob' })
  const blob = data instanceof Blob ? data : new Blob([data])
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = sanitizeDownloadName(filename)
  document.body.appendChild(link)
  link.click()
  URL.revokeObjectURL(link.href)
  link.remove()
}

export function unwrapFileUploadResponse(response) {
  const candidates = [
    response?.data?.data,
    response?.data,
    response
  ]
  const info = candidates.find((item) => item && typeof item === 'object' && item.file_id)
  if (!info?.file_id) {
    throw new Error(t('files.message.missingUploadFileId'))
  }
  return info
}

function sanitizeDownloadName(value) {
  return String(value || 'download').replace(/[\\/:*?"<>|]/g, '_')
}
