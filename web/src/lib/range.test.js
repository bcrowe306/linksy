import { describe, it, expect, vi, afterEach } from 'vitest'
import { rangeParams, rangeFromQuery } from './range'

afterEach(() => vi.useRealTimers())

describe('rangeParams', () => {
  it('turns presets into inclusive local dates', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 9, 25, 15, 0))
    expect(rangeParams({ preset: '7d' }, 'UTC')).toEqual({ from: '2026-10-19', to: '2026-10-25', tz: 'UTC' })
    expect(rangeParams({ preset: 'today' }, 'UTC')).toEqual({ from: '2026-10-25', to: '2026-10-25', tz: 'UTC' })
    expect(rangeParams({ preset: '30d' }, 'UTC').from).toBe('2026-09-26')
  })
  it('passes custom ranges through', () => {
    expect(rangeParams({ preset: 'custom', from: '2026-01-01', to: '2026-01-31' }, 'Europe/Paris')).toEqual({ from: '2026-01-01', to: '2026-01-31', tz: 'Europe/Paris' })
  })
})

describe('rangeFromQuery', () => {
  it('reads shareable URLs and falls back to 30 days', () => {
    expect(rangeFromQuery({ range: '90d' })).toEqual({ preset: '90d' })
    expect(rangeFromQuery({ range: 'custom', from: 'a', to: 'b' })).toEqual({ preset: 'custom', from: 'a', to: 'b' })
    expect(rangeFromQuery({ range: 'bogus' })).toEqual({ preset: '30d' })
  })
})
