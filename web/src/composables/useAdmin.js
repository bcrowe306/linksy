import { reactive } from 'vue'
import { useToast } from './useToast'

// Shared admin-console state: any admin call answered with "admin_reauth"
// sends the console back to the confirm-identity screen.
export const adminState = reactive({ needsReauth: false, elevatedUntil: null })

export function useAdminCall() {
  const toast = useToast()
  return async function call(fn, { quiet = false } = {}) {
    try {
      return await fn()
    } catch (err) {
      if (err.status === 403 && err.body?.code === 'admin_reauth') {
        adminState.needsReauth = true
      } else if (!quiet) {
        toast.error(err.message)
      }
      throw err
    }
  }
}

// Plan state → badge class and label (shared by workspace views).
export const PLAN_BADGES = {
  trial: ['badge-brand', 'Trial'],
  trialing: ['badge-success', 'Subscribed'],
  active: ['badge-success', 'Active'],
  past_due: ['badge-warning', 'Past due'],
  comped: ['badge-success', 'Comped'],
  canceled: ['badge-danger', 'Canceled'],
  expired: ['badge-neutral', 'Trial ended']
}

export const shortDate = (d) => (d ? new Date(d).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' }) : '—')
export const money = (cents) => `$${(cents / 100).toLocaleString(undefined, { minimumFractionDigits: cents % 100 ? 2 : 0, maximumFractionDigits: 2 })}`
