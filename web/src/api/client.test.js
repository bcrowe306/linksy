import { describe, it, expect, vi, beforeEach } from 'vitest'
import { request, api, onUnauthorized, onPaymentRequired, ApiError } from './client'

function respond(status, body) {
  return vi.fn().mockResolvedValue(new Response(body === undefined ? '' : JSON.stringify(body), { status }))
}

describe('API client', () => {
  beforeEach(() => {
    onUnauthorized(null)
    onPaymentRequired(null)
  })

  it('sends JSON with the session cookie and parses the reply', async () => {
    globalThis.fetch = respond(200, { ok: true })
    expect(await api.billing()).toEqual({ ok: true })
    const [url, init] = fetch.mock.calls[0]
    expect(url).toBe('/api/v1/billing')
    expect(init.credentials).toBe('same-origin')

    globalThis.fetch = respond(200, { url: 'https://checkout.stripe.com/x' })
    await api.startCheckout('year', 'LAUNCH50')
    expect(JSON.parse(fetch.mock.calls[0][1].body)).toEqual({ interval: 'year', code: 'LAUNCH50' })
    expect(fetch.mock.calls[0][1].headers['Content-Type']).toBe('application/json')
  })

  it('drops empty query parameters', async () => {
    globalThis.fetch = respond(200, { users: [] })
    await api.admin.users('', 0)
    expect(fetch.mock.calls[0][0]).toBe('/api/v1/admin/users?offset=0&limit=50')
  })

  it('throws ApiError with the server message and body', async () => {
    globalThis.fetch = respond(403, { error: 'Confirm it', code: 'admin_reauth' })
    const err = await request('GET', '/api/v1/admin/metrics').catch((e) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err.message).toBe('Confirm it')
    expect(err.status).toBe(403)
    expect(err.body.code).toBe('admin_reauth')
  })

  it('signals an ended session (401) and a lapsed plan (402)', async () => {
    const unauthorized = vi.fn()
    const lapsed = vi.fn()
    onUnauthorized(unauthorized)
    onPaymentRequired(lapsed)
    globalThis.fetch = respond(401, { error: 'no' })
    await request('GET', '/api/v1/links').catch(() => {})
    expect(unauthorized).toHaveBeenCalled()
    globalThis.fetch = respond(402, { error: 'Subscribe', code: 'subscription_required' })
    await request('POST', '/api/v1/links', { body: {} }).catch(() => {})
    expect(lapsed).toHaveBeenCalledTimes(1)
  })
})
