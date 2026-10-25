import { describe, it, expect } from 'vitest'
import { generatorPayload } from './generator'

function form(html) {
  const f = document.createElement('form')
  f.innerHTML = html
  return f
}

const style = '<input data-style="fg_color" value="#4f46e5"><select data-style="dot_style"><option value="dots" selected>Dots</option></select>'

describe('free QR generator payload', () => {
  it('sends text codes as the text itself', () => {
    const body = generatorPayload(form('<textarea data-raw="text">Hello there</textarea>' + style), 'text')
    expect(body.raw_data).toBe('Hello there')
    expect(body.fg_color).toBe('#4f46e5')
    expect(body.dot_style).toBe('dots')
  })
  it('sends structured types as JSON, including checkboxes', () => {
    const body = generatorPayload(form('<input data-raw="ssid" value="Cafe"><input type="checkbox" data-raw="hidden" checked>'), 'wifi')
    expect(JSON.parse(body.raw_data)).toEqual({ ssid: 'Cafe', hidden: true })
  })
  it('sends the website URL as the target', () => {
    const body = generatorPayload(form('<input data-target value="https://example.com">'), 'website')
    expect(body.target_url).toBe('https://example.com')
  })
})
