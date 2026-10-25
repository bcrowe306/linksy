import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, RouterLinkStub } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import Toggle from './ui/Toggle.vue'
import BillingBanner from './BillingBanner.vue'
import { useAuthStore } from '../stores/auth'

vi.mock('../api/client', () => ({
  api: { authConfig: vi.fn().mockResolvedValue({ enabled: true, providers: [] }), me: vi.fn() },
  onUnauthorized: vi.fn(),
  onPaymentRequired: vi.fn()
}))

describe('Toggle', () => {
  it('always exposes aria-checked and emits the new value', async () => {
    const w = mount(Toggle, { props: { modelValue: false } })
    expect(w.attributes('role')).toBe('switch')
    expect(w.attributes('aria-checked')).toBe('false')
    await w.trigger('click')
    expect(w.emitted('update:modelValue')[0]).toEqual([true])
    await w.setProps({ modelValue: true })
    expect(w.attributes('aria-checked')).toBe('true')
  })
})

describe('BillingBanner', () => {
  let auth
  const mountBanner = () => mount(BillingBanner, { global: { stubs: { RouterLink: RouterLinkStub } } })
  beforeEach(() => {
    setActivePinia(createPinia())
    auth = useAuthStore()
  })

  it('stays hidden when billing is off or the trial has time left', () => {
    auth.billing = { enabled: false }
    expect(mountBanner().find('[data-testid=billing-banner]').exists()).toBe(false)
    auth.billing = { enabled: true, state: 'trial', trial_days_left: 6, can_manage: true }
    expect(mountBanner().find('[data-testid=billing-banner]').exists()).toBe(false)
  })

  it('warns in the last 3 trial days, linking owners to billing', () => {
    auth.billing = { enabled: true, state: 'trial', trial_days_left: 1, can_manage: true }
    const w = mountBanner()
    expect(w.text()).toContain('ends in 1 day')
    expect(w.findComponent(RouterLinkStub).props('to')).toEqual({ path: '/settings', query: { tab: 'billing' } })
  })

  it('tells non-owners to ask the owner instead of linking', () => {
    auth.billing = { enabled: true, state: 'expired', read_only: true, can_manage: false }
    const w = mountBanner()
    expect(w.text()).toContain('read-only')
    expect(w.text()).toContain('Ask the workspace owner')
    expect(w.findComponent(RouterLinkStub).exists()).toBe(false)
  })

  it('explains a suspension even without billing', () => {
    auth.billing = { enabled: false, suspended: true, read_only: true }
    expect(mountBanner().text()).toContain('suspended')
  })

  it('flags failed payments', () => {
    auth.billing = { enabled: true, state: 'past_due', can_manage: true }
    expect(mountBanner().text()).toContain('payment failed')
  })
})
