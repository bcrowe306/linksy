import { describe, it, expect } from 'vitest'
import { compact, delta, niceTicks, relativeTime } from './format'

describe('compact', () => {
  it('abbreviates large numbers', () => {
    expect(compact(950)).toBe('950')
    expect(compact(12_345)).toBe('12.3K')
    expect(compact(250_000)).toBe('250K')
    expect(compact(1_500_000)).toBe('1.5M')
    expect(compact(null)).toBe('–')
  })
})

describe('delta', () => {
  it('compares with the previous period', () => {
    expect(delta(150, 100)).toMatchObject({ label: '+50%', direction: 'up' })
    expect(delta(50, 100)).toMatchObject({ label: '-50%', direction: 'down' })
    expect(delta(100, 100)).toMatchObject({ label: '0%', direction: 'flat' })
  })
  it('treats growth from zero as new', () => {
    expect(delta(5, 0)).toMatchObject({ label: 'New', direction: 'up' })
    expect(delta(0, 0)).toMatchObject({ direction: 'flat' })
  })
})

describe('niceTicks', () => {
  it('rounds the axis to friendly steps', () => {
    expect(niceTicks(73)).toEqual([0, 20, 40, 60, 80])
    expect(niceTicks(0)).toEqual([0, 1, 2, 3, 4])
    expect(niceTicks(3)).toEqual([0, 1, 2, 3, 4]) // never fractional steps
  })
})

describe('relativeTime', () => {
  it('describes recent times', () => {
    const ago = (s) => new Date(Date.now() - s * 1000).toISOString()
    expect(relativeTime(ago(1))).toBe('just now')
    expect(relativeTime(ago(30))).toBe('30s ago')
    expect(relativeTime(ago(600))).toBe('10m ago')
    expect(relativeTime(ago(7200))).toBe('2h ago')
  })
})
