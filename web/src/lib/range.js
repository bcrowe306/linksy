// Converts a date-range selection into API query parameters in the viewer's zone.
export const browserTZ = Intl.DateTimeFormat().resolvedOptions().timeZone

const ymd = (d) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`

export function rangeParams(range, tz = browserTZ) {
  const now = new Date()
  if (range.preset === 'custom' && range.from && range.to) return { from: range.from, to: range.to, tz }
  if (range.preset === 'today') return { from: ymd(now), to: ymd(now), tz }
  const days = { '7d': 7, '30d': 30, '90d': 90, '12m': 365 }[range.preset] || 7
  const start = new Date(now)
  start.setDate(start.getDate() - (days - 1))
  return { from: ymd(start), to: ymd(now), tz }
}

export function rangeFromQuery(q) {
  if (q.range === 'custom' && q.from && q.to) return { preset: 'custom', from: q.from, to: q.to }
  return { preset: ['today', '7d', '30d', '90d', '12m'].includes(q.range) ? q.range : '30d' }
}
