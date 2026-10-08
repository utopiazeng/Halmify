#!/usr/bin/env node

import fs from 'fs'
import path from 'path'
import crypto from 'crypto'
import { parse } from '@vue/compiler-sfc'

const root = process.cwd()
const scopeDir = process.env.I18N_CODEMOD_SCOPE || 'src/views/domains'
const domainsDir = path.join(root, scopeDir)
const i18nFile = path.join(root, 'src/i18n.js')

function walk(dir, files = []) {
  const entries = fs.readdirSync(dir, { withFileTypes: true })
  for (const entry of entries) {
    if (entry.name.startsWith('.')) continue
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) walk(full, files)
    else if (entry.name.endsWith('.vue')) files.push(full)
  }
  return files
}

const files = walk(domainsDir)
const locales = ['zh-CN', 'zh-TW', 'en-US', 'es-ES', 'fr-FR', 'de-DE', 'ja-JP', 'ko-KR', 'pt-PT', 'ar-SA', 'ru-RU', 'it-IT']
const keyToText = new Map()

function hasChinese(text = '') {
  return /[\u3400-\u9FFF]/.test(text)
}

function normalizeText(text) {
  return text.replace(/\s+/g, ' ').trim()
}

function hashKey(text) {
  return crypto.createHash('sha1').update(text).digest('hex').slice(0, 10)
}

function toTraditional(value) {
  return value
    .replaceAll('线索', '線索')
    .replaceAll('联系人', '聯絡人')
    .replaceAll('导出', '匯出')
    .replaceAll('设置', '設定')
    .replaceAll('查询', '查詢')
    .replaceAll('状态', '狀態')
    .replaceAll('创建', '建立')
    .replaceAll('更新', '更新')
    .replaceAll('删除', '刪除')
    .replaceAll('页', '頁')
    .replaceAll('后', '後')
    .replaceAll('无', '無')
    .replaceAll('复', '復')
}

function keyFor(text) {
  const normalized = normalizeText(text)
  if (!normalized || !hasChinese(normalized)) return null
  const key = `domain.auto.${hashKey(normalized)}`
  if (!keyToText.has(key)) keyToText.set(key, normalized)
  return key
}

function ensureTImport(content) {
  const relativeI18n = path.relative(path.dirname(currentFile), i18nFile).replaceAll(path.sep, '/').replace(/\.js$/, '')
  const importPath = relativeI18n.startsWith('.') ? relativeI18n : `./${relativeI18n}`
  const escapedImportPath = importPath.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const importRe = new RegExp(`import\\s*\\{([^}]+)\\}\\s*from\\s*['"]${escapedImportPath}['"]`)
  if (importRe.test(content) && /\bt\b/.test(content.match(importRe)?.[1] || '')) return content
  if (importRe.test(content)) {
    return content.replace(importRe, (m, p1) => {
      if (p1.split(',').map((s) => s.trim()).includes('t')) return m
      return `import { ${p1.trim()}, t } from '${importPath}'`
    })
  }
  return content.replace(/(<script setup[^>]*>\n)/, `$1import { t } from '${importPath}'\n`)
}

function replaceAttr(template) {
  const attrs = [
    'label', 'placeholder', 'title', 'content', 'description', 'aria-label',
    'start-placeholder', 'end-placeholder', 'confirm-button-text', 'cancel-button-text'
  ]
  for (const attr of attrs) {
    const reDouble = new RegExp(`(\\s)${attr}="([^"\\n]*[\\u3400-\\u9FFF][^"\\n]*)"`, 'g')
    template = template.replace(reDouble, (_, ws, value) => {
      const key = keyFor(value)
      if (!key) return `${ws}${attr}="${value}"`
      return `${ws}:${attr}="t('${key}')"`
    })
    const reSingle = new RegExp(`(\\s)${attr}='([^'\\n]*[\\u3400-\\u9FFF][^'\\n]*)'`, 'g')
    template = template.replace(reSingle, (_, ws, value) => {
      const key = keyFor(value)
      if (!key) return `${ws}${attr}='${value}'`
      return `${ws}:${attr}="t('${key}')"`
    })
  }
  return template
}

function replaceStringLiterals(template) {
  template = template.replace(/'([^'\n]*[\u3400-\u9FFF][^'\n]*)'/g, (m, value) => {
    const key = keyFor(value)
    return key ? `t('${key}')` : m
  })
  template = template.replace(/"([^"\n]*[\u3400-\u9FFF][^"\n]*)"/g, (m, value) => {
    const key = keyFor(value)
    return key ? `t('${key}')` : m
  })
  return template
}

function replaceTextNodes(template) {
  return template.replace(/>([^<]*[\u3400-\u9FFF][^<]*)</g, (m, inner) => {
    const parts = inner.split(/(\{\{[\s\S]*?\}\})/g)
    let changed = false
    const next = parts.map((part) => {
      if (/^\{\{[\s\S]*\}\}$/.test(part)) return part
      return part.replace(/([^{}<>]*[\u3400-\u9FFF][^{}<>]*)/g, (chunk) => {
        const trimmed = chunk.trim()
        const key = keyFor(trimmed)
        if (!key) return chunk
        changed = true
        const leading = chunk.match(/^\s*/)?.[0] || ''
        const trailing = chunk.match(/\s*$/)?.[0] || ''
        return `${leading}{{ t('${key}') }}${trailing}`
      })
    }).join('')
    return changed ? `>${next}<` : m
  })
}

let currentFile = ''
function processVue(file) {
  currentFile = file
  let content = fs.readFileSync(file, 'utf8')
  const descriptor = parse(content, { filename: file }).descriptor
  if (!descriptor.template) return
  const tplStart = descriptor.template.loc.start.offset
  const tplEnd = descriptor.template.loc.end.offset
  let tpl = content.slice(tplStart, tplEnd)
  tpl = replaceAttr(tpl)
  tpl = replaceStringLiterals(tpl)
  tpl = replaceTextNodes(tpl)
  content = `${content.slice(0, tplStart)}${tpl}${content.slice(tplEnd)}`
  content = ensureTImport(content)
  writeFileReplace(file, content)
}

for (const file of files) processVue(file)

let i18n = fs.readFileSync(i18nFile, 'utf8')
const begin = '// AUTO_I18N_DOMAIN_BEGIN'
const end = '// AUTO_I18N_DOMAIN_END'
const idxBegin = i18n.indexOf(begin)
const idxEnd = i18n.indexOf(end)
const oldMap = new Map()
if (idxBegin >= 0 && idxEnd > idxBegin) {
  const block = i18n.slice(idxBegin, idxEnd)
  const re = /"(domain\.auto\.[a-f0-9]{10})"\s*:\s*"([^"]*)"/g
  let m
  while ((m = re.exec(block)) !== null) oldMap.set(m[1], m[2])
  i18n = i18n.slice(0, idxBegin) + i18n.slice(idxEnd + end.length)
}

for (const [key, text] of oldMap.entries()) if (!keyToText.has(key)) keyToText.set(key, text)
const sorted = [...keyToText.entries()].sort((a, b) => a[0].localeCompare(b[0]))
const zhMap = Object.fromEntries(sorted)
const twMap = Object.fromEntries(sorted.map(([k, v]) => [k, toTraditional(v)]))
const enMap = Object.fromEntries(sorted)

const block = `\n${begin}\nconst autoDomainI18n = {\n  'zh-CN': ${JSON.stringify(zhMap, null, 2)},\n  'zh-TW': ${JSON.stringify(twMap, null, 2)},\n  'en-US': ${JSON.stringify(enMap, null, 2)}\n}\nfor (const locale of ${JSON.stringify(locales)}) {\n  const fallback = autoDomainI18n['en-US']\n  const localized = autoDomainI18n[locale] || fallback\n  messages[locale] = { ...(messages[locale] || fallback), ...fallback, ...localized }\n}\n${end}\n`

const anchor = "messages['en-US']['app.platformCapabilities'] = 'Platform'"
const pos = i18n.indexOf(anchor)
if (pos > 0) {
  i18n = i18n.slice(0, pos) + block + i18n.slice(pos)
} else {
  i18n += block
}

writeFileReplace(i18nFile, i18n)
console.log(`processed ${files.length} files, keys=${sorted.length}`)

function writeFileReplace(file, content) {
  const tmp = `${file}.i18n-tmp-${process.pid}`
  fs.writeFileSync(tmp, content)
  fs.renameSync(tmp, file)
}
