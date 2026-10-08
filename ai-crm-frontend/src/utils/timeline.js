import { t } from '../i18n'

export function timelinePayload(item) {
  try {
    return typeof item?.payload_json === 'string' ? JSON.parse(item.payload_json || '{}') : (item?.payload_json || {})
  } catch {
    return {}
  }
}

export function timelineOperatorText(item, users = [], formatTime = (value) => value || '-') {
  const payload = timelinePayload(item)
  const operatorID = Number(payload.operator_id || item?.operator_id || item?.created_by || 0)
  const user = users.find((entry) => Number(entry.id) === operatorID)
  const operator = payload.operator_name || payload.operator || item?.operator_name || item?.operator || user?.name || user?.username || user?.email || (operatorID ? t('timeline.userFallback', { id: operatorID }) : t('timeline.unknownUser'))
  const operatedAt = payload.operated_at || payload.happened_at || item?.operated_at || item?.happened_at || item?.created_at
  return t('timeline.operatorText', { operator, time: formatTime(operatedAt) })
}
