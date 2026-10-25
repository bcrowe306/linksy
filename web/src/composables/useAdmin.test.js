import { describe, it, expect, vi } from 'vitest'
import { adminState, useAdminCall, money } from './useAdmin'

vi.mock('./useToast', () => ({ useToast: () => ({ error: vi.fn(), success: vi.fn() }) }))

describe('admin console helpers', () => {
  it('returns to the confirm screen when the admin session expires', async () => {
    adminState.needsReauth = false
    const call = useAdminCall()
    const err = Object.assign(new Error('Confirm it'), { status: 403, body: { code: 'admin_reauth' } })
    await expect(call(() => Promise.reject(err))).rejects.toBe(err)
    expect(adminState.needsReauth).toBe(true)
  })
  it('passes results through', async () => {
    expect(await useAdminCall()(() => Promise.resolve(42))).toBe(42)
  })
  it('formats cents as dollars', () => {
    expect(money(899)).toBe('$8.99')
    expect(money(6000)).toBe('$60')
  })
})
