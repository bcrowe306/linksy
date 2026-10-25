import { describe, it, expect } from 'vitest'
import { QR_TYPES, defaultsFor, rawDataFor, parseRawData, isDynamicFor, validateContent } from './qrContent'

describe('QR content', () => {
  it('covers every type the server supports', () => {
    expect(QR_TYPES.map((t) => t.id).sort()).toEqual(['app', 'document', 'email', 'phone', 'sms', 'text', 'vcard', 'website', 'wifi'])
  })
  it('serializes text as plain text and other types as JSON', () => {
    expect(rawDataFor('text', { text: 'Hello' })).toBe('Hello')
    expect(rawDataFor('website', {})).toBe('')
    expect(JSON.parse(rawDataFor('wifi', { ssid: 'Cafe', password: 'x' }))).toMatchObject({ ssid: 'Cafe' })
  })
  it('parses stored raw data over the defaults', () => {
    expect(parseRawData('wifi', '{"ssid":"Cafe"}')).toEqual({ ...defaultsFor('wifi'), ssid: 'Cafe' })
    expect(parseRawData('text', 'Hi')).toEqual({ text: 'Hi' })
    expect(parseRawData('vcard', '{"address":"1 Main St"}').street).toBe('1 Main St') // legacy field
    expect(parseRawData('email', 'not json')).toEqual(defaultsFor('email'))
  })
  it('mirrors which types can be dynamic', () => {
    expect(isDynamicFor('document', false)).toBe(true)
    expect(isDynamicFor('wifi', true)).toBe(false)
    expect(isDynamicFor('website', true)).toBe(true)
    expect(isDynamicFor('website', false)).toBe(false)
  })
  it('explains what is missing', () => {
    expect(validateContent({ type: 'website', target_url: 'example.com' })).toMatch(/https/)
    expect(validateContent({ type: 'website', target_url: 'https://example.com' })).toBe('')
    expect(validateContent({ type: 'wifi', data: { ssid: '' } })).toMatch(/SSID/)
    expect(validateContent({ type: 'email', data: { email: 'a@b.co' } })).toBe('')
    expect(validateContent({ type: 'app', data: { web_url: 'ftp://x' } })).toMatch(/https/)
    expect(validateContent({ type: 'phone', data: { phone: '+1 (555) 010' } })).toBe('')
  })
})
