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
    .replaceAll('输单', '輸單')
    .replaceAll('赢单', '贏單')
    .replaceAll('报价', '報價')
    .replaceAll('转派', '轉派')
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

function isLikelyObjectKey(script, start, end) {
  let i = start - 1
  while (i >= 0 && /\s/.test(script[i])) i -= 1
  let j = end + 1
  while (j < script.length && /\s/.test(script[j])) j += 1
  const prev = i >= 0 ? script[i] : ''
  const next = j < script.length ? script[j] : ''
  return next === ':' && (prev === '{' || prev === ',' || prev === '\n')
}

function replaceQuotedChinese(script) {
  let out = ''
  let i = 0
  while (i < script.length) {
    const ch = script[i]
    if (ch !== '\'' && ch !== '"') {
      out += ch
      i += 1
      continue
    }

    const quote = ch
    let j = i + 1
    let escaped = false
    while (j < script.length) {
      const c = script[j]
      if (!escaped && c === '\\') { escaped = true; j += 1; continue }
      if (!escaped && c === quote) break
      escaped = false
      j += 1
    }

    if (j >= script.length) {
      out += script.slice(i)
      break
    }

    const raw = script.slice(i + 1, j)
    const full = script.slice(i, j + 1)
    if (!hasChinese(raw) || isLikelyObjectKey(script, i, j)) {
      out += full
      i = j + 1
      continue
    }

    const key = keyFor(raw)
    out += key ? `t('${key}')` : full
    i = j + 1
  }
  return out
}

function replaceTemplateLiteralContent(text) {
  if (!hasChinese(text)) return text
  const key = keyFor(text)
  return key ? `\${t('${key}')}` : text
}

function convertTemplateLiteral(inner) {
  if (!hasChinese(inner)) return `\`${inner}\``
  const parts = []
  let i = 0
  while (i < inner.length) {
    const start = inner.indexOf('${', i)
    if (start < 0) {
      parts.push({ type: 'text', value: inner.slice(i) })
      break
    }
    if (start > i) parts.push({ type: 'text', value: inner.slice(i, start) })
    let depth = 1
    let j = start + 2
    while (j < inner.length && depth > 0) {
      if (inner[j] === '{') depth += 1
      else if (inner[j] === '}') depth -= 1
      j += 1
    }
    parts.push({ type: 'expr', value: inner.slice(start, j) })
    i = j
  }

  const converted = parts.map((part) => {
    if (part.type === 'expr') return part.value
    return replaceTemplateLiteralContent(part.value)
  }).join('')

  return `\`${converted}\``
}

function replaceBacktickChinese(script) {
  let out = ''
  let i = 0
  while (i < script.length) {
    const ch = script[i]
    if (ch !== '`') {
      out += ch
      i += 1
      continue
    }

    let j = i + 1
    let escaped = false
    while (j < script.length) {
      const c = script[j]
      if (!escaped && c === '\\') { escaped = true; j += 1; continue }
      if (!escaped && c === '`') break
      escaped = false
      j += 1
    }

    if (j >= script.length) {
      out += script.slice(i)
      break
    }

    const inner = script.slice(i + 1, j)
    out += convertTemplateLiteral(inner)
    i = j + 1
  }
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
    while ((m = re.exec(block)) !== null) oldMap.set(m[1], m[2])
    i18n = i18n.slice(0, b) + i18n.slice(e + end.length)
  }

  for (const [k, v] of oldMap.entries()) if (!newMap.has(k)) newMap.set(k, v)

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

let changedFiles = 0
for (const file of files) {
  let content = fs.readFileSync(file, 'utf8')
  const m = content.match(/<script setup[^>]*>[\s\S]*?<\/script>/)
  if (!m) continue

  let script = m[0]
  const next1 = replaceBacktickChinese(script)
  const next2 = replaceQuotedChinese(next1)
  const next = next2

  if (next !== script) {
    content = content.replace(script, next)
    changedFiles += 1
  }

  content = ensureTImport(content)
  writeFileReplace(file, content)
}

let i18n = fs.readFileSync(i18nFile, 'utf8')
i18n = mergeAutoBlock(i18n, keyToText)
writeFileReplace(i18nFile, i18n)

console.log(`script codemod changed ${changedFiles} files; new keys ${keyToText.size}`)

function writeFileReplace(file, content) {
  const tmp = `${file}.i18n-tmp-${process.pid}`
  fs.writeFileSync(tmp, content)
  fs.renameSync(tmp, file)
}
