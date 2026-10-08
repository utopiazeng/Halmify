#!/usr/bin/env node

import fs from 'fs'
import path from 'path'
import crypto from 'crypto'

const root = process.cwd()
const scopeDir = process.env.I18N_CODEMOD_SCOPE || 'src/views/domains'
const domainsDir = path.join(root, scopeDir)
const i18nFile = path.join(root, 'src/i18n.js')
const locales = ['zh-CN', 'zh-TW', 'en-US', 'es-ES', 'fr-FR', 'de-DE', 'ja-JP', 'ko-KR', 'pt-PT', 'ar-SA', 'ru-RU', 'it-IT']

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
const keyToText = new Map()

function hasChinese(text = '') {
  return /[\u3400-\u9FFF]/.test(text)
}

function normalizeText(text = '') {
  return text.replace(/\s+/g, ' ').trim()
}

function hashKey(text) {
  return crypto.createHash('sha1').update(text).digest('hex').slice(0, 10)
}

function keyFor(text) {
  const normalized = normalizeText(text)
  if (!normalized || !hasChinese(normalized)) return null
  const key = `domain.auto.${hashKey(normalized)}`
  if (!keyToText.has(key)) keyToText.set(key, normalized)
  return key
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
    .replaceAll('删除', '刪除')
    .replaceAll('页', '頁')
    .replaceAll('后', '後')
    .replaceAll('无', '無')
    .replaceAll('复', '復')
}

function ensureTImport(content) {
  if (/import\s*\{[^}]*\bt\b[^}]*\}\s*from\s*['"]\.\.\/\.\.\/i18n['"]/.test(content)) return content
  if (/from\s*['"]\.\.\/\.\.\/i18n['"]/.test(content)) {
    return content.replace(/import\s*\{([^}]+)\}\s*from\s*['"]\.\.\/\.\.\/i18n['"]/, (m, p1) => {
      const parts = p1.split(',').map((s) => s.trim()).filter(Boolean)
      if (!parts.includes('t')) parts.push('t')
      return `import { ${parts.join(', ')} } from '../../i18n'`
    })
  }
  return content.replace(/(<script setup[^>]*>\n)/, `$1import { t } from '../../i18n'\n`)
}

function replaceStringLiteralsInExpression(expr) {
  if (!hasChinese(expr)) return expr
  return expr
    .replace(/'([^'\n]*[\u3400-\u9FFF][^'\n]*)'/g, (m, val) => {
      const key = keyFor(val)
      return key ? `t('${key}')` : m
    })
    .replace(/"([^"\n]*[\u3400-\u9FFF][^"\n]*)"/g, (m, val) => {
      const key = keyFor(val)
      return key ? `t('${key}')` : m
    })
}

function replaceTemplate(template) {
  let out = template

  const attrs = [
    'label', 'placeholder', 'title', 'content', 'description', 'aria-label',
    'start-placeholder', 'end-placeholder', 'confirm-button-text', 'cancel-button-text'
  ]

  for (const attr of attrs) {
    const reDouble = new RegExp(`(\\s)${attr}="([^"]*[\\u3400-\\u9FFF][^"]*)"`, 'g')
    out = out.replace(reDouble, (_, ws, value) => {
      const key = keyFor(value)
      return key ? `${ws}:${attr}="t('${key}')"` : `${ws}${attr}="${value}"`
    })
    const reSingle = new RegExp(`(\\s)${attr}='([^']*[\\u3400-\\u9FFF][^']*)'`, 'g')
    out = out.replace(reSingle, (_, ws, value) => {
      const key = keyFor(value)
      return key ? `${ws}:${attr}="t('${key}')"` : `${ws}${attr}='${value}'`
    })
  }

  // {{ ... }}
  out = out.replace(/\{\{([\s\S]*?)\}\}/g, (m, expr) => `{{${replaceStringLiteralsInExpression(expr)}}}`)

  // Directive expressions like :foo="a ? '中文' : 'x'" or @click="..."
  out = out.replace(/([:@][\w-]+(?:\.[\w-]+)?)="([^"]*[\u3400-\u9FFF][^"]*)"/g, (m, name, expr) => {
    return `${name}="${replaceStringLiteralsInExpression(expr)}"`
  })

  // plain text nodes
  out = out.replace(/>([^<]*[\u3400-\u9FFF][^<]*)</g, (m, text) => {
    if (/\{\{[\s\S]*\}\}/.test(text)) return m
    const key = keyFor(text)
    if (!key) return m
    const leading = text.match(/^\s*/)?.[0] || ''
    const trailing = text.match(/\s*$/)?.[0] || ''
    return `>${leading}{{ t('${key}') }}${trailing}<`
  })

  return out
}

function mergeAutoBlock(i18n, newMap) {
  const begin = '// AUTO_I18N_DOMAIN_BEGIN'
  const end = '// AUTO_I18N_DOMAIN_END'
  const b = i18n.indexOf(begin)
  const e = i18n.indexOf(end)

  const oldMap = new Map()
  if (b >= 0 && e > b) {
    const block = i18n.slice(b, e)
    const re = /"(domain\.auto\.[a-f0-9]{10})"\s*:\s*"([^"]*)"/g
    let m
    while ((m = re.exec(block)) !== null) {
      oldMap.set(m[1], m[2])
    }
    i18n = i18n.slice(0, b) + i18n.slice(e + end.length)
  }

  for (const [k, v] of oldMap.entries()) {
    if (!newMap.has(k)) newMap.set(k, v)
  }

  const sorted = [...newMap.entries()].sort((a, b) => a[0].localeCompare(b[0]))
  const zh = Object.fromEntries(sorted)
  const tw = Object.fromEntries(sorted.map(([k, v]) => [k, toTraditional(v)]))
  const en = Object.fromEntries(sorted)

  const block = `\n${begin}\nconst autoDomainI18n = {\n  'zh-CN': ${JSON.stringify(zh, null, 2)},\n  'zh-TW': ${JSON.stringify(tw, null, 2)},\n  'en-US': ${JSON.stringify(en, null, 2)}\n}\nfor (const locale of ${JSON.stringify(locales)}) {\n  const fallback = autoDomainI18n['en-US']\n  const localized = autoDomainI18n[locale] || fallback\n  messages[locale] = { ...(messages[locale] || fallback), ...fallback, ...localized }\n}\n${end}\n`

  const anchor = "messages['en-US']['app.platformCapabilities'] = 'Platform'"
  const pos = i18n.indexOf(anchor)
  if (pos > 0) return i18n.slice(0, pos) + block + i18n.slice(pos)
  return i18n + block
}

let updated = 0
for (const file of files) {
  let content = fs.readFileSync(file, 'utf8')
  const m = content.match(/<template>[\s\S]*?<\/template>/)
  if (!m) continue
  const nextTpl = replaceTemplate(m[0])
  if (nextTpl !== m[0]) {
    content = content.replace(m[0], nextTpl)
    updated += 1
  }
  content = ensureTImport(content)
  fs.writeFileSync(file, content)
}

let i18n = fs.readFileSync(i18nFile, 'utf8')
i18n = mergeAutoBlock(i18n, keyToText)
fs.writeFileSync(i18nFile, i18n)
console.log(`safe codemod updated ${updated} files; keys ${keyToText.size}`)
