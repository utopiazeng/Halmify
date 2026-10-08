#!/usr/bin/env node

import fs from 'fs'
import path from 'path'
import { localizeGeneratedText, strictLocales } from '../src/i18n-strict-locales.js'

const args = new Set(process.argv.slice(2))
const failOnHardcoded = args.has('--fail-on-hardcoded') || process.env.I18N_SCAN_FAIL_ON_HARDCODED === '1'
const onlyDomains = args.has('--domains-only')
const allSrc = args.has('--all-src')

const root = path.resolve(process.cwd(), onlyDomains ? 'src/views/domains' : allSrc ? 'src' : 'src/views')
const ignoreDirs = new Set(['__tests__', '__mocks__', 'assets'])
const ignoreFiles = new Set(['src/i18n.js', 'src/i18n-dom.js', 'src/i18n-strict-locales.js', 'src/router/routes.js', 'src/utils/confirm.js'])
const textExt = new Set(['.vue', '.js', '.ts'])
const dynamicTranslationEventKeys = new Set([
  'domain.key',
  'domain.workspace.domain.workspace.title',
  'im.message.new',
  'im.message.ack',
  'im.conversation.unread',
  'im.presence.changed',
  'im.system.unread'
])
const simplifiedChinesePattern = /(密码|请输入|请选择|状态|删除|启用|暂无|导出|导入|线索|联系人|项目|订单|商机|记录|数据|消息|文件|登录|验证|链接|选择|输入|报价|履约|售后|模块|页面|说明|视频|资料|报告|线上|会议|广告|字段|统计|归集|归档|来源|宣传|报名|活动|手动|自动|阶段|维护|产品|明细|建议|编码|权限|点击|这里|确认|开票|登记|发货|账号|用户名|生成|报价单|服务|运营|动作|口径|工单)/
const genericFallbackPattern = /^(Details|Detalles|Détails|التفاصيل|Сведения|Dettagli|詳細|상세|詳情|Access settings|Admin settings|Lead handling|Deal progress|Order finance|Campaign results|Project progress|Contact details|Customer health|Ticket handling|Product inventory)$/i
const pendingTranslationPattern = /^__PENDING_TRANSLATION__:/i
const forbiddenVisiblePattern = /(Create Details|Details name|Order finance Details|PublicOpportunity|PrivateOpportunity|BatchDeleteInvalidLateLead|Interface text|CRM interface text|UI text)/
const codeLikeFallbackPattern = /(^domain auto [a-f0-9]+$|^domain\.auto\.[a-f0-9]+$|^[a-z]+ [a-z]+ [a-f0-9]{4,}$)/i
const genericFieldFallbackByLocale = {
  'en-US': new Set(['Field', 'Info', 'Object', 'Item']),
  'es-ES': new Set(['Campo', 'Elemento']),
  'fr-FR': new Set(['Champ', 'Élément', 'Element']),
  'de-DE': new Set(['Feld', 'Element']),
  'ja-JP': new Set(['フィールド', '項目']),
  'ko-KR': new Set(['필드', '항목']),
  'pt-PT': new Set(['Campo', 'Item', 'Elemento']),
  'ar-SA': new Set(['حقل', 'عنصر']),
  'ru-RU': new Set(['Поле', 'Элемент']),
  'it-IT': new Set(['Campo', 'Elemento'])
}

function walk(dir, files = []) {
  const entries = fs.readdirSync(dir, { withFileTypes: true })
  for (const entry of entries) {
    if (entry.name.startsWith('.')) continue
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      if (!ignoreDirs.has(entry.name)) walk(full, files)
      continue
    }
    if (textExt.has(path.extname(entry.name))) files.push(full)
  }
  return files
}

function stripComments(content) {
  return content
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/\/\/.*$/gm, '')
    .replace(/<!--[\s\S]*?-->/g, '')
}

function countChineseToken(source) {
  const matches = source.match(/[\u3400-\u9FFF]+/g)
  return matches ? matches.length : 0
}

function splitVue(content) {
  const templateMatch = content.match(/<template>[\s\S]*?<\/template>/)
  const scriptMatch = content.match(/<script(?:\s+setup)?[^>]*>[\s\S]*?<\/script>/)
  return {
    template: templateMatch ? templateMatch[0] : '',
    script: scriptMatch ? scriptMatch[0] : ''
  }
}

if (!fs.existsSync(root)) {
  console.error(`[i18n-scan] views path not found: ${root}`)
  process.exit(1)
}

const files = walk(root)
const stats = []
const missingTImports = []
const referencedKeys = new Set()
let total = 0
let totalTemplate = 0
let totalScript = 0

for (const file of files) {
  const raw = fs.readFileSync(file, 'utf8')
  const relative = path.relative(process.cwd(), file)
  if (ignoreFiles.has(relative)) continue
  collectReferencedKeys(raw, referencedKeys)
  if (usesI18nT(raw) && !declaresLocalT(raw) && !importsI18nT(raw)) {
    missingTImports.push(relative)
  }

  if (file.endsWith('.vue')) {
    const script = splitVue(raw).script
    const scriptCount = countChineseToken(stripComments(script))
    const templateCount = Math.max(0, countChineseToken(stripComments(raw)) - scriptCount)
    const count = templateCount + scriptCount

    if (count > 0) {
      stats.push({ file: relative, count, templateCount, scriptCount })
      total += count
      totalTemplate += templateCount
      totalScript += scriptCount
    }
    continue
  }

  const count = countChineseToken(stripComments(raw))
  if (count > 0) {
    stats.push({ file: relative, count, templateCount: 0, scriptCount: count })
    total += count
    totalScript += count
  }
}

stats.sort((a, b) => b.count - a.count)

console.log(`[i18n-scan] scope: ${onlyDomains ? 'src/views/domains' : allSrc ? 'src' : 'src/views'}`)
console.log(`[i18n-scan] total hardcoded Chinese tokens: ${total}`)
console.log(`[i18n-scan] template tokens: ${totalTemplate}`)
console.log(`[i18n-scan] script tokens: ${totalScript}`)
console.log(`[i18n-scan] files with hardcoded Chinese: ${stats.length}`)
console.log(`[i18n-scan] files using t() without importing it: ${missingTImports.length}`)

const coverage = checkCoverage(referencedKeys)
console.log(`[i18n-scan] referenced translation keys: ${referencedKeys.size}`)
console.log(`[i18n-scan] missing locale entries: ${coverage.missing.length}`)
console.log(`[i18n-scan] invalid locale entries: ${coverage.invalid.length}`)

for (const row of stats.slice(0, 30)) {
  console.log(`- ${row.file}: total=${row.count}, template=${row.templateCount}, script=${row.scriptCount}`)
}

for (const file of missingTImports.slice(0, 30)) {
  console.log(`- ${file}: missing import { t } from i18n`)
}

for (const row of coverage.missing.slice(0, 30)) {
  console.log(`- ${row.key}: missing ${row.locale}`)
}

for (const row of coverage.invalid.slice(0, 30)) {
  console.log(`- ${row.key}: invalid ${row.locale} -> ${row.reason}`)
}

if (failOnHardcoded && (total > 0 || missingTImports.length > 0 || coverage.missing.length > 0 || coverage.invalid.length > 0)) {
  console.error('[i18n-scan] fail-on-hardcoded is enabled and i18n issues remain.')
  process.exit(1)
}

function usesI18nT(source) {
  return /\bt\s*\(/.test(stripComments(source))
}

function declaresLocalT(source) {
  return /function\s+t\s*\(|(?:const|let|var)\s+t\s*=/.test(stripComments(source))
}

function importsI18nT(source) {
  return /import\s*\{[^}]*\bt\b[^}]*\}\s*from\s*['"][^'"]*i18n['"]/s.test(source)
}

function collectReferencedKeys(source, target) {
  const clean = stripComments(source)
  const re = /['"]((?:domain\.auto|common|app|login|operation|nav|navDesc|menuGroup|domain|system|platform|im|files|campaign)\.[A-Za-z0-9_.:\-\u3400-\u9FFF]+)['"]/g
  for (const match of clean.matchAll(re)) {
    if (dynamicTranslationEventKeys.has(match[1])) continue
    target.add(match[1])
  }
}

function checkCoverage(keys) {
  const i18nSource = fs.readFileSync(path.resolve(process.cwd(), 'src/i18n.js'), 'utf8')
  const localeMessages = extractLocaleMessages(i18nSource)
  const missing = []
  const invalid = []
  const nonHanLocales = new Set(['en-US', 'es-ES', 'fr-FR', 'de-DE', 'pt-PT', 'ar-SA', 'ru-RU', 'it-IT'])
  const zhAutoSource = localeMessages['zh-CN'] || {}

  for (const key of keys) {
    for (const locale of strictLocales) {
      let value = localeMessages[locale]?.[key]
      if (locale === 'zh-TW' && value !== undefined && simplifiedChinesePattern.test(String(value))) {
        value = localizeGeneratedText(zhAutoSource[key] || value, locale, key)
      }
      if (value !== undefined && nonHanLocales.has(locale) && /[\u3400-\u9FFF]/.test(String(value))) {
        value = localizeGeneratedText(zhAutoSource[key] || value, locale, key)
      }
      if ((value === undefined || value === null || value === key) && zhAutoSource[key] !== undefined) {
        value = localizeGeneratedText(zhAutoSource[key], locale, key)
      }
      if (key.startsWith('domain.auto.') && (value === undefined || value === null || value === key)) {
        value = localizeGeneratedText(zhAutoSource[key] || key, locale, key)
      }
      if (value === undefined || value === null || String(value).trim() === '') {
        missing.push({ key, locale })
        continue
      }
      const text = String(value).trim()
      if (text === key || /^domain\.auto\.[a-f0-9]+$/i.test(text)) {
        invalid.push({ key, locale, reason: 'unresolved key' })
        continue
      }
      if (codeLikeFallbackPattern.test(text)) {
        invalid.push({ key, locale, reason: 'code-like fallback' })
        continue
      }
      if (/^(Details|Detalles|Détails|التفاصيل|Сведения|Dettagli|詳細|상세|詳情)\s+[a-f0-9]{4}$/i.test(text)) {
        invalid.push({ key, locale, reason: 'leaked fallback key' })
        continue
      }
      if (genericFallbackPattern.test(text) || forbiddenVisiblePattern.test(text)) {
        invalid.push({ key, locale, reason: 'generic fallback' })
        continue
      }
      if (pendingTranslationPattern.test(text)) {
        invalid.push({ key, locale, reason: 'pending translation' })
        continue
      }
      if (genericFieldFallbackByLocale[locale]?.has(text)) {
        invalid.push({ key, locale, reason: 'generic field fallback' })
        continue
      }
      if (nonHanLocales.has(locale) && /[\u3400-\u9FFF]/.test(text)) {
        invalid.push({ key, locale, reason: 'contains Han characters' })
        continue
      }
      if (locale === 'zh-TW' && simplifiedChinesePattern.test(text)) {
        invalid.push({ key, locale, reason: 'contains simplified Chinese wording' })
      }
    }
  }
  return { missing, invalid }
}

function extractLocaleMessages(source) {
  const messages = Object.fromEntries(strictLocales.map((locale) => [locale, {}]))
  for (const locale of strictLocales) {
    for (const block of findLocaleObjectBlocks(source, locale)) {
      Object.assign(messages[locale], extractPairs(block))
    }
  }
  const assignRe = /messages\[['"](zh-CN|zh-TW|en-US|es-ES|fr-FR|de-DE|ja-JP|ko-KR|pt-PT|ar-SA|ru-RU|it-IT)['"]\]\[['"]([^'"]+)['"]\]\s*=\s*['"]((?:\\.|[^'"])*)['"]/g
  for (const match of source.matchAll(assignRe)) {
    messages[match[1]][match[2]] = unescapeString(match[3])
  }
  Object.assign(messages, extractLayeredKeyMessages(source, messages))
  Object.assign(messages, extractNamedI18nObject(source, messages, 'compactAutoOverrides'))
  Object.assign(messages, extractNamedI18nObject(source, messages, 'businessDataDisplayI18n'))
  for (const name of ['dashboardMessages', 'iotMessages', 'securityMessages', 'tenantSettingsMessages', 'uiFixMessages', 'crmUiFixMessages', 'orderFixMessages', 'errorMessages', 'industryMessages']) {
    Object.assign(messages, extractImportedMessagePack(source, messages, name))
  }
  Object.assign(messages, extractTranslateMarkdownMessages(messages))
  return messages
}

function extractTranslateMarkdownMessages(messages) {
  const next = Object.fromEntries(Object.entries(messages).map(([locale, map]) => [locale, { ...map }]))
  const file = path.resolve(process.cwd(), 'src/translate.md')
  if (!fs.existsSync(file)) return next
  const markdown = fs.readFileSync(file, 'utf8')
  const match = markdown.match(/```json\s*([\s\S]*?)```/i)
  try {
    const pack = JSON.parse(match ? match[1] : markdown)
    for (const locale of strictLocales) {
      if (pack.messages?.[locale]) next[locale] = { ...next[locale], ...pack.messages[locale] }
    }
  } catch {
    // Syntax errors are reported by runtime/build checks; keep this scanner non-throwing.
  }
  return next
}

function extractLayeredKeyMessages(source, messages) {
  const next = Object.fromEntries(Object.entries(messages).map(([locale, map]) => [locale, { ...map }]))
  const marker = 'const layeredKeys ='
  const start = source.indexOf(marker)
  if (start < 0) return next
  const openIndex = source.indexOf('{', start)
  const closeIndex = findMatchingBrace(source, openIndex)
  if (openIndex < 0 || closeIndex < 0) return next
  const block = source.slice(openIndex + 1, closeIndex)
  for (const entry of extractObjectEntries(block)) {
    const key = entry.key
    const values = extractPairs(entry.value)
    for (const [locale, value] of Object.entries(values)) {
      if (next[locale]) next[locale][key] = value
    }
  }
  return next
}

function extractNamedI18nObject(source, messages, objectName) {
  const next = Object.fromEntries(Object.entries(messages).map(([locale, map]) => [locale, { ...map }]))
  const marker = `const ${objectName} =`
  const start = source.indexOf(marker)
  if (start < 0) return next
  const openIndex = source.indexOf('{', start)
  const closeIndex = findMatchingBrace(source, openIndex)
  if (openIndex < 0 || closeIndex < 0) return next
  const block = source.slice(openIndex + 1, closeIndex)
  for (const entry of extractObjectEntries(block)) {
    const values = extractPairs(entry.value)
    for (const [locale, value] of Object.entries(values)) {
      if (next[locale]) next[locale][entry.key] = value
    }
  }
  return next
}

function extractImportedMessagePack(source, messages, importName) {
  const next = Object.fromEntries(Object.entries(messages).map(([locale, map]) => [locale, { ...map }]))
  const importRe = new RegExp(`import\\s+${escapeRegExp(importName)}\\s+from\\s+['"]([^'"]+)['"]`)
  const match = source.match(importRe)
  if (!match) return next
  const relativePath = match[1].endsWith('.js') ? match[1] : `${match[1]}.js`
  const file = path.resolve(process.cwd(), 'src', relativePath.replace(/^\.\//, ''))
  if (!fs.existsSync(file)) return next
  const packSource = fs.readFileSync(file, 'utf8')
  for (const locale of strictLocales) {
    for (const block of findLocaleObjectBlocks(packSource, locale)) {
      Object.assign(next[locale], extractPairs(block))
    }
  }
  return next
}

function extractObjectEntries(block) {
  const entries = []
  const keyRe = /['"]([^'"]+)['"]\s*:\s*\{/g
  for (const match of block.matchAll(keyRe)) {
    const openIndex = block.indexOf('{', match.index)
    const closeIndex = findMatchingBrace(block, openIndex)
    if (openIndex < 0 || closeIndex < 0) continue
    entries.push({
      key: match[1],
      value: block.slice(openIndex + 1, closeIndex)
    })
    keyRe.lastIndex = closeIndex + 1
  }
  return entries
}

function findLocaleObjectBlocks(source, locale) {
  const blocks = []
  const patterns = [
    new RegExp(`['"]${escapeRegExp(locale)}['"]\\s*:\\s*\\{`, 'g'),
    new RegExp(`messages\\[['"]${escapeRegExp(locale)}['"]\\]\\s*=\\s*\\{`, 'g')
  ]
  for (const re of patterns) {
    for (const match of source.matchAll(re)) {
      const openIndex = source.indexOf('{', match.index)
      const closeIndex = findMatchingBrace(source, openIndex)
      if (openIndex >= 0 && closeIndex > openIndex) {
        blocks.push(source.slice(openIndex + 1, closeIndex))
      }
    }
  }
  return blocks
}

function findMatchingBrace(source, openIndex) {
  let depth = 0
  let quote = ''
  let escaped = false
  for (let i = openIndex; i < source.length; i += 1) {
    const ch = source[i]
    if (quote) {
      if (!escaped && ch === '\\') {
        escaped = true
        continue
      }
      if (!escaped && ch === quote) quote = ''
      escaped = false
      continue
    }
    if (ch === '"' || ch === "'" || ch === '`') {
      quote = ch
      continue
    }
    if (ch === '{') depth += 1
    if (ch === '}') {
      depth -= 1
      if (depth === 0) return i
    }
  }
  return -1
}

function escapeRegExp(value) {
  return String(value).replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function extractPairs(block) {
  const result = {}
  const pairRe = /['"]([^'"]+)['"]\s*:\s*['"]((?:\\.|[^'"])*)['"]/g
  for (const match of block.matchAll(pairRe)) {
    result[match[1]] = unescapeString(match[2])
  }
  return result
}

function unescapeString(value) {
  return String(value)
    .replace(/\\n/g, '\n')
    .replace(/\\'/g, "'")
    .replace(/\\"/g, '"')
    .replace(/\\\\/g, '\\')
}
