import { reactive } from 'vue'

// Global toast queue rendered by <ToastHost />.
const state = reactive({ items: [] })
let seq = 0

function push(type, message, opts = {}) {
  const id = ++seq
  const item = { id, type, message, title: opts.title || '', action: opts.action || null }
  state.items.push(item)
  const ttl = opts.duration ?? (type === 'error' ? 6000 : 3500)
  if (ttl > 0) setTimeout(() => dismiss(id), ttl)
  return id
}

export function dismiss(id) {
  const i = state.items.findIndex((t) => t.id === id)
  if (i !== -1) state.items.splice(i, 1)
}

export const toastState = state

export function useToast() {
  return {
    success: (msg, opts) => push('success', msg, opts),
    error: (msg, opts) => push('error', msg, opts),
    info: (msg, opts) => push('info', msg, opts),
    warning: (msg, opts) => push('warning', msg, opts),
    dismiss
  }
}

// Copies text and confirms with a toast.
export async function copyToClipboard(text, label = 'Copied to clipboard') {
  try {
    await navigator.clipboard.writeText(text)
    push('success', label)
    return true
  } catch {
    push('error', 'Clipboard access was blocked by the browser')
    return false
  }
}
