import { reactive } from 'vue'

// Global help state so any screen can open a specific guide section.
export const helpState = reactive({ open: false, slug: '', anchor: '', shortcuts: false })

export function useHelp() {
  return {
    open(slug = '', anchor = '') {
      Object.assign(helpState, { open: true, slug, anchor })
    },
    close() {
      helpState.open = false
    },
    shortcuts() {
      helpState.shortcuts = true
    }
  }
}
