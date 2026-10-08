#!/usr/bin/env node
import fs from 'fs'
import path from 'path'

const dir = path.join(process.cwd(), 'src/views/domains')
const files = fs.readdirSync(dir).filter((f) => f.endsWith('.vue'))
const rows = []
for (const f of files) {
  const p = path.join(dir, f)
  const c = fs.readFileSync(p, 'utf8')
  const m = c.match(/<template>[\s\S]*?<\/template>/)
  if (!m) continue
  const tpl = m[0]
  const lines = tpl.split('\n')
  let n = 0
  for (const line of lines) {
    if (/[\u3400-\u9FFF]/.test(line)) n += 1
  }
  rows.push({ file: f, lines: n })
}
rows.sort((a, b) => b.lines - a.lines)
const total = rows.reduce((s, r) => s + r.lines, 0)
console.log(`template lines with Chinese: ${total}`)
for (const r of rows) {
  if (!r.lines) continue
  console.log(`${r.file}: ${r.lines}`)
}
