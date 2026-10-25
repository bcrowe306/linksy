import { describe, it, expect } from 'vitest'
import { blankLinkForm, linkToForm, formToPayload } from './linkForm'

describe('link form', () => {
  it('only sends the password when it changed', () => {
    const form = linkToForm({ target_url: 'https://a.example', slug: 's', has_password: true, utm_source: 'news' })
    expect(form.utm_source).toBe('news')
    expect('password' in formToPayload(form)).toBe(false) // unchanged
    expect(formToPayload({ ...form, password: '' }).password).toBe('') // remove
    expect(formToPayload({ ...form, password: 'secret' }).password).toBe('secret')
    expect('password' in formToPayload({ ...form, password: 'change' })).toBe(false) // editing, nothing typed yet
  })
  it('converts the local expiry to an ISO timestamp', () => {
    const body = formToPayload({ ...blankLinkForm(), target_url: 'https://a.example', expires_local: '2026-12-31T23:30' })
    expect(body.expires_at).toBe(new Date('2026-12-31T23:30').toISOString())
    expect(formToPayload(blankLinkForm()).expires_at).toBeNull()
  })
  it('round-trips the expiry into the datetime-local format', () => {
    const iso = new Date(2026, 0, 2, 3, 4).toISOString()
    expect(linkToForm({ expires_at: iso }).expires_local).toBe('2026-01-02T03:04')
  })
})
