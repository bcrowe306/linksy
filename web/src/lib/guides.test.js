import { describe, it, expect } from 'vitest'
import { guides, guideBySlug, searchGuides } from './guides'

describe('guides', () => {
  it('loads every guide with front matter, in order', () => {
    expect(guides.length).toBeGreaterThanOrEqual(9)
    expect(guides[0].slug).toBe('getting-started')
    guides.forEach((g) => {
      expect(g.title).not.toBe(g.slug)
      expect(g.description.length).toBeGreaterThan(20)
    })
  })
  it('renders explicit heading anchors for deep links', () => {
    const html = guides.map((g) => g.html).join('')
    expect(html).toMatch(/<h2 id="[\w-]+">/)
  })
  it('finds guides by slug and search terms', () => {
    expect(guideBySlug('qr-types')?.title).toMatch(/QR code types/i)
    expect(searchGuides('password').some((g) => g.slug === 'short-links')).toBe(true)
  })
})
