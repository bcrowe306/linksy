import { reactive } from 'vue'

// Promise-based replacement for window.confirm(), rendered by <ConfirmHost />.
export const confirmState = reactive({
  open: false,
  title: '',
  message: '',
  confirmText: 'Confirm',
  cancelText: 'Cancel',
  danger: false,
  requireText: '',
  resolve: null
})

export function useConfirm() {
  return function confirm(opts) {
    const o = typeof opts === 'string' ? { message: opts } : opts
    Object.assign(confirmState, {
      open: true,
      title: o.title || 'Are you sure?',
      message: o.message || '',
      confirmText: o.confirmText || 'Confirm',
      cancelText: o.cancelText || 'Cancel',
      danger: !!o.danger,
      requireText: o.requireText || ''
    })
    return new Promise((resolve) => {
      confirmState.resolve = resolve
    })
  }
}

export function settleConfirm(result) {
  confirmState.open = false
  confirmState.resolve?.(result)
  confirmState.resolve = null
}
