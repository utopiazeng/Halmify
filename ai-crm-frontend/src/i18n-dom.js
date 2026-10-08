import { nextTick, watch } from 'vue'
import { i18nState, t } from './i18n'

let observer

export function installDomI18n(app) {
  app.mixin({
    mounted() {
      scheduleTranslate()
    },
    updated() {
      scheduleTranslate()
    }
  })
  watch(() => i18nState.locale, () => scheduleTranslate())
}

function scheduleTranslate() {
  nextTick(() => translateDocument())
}

function translateDocument() {
  const root = document.body
  if (!root) return
  translateTextNodes(root)
  translateAttributes(root)
  if (!observer) {
    observer = new MutationObserver(() => {
      translateDocument()
    })
    observer.observe(root, { childList: true, subtree: true })
  }
}

function translateTextNodes(root) {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT)
  const nodes = []
  while (walker.nextNode()) nodes.push(walker.currentNode)
  for (const node of nodes) {
    const parentTag = node.parentElement?.tagName
    if (!node.nodeValue || parentTag === 'SCRIPT' || parentTag === 'STYLE') continue
    const original = node.nodeValue
    const trimmed = original.trim()
    if (!trimmed) continue
    if (isAutoDomainKey(trimmed)) {
      const localized = t(trimmed)
      if (localized && localized !== trimmed) {
        node.nodeValue = original.replace(trimmed, localized)
      }
      continue
    }
  }
}

function translateAttributes(root) {
  const attrs = ['placeholder', 'title', 'aria-label']
  for (const attr of attrs) {
    root.querySelectorAll(`[${attr}]`).forEach((node) => {
      const value = node.getAttribute(attr)
      if (!value) return
      if (isAutoDomainKey(value.trim())) {
        const localized = t(value.trim())
        if (localized && localized !== value.trim()) {
          node.setAttribute(attr, localized)
        }
        return
      }
    })
  }
}

function isAutoDomainKey(value = '') {
  return /^domain\.auto\.[a-f0-9]+$/i.test(String(value).trim())
}
