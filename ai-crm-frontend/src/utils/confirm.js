import { t } from '../i18n'

const CONFIRM_BOX_Z_INDEX = 10000
const confirmedDeleteEvents = new WeakSet()
let deleteConfirmedAt = 0
let messageBoxPatched = false

export async function confirmDangerAction(message = t('common.message.confirmDelete'), title = t('system.message.deleteTitle', { type: '' }), options = {}) {
  const ElMessageBox = await loadMessageBox()
  const result = await ElMessageBox.confirm(message, title, {
    type: 'warning',
    confirmButtonText: options.confirmButtonText || t('common.action.delete'),
    cancelButtonText: options.cancelButtonText || t('common.action.cancel'),
    distinguishCancelAndClose: true,
    zIndex: CONFIRM_BOX_Z_INDEX,
    ...options
  })
  markDeleteConfirmed()
  return result
}

export function installDeleteClickGuard() {
  patchMessageBoxConfirm()
  document.addEventListener('click', async (event) => {
    if (confirmedDeleteEvents.has(event)) return
    const target = event.target instanceof Element ? event.target : null
    const trigger = target?.closest('button, [role="button"], .el-dropdown-menu__item, .el-button')
    if (!trigger) return
    if (trigger.closest('.el-popconfirm, .el-popover, .el-popper') && isDeleteTrigger(trigger)) {
      markDeleteConfirmed()
      return
    }
    if (!isLocalDeleteTrigger(trigger)) return

    event.preventDefault()
    event.stopPropagation()
    event.stopImmediatePropagation()
    try {
      await confirmDangerAction(deleteConfirmMessage(trigger), t('system.message.deleteTitle', { type: '' }))
      const next = new MouseEvent('click', event)
      confirmedDeleteEvents.add(next)
      trigger.dispatchEvent(next)
    } catch {
      // User canceled the destructive action.
    }
  }, true)
}

export function hasRecentDeleteConfirmation(maxAgeMs = 5000) {
  return Date.now() - deleteConfirmedAt < maxAgeMs
}

function markDeleteConfirmed() {
  deleteConfirmedAt = Date.now()
}

async function patchMessageBoxConfirm() {
  if (messageBoxPatched) return
  messageBoxPatched = true
  const ElMessageBox = await loadMessageBox()
  const originalConfirm = ElMessageBox.confirm
  ElMessageBox.confirm = (...args) => {
    args[2] = {
      zIndex: CONFIRM_BOX_Z_INDEX,
      ...(args[2] || {})
    }
    const promise = originalConfirm(...args)
    if (isDeleteConfirmation(args)) {
      promise.then(markDeleteConfirmed).catch(() => {})
    }
    return promise
  }
}

async function loadMessageBox() {
  const module = await import('element-plus/es/components/message-box/index')
  return module.ElMessageBox
}

function isDeleteConfirmation(args) {
  return args.some((item) => {
    if (typeof item === 'string') return /删除|刪除|delete/i.test(item)
    if (!item || typeof item !== 'object') return false
    return /删除|刪除|delete/i.test(`${item.confirmButtonText || ''} ${item.title || ''} ${item.message || ''}`)
  })
}

function isLocalDeleteTrigger(el) {
  if (!isDeleteTrigger(el)) return false
  return Boolean(el.closest('.visual-step-list, .acl-editor, .nurture-node-row'))
}

function isDeleteTrigger(el) {
  if (el.closest('.el-message-box, .el-popconfirm')) return false
  const text = compactText(el.textContent)
  const label = compactText(`${el.getAttribute('aria-label') || ''} ${el.getAttribute('title') || ''}`)
  const html = el.innerHTML || ''
  return text === '删除' ||
    text === '刪除' ||
    text.includes('删除作废线索') ||
    label.includes('删除') ||
    /delete-bin|trash|delete/i.test(html)
}

function deleteConfirmMessage(el) {
  const text = compactText(el.textContent)
  if (text && text !== '删除' && text !== '刪除') return t('common.message.confirmActionIrreversible', { action: text })
  return t('common.message.confirmDeleteData')
}

function compactText(value = '') {
  return String(value).replace(/\s+/g, '')
}
