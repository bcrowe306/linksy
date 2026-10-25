// In-app guides, built from ../docs/guides/*.md (also served as public pages).
import { marked } from 'marked'

const files = import.meta.glob('../../../docs/guides/*.md', { query: '?raw', import: 'default', eager: true })

function parse(raw, path) {
  const slug = path.split('/').pop().replace(/\.md$/, '')
  const meta = {}
  let body = raw
  const fm = raw.match(/^---\n([\s\S]*?)\n---\n/)
  if (fm) {
    fm[1].split('\n').forEach((line) => {
      const i = line.indexOf(':')
      if (i > 0) meta[line.slice(0, i).trim()] = line.slice(i + 1).trim()
    })
    body = raw.slice(fm[0].length)
  }
  // "## Title {#anchor}" → explicit heading ids for deep links
  const html = marked
    .parse(body.replace(/^(#{1,6} .*?)\s*\{#([\w-]+)\}\s*$/gm, (_, h, id) => `${h} <!--id:${id}-->`))
    .replace(/<(h[1-6])>(.*?) <!--id:([\w-]+)-->/g, '<$1 id="$3">$2')
  const text = body.replace(/[#*`>|_-]/g, ' ').toLowerCase()
  return { slug, title: meta.title || slug, description: meta.description || '', order: Number(meta.order || 99), html, text }
}

export const guides = Object.entries(files)
  .map(([path, raw]) => parse(raw, path))
  .sort((a, b) => a.order - b.order)

export const guideBySlug = (slug) => guides.find((g) => g.slug === slug)

export function searchGuides(q) {
  const terms = q.toLowerCase().split(/\s+/).filter(Boolean)
  if (!terms.length) return guides
  return guides
    .map((g) => ({ g, score: terms.reduce((s, t) => s + (g.title.toLowerCase().includes(t) ? 5 : 0) + (g.text.includes(t) ? 1 : 0), 0) }))
    .filter((x) => x.score > 0 && terms.every((t) => x.g.text.includes(t) || x.g.title.toLowerCase().includes(t)))
    .sort((a, b) => b.score - a.score)
    .map((x) => x.g)
}
