#!/usr/bin/env node

import fs from 'fs'
import path from 'path'
import { pathToFileURL } from 'url'
import { strictLocales } from '../src/i18n-strict-locales.js'

const args = new Set(process.argv.slice(2))
const checkOnly = args.has('--check')
const reportOnly = args.has('--report')
const prune = args.has('--prune')
const noTranslate = args.has('--no-translate') || checkOnly || reportOnly
const root = process.cwd()
const translateFile = path.resolve(root, 'src/translate.md')
const scanRoots = ['src/views', 'src/components', 'src/router', 'src/utils', 'src/App.vue', 'src/main.js']
const textExt = new Set(['.vue', '.js', '.ts'])
const ignoreDirs = new Set(['assets', '__tests__', '__mocks__'])
const protectedAcronyms = new Set([
  'AI', 'AICRM', 'API', 'CRM', 'ROI', 'UTM', 'SLA', 'SDR', 'SQL', 'MQL', 'CAC', 'CPL',
  'ABM', 'MFA', 'URL', 'ID', 'VIP', 'VAT', 'SKU', 'PDF', 'CSV', 'IM', 'ACL'
])
const targetLangMap = {
  'zh-TW': 'zh-TW',
  'en-US': 'en',
  'es-ES': 'es',
  'fr-FR': 'fr',
  'de-DE': 'de',
  'ja-JP': 'ja',
  'ko-KR': 'ko',
  'pt-PT': 'pt',
  'ar-SA': 'ar',
  'ru-RU': 'ru',
  'it-IT': 'it'
}
const pendingPrefix = '__PENDING_TRANSLATION__:'
const missingMarkers = new Set([
  'Missing translation',
  '翻译缺失',
  '翻譯缺失',
  'Falta traducción',
  'Traduction manquante',
  'Fehlende Übersetzung',
  '翻訳がありません',
  '번역 없음',
  'Tradução em falta',
  'الترجمة مفقودة',
  'Отсутствует перевод',
  'Traduzione mancante'
])
const genericFallbackValues = new Set([
  'Label',
  'Content',
  'Interface text',
  'CRM interface text',
  'Details',
  'Admin settings',
  'Lead handling',
  'Business information',
  'No data available',
  'Operation completed',
  'Operation failed'
])
const localTerms = {
  线索: {
    'zh-TW': '線索', 'en-US': 'Lead', 'es-ES': 'Lead', 'fr-FR': 'Prospect', 'de-DE': 'Lead',
    'ja-JP': 'リード', 'ko-KR': '리드', 'pt-PT': 'Lead', 'ar-SA': 'عميل محتمل',
    'ru-RU': 'Лид', 'it-IT': 'Lead'
  },
  商机: {
    'zh-TW': '商機', 'en-US': 'Opportunity', 'es-ES': 'Opportunity', 'fr-FR': 'Opportunité', 'de-DE': 'Chance',
    'ja-JP': '商談', 'ko-KR': '영업기회', 'pt-PT': 'Oportunidade', 'ar-SA': 'فرصة',
    'ru-RU': 'Возможность', 'it-IT': 'Opportunità'
  },
  市场活动: {
    'zh-TW': '市場活動', 'en-US': 'Campaign', 'es-ES': 'Campaña', 'fr-FR': 'Campagne', 'de-DE': 'Kampagne',
    'ja-JP': 'キャンペーン', 'ko-KR': '캠페인', 'pt-PT': 'Campanha', 'ar-SA': 'حملة',
    'ru-RU': 'Кампания', 'it-IT': 'Campagna'
  },
  成交记录: {
    'zh-TW': '成交記錄', 'en-US': 'Deal', 'es-ES': 'Deal', 'fr-FR': 'Affaire', 'de-DE': 'Deal',
    'ja-JP': '成約記録', 'ko-KR': '거래 기록', 'pt-PT': 'Negócio', 'ar-SA': 'صفقة',
    'ru-RU': 'Сделка', 'it-IT': 'Accordo'
  },
  订单: {
    'zh-TW': '訂單', 'en-US': 'Order', 'es-ES': 'Pedido', 'fr-FR': 'Commande', 'de-DE': 'Auftrag',
    'ja-JP': '注文', 'ko-KR': '주문', 'pt-PT': 'Encomenda', 'ar-SA': 'طلب',
    'ru-RU': 'Заказ', 'it-IT': 'Ordine'
  },
  回款: {
    'zh-TW': '回款', 'en-US': 'Payment', 'es-ES': 'Pago', 'fr-FR': 'Paiement', 'de-DE': 'Zahlung',
    'ja-JP': '入金', 'ko-KR': '수금', 'pt-PT': 'Pagamento', 'ar-SA': 'التحصيل',
    'ru-RU': 'Платеж', 'it-IT': 'Pagamento'
  },
  工单: {
    'zh-TW': '工單', 'en-US': 'Ticket', 'es-ES': 'Ticket', 'fr-FR': 'Ticket', 'de-DE': 'Ticket',
    'ja-JP': 'チケット', 'ko-KR': '티켓', 'pt-PT': 'Ticket', 'ar-SA': 'تذكرة',
    'ru-RU': 'Заявка', 'it-IT': 'Ticket'
  },
  客户: {
    'zh-TW': '客戶', 'en-US': 'Customer', 'es-ES': 'Cliente', 'fr-FR': 'Client', 'de-DE': 'Kunde',
    'ja-JP': '顧客', 'ko-KR': '고객', 'pt-PT': 'Cliente', 'ar-SA': 'العميل',
    'ru-RU': 'Клиент', 'it-IT': 'Cliente'
  },
  联系人: {
    'zh-TW': '聯絡人', 'en-US': 'Contact', 'es-ES': 'Contacto', 'fr-FR': 'Contact', 'de-DE': 'Kontakt',
    'ja-JP': '連絡先', 'ko-KR': '연락처', 'pt-PT': 'Contacto', 'ar-SA': 'جهة الاتصال',
    'ru-RU': 'Контакт', 'it-IT': 'Contatto'
  },
  公司: {
    'zh-TW': '公司', 'en-US': 'Company', 'es-ES': 'Empresa', 'fr-FR': 'Entreprise', 'de-DE': 'Unternehmen',
    'ja-JP': '会社', 'ko-KR': '회사', 'pt-PT': 'Empresa', 'ar-SA': 'الشركة',
    'ru-RU': 'Компания', 'it-IT': 'Azienda'
  },
  权限: {
    'zh-TW': '權限', 'en-US': 'Permissions', 'es-ES': 'Permisos', 'fr-FR': 'Droits', 'de-DE': 'Rechte',
    'ja-JP': '権限', 'ko-KR': '권한', 'pt-PT': 'Permissões', 'ar-SA': 'الصلاحيات',
    'ru-RU': 'Права', 'it-IT': 'Permessi'
  },
  语言: {
    'zh-TW': '語言', 'en-US': 'Language', 'es-ES': 'Idioma', 'fr-FR': 'Langue', 'de-DE': 'Sprache',
    'ja-JP': '言語', 'ko-KR': '언어', 'pt-PT': 'Idioma', 'ar-SA': 'اللغة',
    'ru-RU': 'Язык', 'it-IT': 'Lingua'
  },
  退出: {
    'zh-TW': '登出', 'en-US': 'Log out', 'es-ES': 'Salir', 'fr-FR': 'Déconnexion', 'de-DE': 'Abmelden',
    'ja-JP': 'ログアウト', 'ko-KR': '로그아웃', 'pt-PT': 'Sair', 'ar-SA': 'تسجيل الخروج',
    'ru-RU': 'Выйти', 'it-IT': 'Esci'
  }
}

const pack = readTranslatePack()
ensurePackShape(pack)

const referencedKeys = collectReferencedKeys()
const messages = pack.messages
const issues = []
const changed = []
const stale = []

main().catch((error) => {
  console.error(`[i18n-sync] ${error?.message || error}`)
  process.exit(1)
})

async function main() {
  const sourceMessages = await loadRuntimeZhMessages(pack)

  for (const key of referencedKeys) {
    const zhValue = resolveZhSource(key, messages['zh-CN'][key], sourceMessages[key])
    if (!zhValue) {
      issues.push({ type: 'missing-zh-source', key, locale: 'zh-CN', value: '' })
      continue
    }
    if (messages['zh-CN'][key] !== zhValue) {
      messages['zh-CN'][key] = zhValue
      changed.push({ key, locale: 'zh-CN', action: 'set-source' })
    }
    for (const locale of strictLocales) {
      if (locale === 'zh-CN') continue
      const current = messages[locale][key]
      if (isUsableTranslation(current, locale, key)) continue
      if (noTranslate) {
        issues.push({ type: 'missing-or-invalid', key, locale, value: current || '' })
        continue
      }
      const translated = await translateWithFallback(zhValue, locale, key)
      messages[locale][key] = translated
      changed.push({ key, locale, action: translated.startsWith(pendingPrefix) ? 'pending' : 'translated' })
      if (translated.startsWith(pendingPrefix)) {
        issues.push({ type: 'pending', key, locale, value: translated })
      }
    }
  }

  for (const key of Object.keys(messages['zh-CN']).sort()) {
    if (!referencedKeys.has(key)) stale.push(key)
  }

  if (prune && !checkOnly && !reportOnly) {
    for (const key of stale) {
      for (const locale of strictLocales) delete messages[locale][key]
      changed.push({ key, locale: '*', action: 'pruned-stale' })
    }
  }

  sortMessages(messages)
  pack.generated_at = new Date().toISOString()

  printSummary({ referencedKeys, changed, issues, stale })

  if (reportOnly) {
    process.exit(issues.length ? 1 : 0)
  }

  if (checkOnly) {
    if (issues.length) {
      console.error('[i18n-sync] translate.md is not synchronized. Run npm run i18n:sync.')
      process.exit(1)
    }
    console.log('[i18n-sync] translate.md is synchronized.')
    process.exit(0)
  }

  if (changed.length) {
    fs.writeFileSync(translateFile, renderTranslateMarkdown(pack), 'utf8')
    console.log(`[i18n-sync] updated ${path.relative(root, translateFile)}`)
  } else {
    console.log('[i18n-sync] no changes needed.')
  }

  if (issues.length) process.exit(1)
}

function readTranslatePack() {
  const markdown = fs.readFileSync(translateFile, 'utf8')
  const match = markdown.match(/```json\s*([\s\S]*?)```/i)
  if (!match) throw new Error('src/translate.md must contain a fenced json block.')
  return JSON.parse(match[1])
}

function ensurePackShape(value) {
  value.version = value.version || 1
  value.locales = strictLocales
  value.messages = value.messages || {}
  for (const locale of strictLocales) {
    value.messages[locale] = value.messages[locale] || {}
  }
}

function collectReferencedKeys() {
  const keys = new Set()
  for (const scanRoot of scanRoots) {
    const full = path.resolve(root, scanRoot)
    if (!fs.existsSync(full)) continue
    const stat = fs.statSync(full)
    const files = stat.isDirectory() ? walk(full) : [full]
    for (const file of files) {
      const source = stripComments(fs.readFileSync(file, 'utf8'))
      const stringKeyRe = /['"]((?:domain\.auto|common|app|login|operation|nav|navDesc|menuGroup|domain|system|platform|im|files|campaign)\.[A-Za-z0-9_.:\-\u3400-\u9FFF]+)['"]/g
      const tCallRe = /\bt\s*\(\s*['"]([^'"]+)['"]/g
      for (const match of source.matchAll(stringKeyRe)) keys.add(match[1])
      for (const match of source.matchAll(tCallRe)) keys.add(match[1])
    }
  }
  keys.delete('domain.key')
  keys.delete('domain.workspace.domain.workspace.title')
  return keys
}

function walk(dir, files = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
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

async function loadRuntimeZhMessages(currentPack) {
  bootstrapBrowserGlobals()
  const mod = await import(pathToFileURL(path.resolve(root, 'src/i18n.js')).href)
  mod.applyExternalMessages(currentPack)
  await mod.ensureLocaleMessages('zh-CN')
  mod.i18nState.locale = 'zh-CN'
  const result = {}
  for (const key of referencedKeys) {
    const value = String(mod.t(key) || '').trim()
    if (value && !isMissingLike(value) && value !== key) result[key] = value
  }
  return result
}

function bootstrapBrowserGlobals() {
  const storage = new Map()
  globalThis.localStorage = globalThis.localStorage || {
    getItem(key) { return storage.has(key) ? storage.get(key) : null },
    setItem(key, value) { storage.set(key, String(value)) },
    removeItem(key) { storage.delete(key) },
    clear() { storage.clear() }
  }
  globalThis.navigator = globalThis.navigator || { language: 'zh-CN', languages: ['zh-CN'] }
  globalThis.document = globalThis.document || {
    documentElement: { lang: '', dir: '' },
    createElement() { return { innerHTML: '', content: { firstChild: null }, cloneNode() { return this } } },
    createElementNS() { return this.createElement() }
  }
  globalThis.window = globalThis.window || globalThis
}

function resolveZhSource(key, current, runtime) {
  const candidates = [current, runtime]
  for (const candidate of candidates) {
    const text = String(candidate || '').trim()
    if (!text || text === key || isMissingLike(text) || isGenericFallback(text)) continue
    return text
  }
  return ''
}

function isUsableTranslation(value, locale, key) {
  const text = String(value || '').trim()
  if (!text || text === key) return false
  if (text.startsWith(pendingPrefix)) return false
  if (isMissingLike(text) || isGenericFallback(text)) return false
  if (/^domain\.auto\.[a-f0-9]+$/i.test(text)) return false
  if (/^(Details|Detalles|Détails|التفاصيل|Сведения|Dettagli|詳細|상세|詳情)\s+[a-f0-9]{4}$/i.test(text)) return false
  if (/(Create Details|Details name|Order finance Details|PublicOpportunity|PrivateOpportunity|BatchDeleteInvalidLateLead|Interface text|CRM interface text|UI text)/.test(text)) return false
  if (locale !== 'zh-CN' && locale !== 'zh-TW' && locale !== 'ja-JP' && /[\u3400-\u9FFF]/.test(stripProtectedSegments(text))) return false
  if (locale === 'ko-KR' && /[\u3400-\u9FFF]/.test(stripProtectedSegments(text))) return false
  if (isCompactUiKey(key) && text.length > compactLimit(key)) return false
  return true
}

function isMissingLike(value) {
  const text = String(value || '').trim()
  if (missingMarkers.has(text)) return true
  return /^(Missing translation|CRM interface text|Interface text|Text unavailable|Label)(\s+[a-f0-9]{4})?$/i.test(text)
}

function isGenericFallback(value) {
  const text = String(value || '').trim()
  return genericFallbackValues.has(text)
}

async function translateWithFallback(zhValue, locale, key) {
  const local = translateLocalTerm(zhValue, locale)
  if (local) return local
  try {
    const translated = await translateFree(zhValue, locale)
    const restored = restoreProtectedSegments(translated.text, translated.tokens).trim()
    if (isUsableTranslation(restored, locale, key)) return restored
  } catch (error) {
    // The pending marker is intentionally handled by quality gates.
  }
  return `${pendingPrefix}${zhValue}`
}

function translateLocalTerm(zhValue, locale) {
  const text = String(zhValue || '').trim()
  return localTerms[text]?.[locale] || ''
}

async function translateFree(value, locale) {
  const protectedValue = protectSegments(value)
  const url = new URL('https://translate.googleapis.com/translate_a/single')
  url.searchParams.set('client', 'gtx')
  url.searchParams.set('sl', 'zh-CN')
  url.searchParams.set('tl', targetLangMap[locale] || locale)
  url.searchParams.set('dt', 't')
  url.searchParams.set('q', protectedValue.text)
  const response = await fetch(url, {
    headers: { 'User-Agent': 'ai-crm-i18n-sync/1.0' },
    signal: AbortSignal.timeout(15000)
  })
  if (!response.ok) throw new Error(`translate failed: ${response.status}`)
  const body = await response.json()
  const translated = Array.isArray(body?.[0])
    ? body[0].map((item) => item?.[0] || '').join('')
    : ''
  if (!translated) throw new Error('empty translation response')
  return { text: translated, tokens: protectedValue.tokens }
}

function protectSegments(value) {
  const tokens = []
  const patterns = [
    /\$\{[^}]*\}/g,
    /\{[A-Za-z0-9_]+\}/g,
    /https?:\/\/\S+/g,
    /\b[A-Z]+_[A-Z0-9_]+\b/g,
    /\b[a-z]+:[a-z_:-]+\b/g,
    /\b[A-Z]{2,}\b/g,
    /\b[\w.+-]+@[\w.-]+\.[A-Za-z]{2,}\b/g
  ]
  let text = String(value || '')
  for (const pattern of patterns) {
    text = text.replace(pattern, (match) => {
      if (/^[A-Z]{2,}$/.test(match) && !protectedAcronyms.has(match)) return match
      const token = `__I18N_${tokens.length}__`
      tokens.push({ token, value: match })
      return token
    })
  }
  return { text, tokens }
}

function restoreProtectedSegments(value, tokens) {
  let text = String(value || '')
  for (const item of tokens) {
    const tokenPattern = new RegExp(item.token.replace(/[.*+?^${}()|[\]\\]/g, '\\$&').replace(/_/g, '\\s*_\\s*'), 'g')
    text = text.replace(tokenPattern, item.value)
  }
  return text
}

function stripProtectedSegments(value = '') {
  return String(value)
    .replace(/https?:\/\/\S+/g, '')
    .replace(/\$\{[^}]*\}/g, '')
    .replace(/\{[^}]*\}/g, '')
    .replace(/\bR_[A-Z0-9_]+\b/g, '')
    .replace(/\b[A-Z]{2,}\b/g, '')
    .replace(/\b[A-Za-z]+(?:\.[A-Za-z0-9]+)+\b/g, '')
}

function isCompactUiKey(key) {
  return /\.(button|action|label|title|status|type|name|filter|column|tab|tag|menu|nav|header|empty)$/.test(key)
}

function compactLimit(key) {
  if (/\.(button|action|label|status|type|column|tab|tag|menu|nav|header)$/.test(key)) return 56
  return 80
}

function sortMessages(localeMessages) {
  for (const locale of strictLocales) {
    localeMessages[locale] = Object.fromEntries(Object.entries(localeMessages[locale]).sort(([a], [b]) => a.localeCompare(b)))
  }
}

function renderTranslateMarkdown(value) {
  return `# AI CRM 翻译词库

本文件是前端运行时翻译词库。前端通过 \`src/i18n.js\` 读取下方 JSON 数据块，并将其作为界面文案的最终覆盖来源。

维护规则：
- 只维护系统界面文案，不翻译用户录入的业务数据。
- \`zh-CN\` 是唯一原文；其他语言由 \`npm run i18n:sync\` 根据中文原文自动补齐。
- 新增界面文案必须使用 \`t(key)\`，提交前运行 \`npm run i18n:sync\`。
- CI 使用 \`npm run i18n:sync:check\`、\`npm run i18n:scan:ci\`、\`npm run i18n:runtime:ci\` 拦截缺失、串语言和占位翻译。
- 按钮、菜单、表格列、状态标签使用短文案，避免撑破布局。
- 不允许使用 Label、Interface text、Details 这类泛化兜底词。

\`\`\`json
${JSON.stringify(value, null, 2)}
\`\`\`
`
}

function printSummary({ referencedKeys, changed, issues, stale }) {
  console.log(`[i18n-sync] referenced keys: ${referencedKeys.size}`)
  console.log(`[i18n-sync] changed entries: ${changed.length}`)
  console.log(`[i18n-sync] issues: ${issues.length}`)
  console.log(`[i18n-sync] stale keys: ${stale.length}`)
  for (const item of issues.slice(0, 80)) {
    console.log(`- ${item.type}: ${item.locale} ${item.key}${item.value ? ` -> ${item.value}` : ''}`)
  }
  if (stale.length) {
    console.log('[i18n-sync] stale keys are reported only. Use --prune to remove them intentionally.')
    for (const key of stale.slice(0, 20)) console.log(`- stale: ${key}`)
  }
}
