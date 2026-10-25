import { onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { helpState } from './useHelp'
import { useThemeStore } from '../stores/theme'
import { useAuthStore } from '../stores/auth'

export const SHORTCUT_GROUPS = [
  {
    title: 'General',
    items: [
      { label: 'Search', keys: ['/'] },
      { label: 'Keyboard shortcuts', keys: ['?'] },
      { label: 'Help & quick docs', keys: ['H'] },
      { label: 'Toggle light / dark theme', keys: ['T'] },
      { label: 'Close dialog or menu', keys: ['Esc'] }
    ]
  },
  {
    title: 'Create',
    items: [
      { label: 'New short link', keys: ['N', 'L'] },
      { label: 'New QR code', keys: ['N', 'Q'] },
      { label: 'Upload document', keys: ['N', 'D'] },
      { label: 'Open QR Studio', keys: ['N', 'S'] }
    ]
  },
  {
    title: 'Go to',
    items: [
      { label: 'Dashboard', keys: ['G', 'H'] },
      { label: 'Short links', keys: ['G', 'L'] },
      { label: 'QR codes', keys: ['G', 'Q'] },
      { label: 'Documents', keys: ['G', 'D'] },
      { label: 'Analytics', keys: ['G', 'A'] },
      { label: 'Team & settings', keys: ['G', 'S'] },
      { label: 'Profile', keys: ['G', 'P'] }
    ]
  }
]

const GOTO = { h: '/', l: '/links', q: '/qr-codes', d: '/documents', a: '/analytics', s: '/settings', p: '/profile' }

// useShortcuts wires global single-key and two-key ("G then L") shortcuts.
// Keys are ignored while typing in a field or when a modifier is held.
export function useShortcuts(openCreate) {
  const router = useRouter()
  const theme = useThemeStore()
  const auth = useAuthStore()
  let prefix = ''
  let timer = null

  function typing(e) {
    const el = e.target
    return el && (['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName) || el.isContentEditable)
  }

  function onKey(e) {
    if (e.metaKey || e.ctrlKey || e.altKey || typing(e) || document.querySelector('[role=dialog][data-open=true]')) return
    const k = e.key.toLowerCase()

    if (prefix) {
      const p = prefix
      prefix = ''
      clearTimeout(timer)
      if (p === 'g' && GOTO[k]) {
        e.preventDefault()
        router.push(GOTO[k])
      } else if (p === 'n' && auth.can('assets:edit')) {
        e.preventDefault()
        if (k === 'l') openCreate('link')
        else if (k === 'q') openCreate('qr')
        else if (k === 'd') openCreate('doc')
        else if (k === 's') router.push('/qr-studio')
      }
      return
    }
    if (k === 'g' || k === 'n') {
      prefix = k
      timer = setTimeout(() => (prefix = ''), 1200)
      return
    }
    if (e.key === '?') {
      e.preventDefault()
      helpState.open = false
      helpState.shortcuts = true
    } else if (k === 'h') {
      helpState.open = !helpState.open
    } else if (k === 't') {
      theme.setTheme(theme.effectiveTheme === 'dark' ? 'light' : 'dark')
    }
  }

  onMounted(() => document.addEventListener('keydown', onKey))
  onBeforeUnmount(() => document.removeEventListener('keydown', onKey))
}
