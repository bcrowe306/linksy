// Number, delta and time formatting shared by the analytics views.

export function compact(n) {
  if (n === null || n === undefined) return '–'
  const abs = Math.abs(n)
  if (abs >= 1e6) return `${(n / 1e6).toFixed(abs >= 1e7 ? 0 : 1)}M`
  if (abs >= 1e4) return `${(n / 1e3).toFixed(abs >= 1e5 ? 0 : 1)}K`
  return n.toLocaleString()
}

// delta returns { pct, label, direction } comparing cur to prev.
export function delta(cur, prev) {
  if (!prev) return cur ? { pct: null, label: 'New', direction: 'up' } : { pct: 0, label: '0%', direction: 'flat' }
  const pct = ((cur - prev) / prev) * 100
  const r = Math.round(pct)
  return { pct, label: `${r > 0 ? '+' : ''}${r}%`, direction: r > 0 ? 'up' : r < 0 ? 'down' : 'flat' }
}

export function bucketLabel(iso, granularity, opts = {}) {
  const d = new Date(iso)
  if (granularity === 'hour') return d.toLocaleTimeString(undefined, { hour: 'numeric', ...opts })
  if (granularity === 'week') return `Wk of ${d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}`
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', ...opts })
}

export function tooltipDate(iso, granularity) {
  const d = new Date(iso)
  if (granularity === 'hour') return d.toLocaleString(undefined, { weekday: 'short', month: 'short', day: 'numeric', hour: 'numeric' })
  if (granularity === 'week') return `Week of ${d.toLocaleDateString(undefined, { month: 'long', day: 'numeric', year: 'numeric' })}`
  return d.toLocaleDateString(undefined, { weekday: 'short', month: 'long', day: 'numeric', year: 'numeric' })
}

// niceMax rounds an axis maximum up to 1/2/2.5/5×10^n and returns 4 ticks.
export function niceTicks(max, count = 4) {
  if (max <= 0) return [0, 1, 2, 3, 4].slice(0, count + 1)
  const raw = max / count
  const pow = 10 ** Math.floor(Math.log10(raw))
  const step = [1, 2, 2.5, 5, 10].map((m) => m * pow).find((s) => s >= raw)
  const niceStep = Math.max(1, step)
  return Array.from({ length: count + 1 }, (_, i) => Math.round(i * niceStep))
}

export function relativeTime(iso) {
  const s = (Date.now() - new Date(iso)) / 1000
  if (s < 5) return 'just now'
  if (s < 60) return `${Math.floor(s)}s ago`
  if (s < 3600) return `${Math.floor(s / 60)}m ago`
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`
  return new Date(iso).toLocaleDateString()
}
